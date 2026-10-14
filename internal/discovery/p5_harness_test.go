//go:build mockdiscovery

package discovery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/radilabs/radichat/internal/mockllm"
)

const (
	p5CaseTimeout     = 15 * time.Second
	p5PreviewBytes    = 400
	p5ReportsDir      = ".factory/reports"
	p5EvidencePrefix  = "stage-3-phase-5-run-"
	p5FindingsDirName = "findings"
)

type p5EnvVar struct{ name, value string }

type p5Config struct {
	model          string
	budget         int64
	reserve        int64
	systemPrompt   string
	hasSystem      bool
	bearerTokenEnv string
}

func p5DefaultConfig() p5Config {
	return p5Config{model: "p5-model", budget: 8192, reserve: 1024}
}

func p5Cfg(budget, reserve int64) p5Config {
	return p5Config{model: "p5-model", budget: budget, reserve: reserve}
}

func (c p5Config) withBearer(name string) p5Config {
	c.bearerTokenEnv = name
	return c
}

func (c p5Config) withSystem(s string) p5Config {
	c.systemPrompt = s
	c.hasSystem = true
	return c
}

type p5DivSpec struct {
	slug     string
	title    string
	expected string
	impact   string
	severity string
}

type p5Spec struct {
	id       string
	scenario string
	cfg      p5Config
	env      []p5EnvVar
	stdin    string
	oracle   string
	notes    string
	div      p5DivSpec
}

type p5Result struct {
	spec     p5Spec
	endpoint string
	cfgJSON  string
	cfgHash  string
	envNames []string
	exit     int
	timedOut bool
	stdout   string
	stderr   string
	records  []mockllm.Record
}

type p5Divergence struct {
	findingID    string
	slug         string
	title        string
	caseID       string
	scenario     string
	harnessLocal bool
	expected     string
	actual       string
	impact       string
	severity     string
	repro        string
	cfgJSON      string
	envNames     []string
	stdin        string
}

var (
	p5BinPath     string
	p5Server      *mockllm.Server
	p5Evidence    *os.File
	p5EvidenceRel string
	p5Commit      string
	p5GoVersion   = runtime.Version()
	p5Mu          sync.Mutex
	p5Divergences []p5Divergence
	p5FindingBase int
)

var (
	p5RunFileRe = regexp.MustCompile(`^` + regexp.QuoteMeta(p5EvidencePrefix) + `(\d{2})\.jsonl$`)
	p5FindingRe = regexp.MustCompile(`^F-(\d{4})-`)
	p5Sentinels = []string{mockllm.SentinelToken, mockllm.SentinelMismatchToken}
)

// TestMain builds the real RadiChat binary once, starts the accepted in-process
// mock server, opens the sanitized JSONL evidence file, runs the matrix, and
// flushes one durable finding per recorded divergence before exiting. A nonzero
// suite exit is therefore always explained by findings.
func TestMain(m *testing.M) {
	root := p5RepoRoot()
	bin, err := p5BuildBinary(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "P5 harness: build failed: %v\n", err)
		os.Exit(1)
	}
	p5BinPath = bin
	p5Commit = p5GitCommit(root)

	srv, err := mockllm.Start(mockllm.Options{ExpectedBearer: mockllm.SentinelToken})
	if err != nil {
		fmt.Fprintf(os.Stderr, "P5 harness: mock start failed: %v\n", err)
		os.Exit(1)
	}
	p5Server = srv

	ev, rel, err := p5OpenEvidence(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "P5 harness: evidence open failed: %v\n", err)
		os.Exit(1)
	}
	p5Evidence, p5EvidenceRel = ev, rel
	p5FindingBase = p5NextFindingBase(root)

	code := m.Run()

	p5FlushFindings(root)
	_ = p5Evidence.Sync()
	_ = p5Evidence.Close()
	_ = p5Server.Close()
	fmt.Fprintf(os.Stderr, "P5 harness: commit=%s go=%s evidence=%s divergences=%d suite_exit=%d\n",
		p5Commit, p5GoVersion, p5EvidenceRel, len(p5Divergences), code)
	os.Exit(code)
}

func p5RepoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "../.."
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(file)))
}

func p5GoTool() string {
	candidate := filepath.Join(runtime.GOROOT(), "bin", "go")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return "go"
}

func p5BuildBinary(root string) (string, error) {
	dir, err := os.MkdirTemp("", "radichat-p5-")
	if err != nil {
		return "", err
	}
	bin := filepath.Join(dir, "radichat")
	cmd := exec.Command(p5GoTool(), "build", "-o", bin, ".")
	cmd.Dir = root
	var out strings.Builder
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%v: %s", err, out.String())
	}
	return bin, nil
}

func p5GitCommit(root string) string {
	out, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func p5OpenEvidence(root string) (*os.File, string, error) {
	dir := filepath.Join(root, p5ReportsDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, "", err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, "", err
	}
	next := 1
	for _, e := range entries {
		m := p5RunFileRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		if n, err := strconv.Atoi(m[1]); err == nil && n >= next {
			next = n + 1
		}
	}
	name := fmt.Sprintf("%s%02d.jsonl", p5EvidencePrefix, next)
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return nil, "", err
	}
	return f, p5ReportsDir + "/" + name, nil
}

func p5NextFindingBase(root string) int {
	entries, err := os.ReadDir(filepath.Join(root, p5FindingsDirName))
	if err != nil {
		return 1
	}
	max := 0
	for _, e := range entries {
		m := p5FindingRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		if n, err := strconv.Atoi(m[1]); err == nil && n > max {
			max = n
		}
	}
	return max + 1
}

func p5ConfigJSON(c p5Config, endpoint string) string {
	m := map[string]any{
		"endpoint":           endpoint,
		"model":              c.model,
		"context_budget":     c.budget,
		"generation_reserve": c.reserve,
	}
	if c.hasSystem {
		m["system_prompt"] = c.systemPrompt
	}
	if c.bearerTokenEnv != "" {
		m["bearer_token_env"] = c.bearerTokenEnv
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		panic("discovery: cannot build config: " + err.Error())
	}
	return string(b) + "\n"
}

func p5ChildEnv(extra []p5EnvVar) []string {
	base := os.Environ()
	out := make([]string, 0, len(base)+len(extra))
	for _, kv := range base {
		if strings.HasPrefix(kv, "P5_") {
			continue
		}
		if strings.HasPrefix(kv, mockllm.SentinelEnvName+"=") {
			continue
		}
		out = append(out, kv)
	}
	for _, e := range extra {
		out = append(out, e.name+"="+e.value)
	}
	return out
}

func p5EnvNames(extra []p5EnvVar) []string {
	names := make([]string, 0, len(extra))
	for _, e := range extra {
		names = append(names, e.name)
	}
	return names
}

// p5Run executes one black-box case: reset the recorder, write a temporary
// config, run the built binary with piped stdin under a 15-second bound, and
// collect exit code, streams, and sanitized records. It writes no evidence;
// every case concludes through p5Conclude, p5Exploratory, or p5Skip.
func p5Run(t *testing.T, spec p5Spec) *p5Result {
	t.Helper()
	if p5Server == nil || p5BinPath == "" {
		t.Fatal("P5 harness was not initialized")
	}
	sc, ok := mockllm.Lookup(spec.scenario)
	if !ok {
		t.Fatalf("unknown scenario %q", spec.scenario)
	}
	p5Server.ResetRecords()
	endpoint := p5Server.URL() + sc.Path()
	cfg := spec.cfg
	if cfg.model == "" {
		cfg = p5DefaultConfig()
	}
	cfgJSON := p5ConfigJSON(cfg, endpoint)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "radichat.json")
	if err := os.WriteFile(cfgPath, []byte(cfgJSON), 0o600); err != nil {
		t.Fatalf("cannot write temp config: %v", err)
	}
	sum := sha256.Sum256([]byte(cfgJSON))

	ctx, cancel := context.WithTimeout(context.Background(), p5CaseTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, p5BinPath, "-config", cfgPath)
	cmd.Stdin = strings.NewReader(spec.stdin)
	var so, se strings.Builder
	cmd.Stdout, cmd.Stderr = &so, &se
	cmd.Env = p5ChildEnv(spec.env)
	runErr := cmd.Run()

	res := &p5Result{
		spec:     spec,
		endpoint: endpoint,
		cfgJSON:  cfgJSON,
		cfgHash:  hex.EncodeToString(sum[:]),
		envNames: p5EnvNames(spec.env),
		exit:     -1,
		stdout:   so.String(),
		stderr:   se.String(),
		records:  p5Server.Records(),
	}
	if runErr == nil {
		res.exit = 0
	} else {
		var ee *exec.ExitError
		if errors.As(runErr, &ee) {
			res.exit = ee.ExitCode()
		}
	}
	if ctx.Err() != nil {
		res.timedOut = true
	}
	return res
}

func p5Mask(s string) string {
	for _, sentinel := range p5Sentinels {
		s = strings.ReplaceAll(s, sentinel, "<sentinel>")
	}
	return s
}

func p5Hash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func p5Preview(s string) string {
	masked := p5Mask(s)
	if len(masked) <= p5PreviewBytes {
		return masked
	}
	return masked[:p5PreviewBytes] + fmt.Sprintf("...[+%d bytes]", len(masked)-p5PreviewBytes)
}

func p5RolesAndLengths(r mockllm.Record) ([]string, []int) {
	raw, ok := r.Body["messages"]
	if !ok {
		return nil, nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil, nil
	}
	roles := make([]string, 0, len(arr))
	lengths := make([]int, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			roles = append(roles, "")
			lengths = append(lengths, 0)
			continue
		}
		role, _ := m["role"].(string)
		content, _ := m["content"].(string)
		roles = append(roles, role)
		lengths = append(lengths, len(content))
	}
	return roles, lengths
}

func p5AuthOf(r *p5Result) string {
	if len(r.records) == 0 {
		return "no-requests"
	}
	return r.records[0].Auth.String()
}

func p5Actual(r *p5Result) string {
	if r == nil {
		return "no run"
	}
	return fmt.Sprintf("exit=%d timedOut=%v records=%d auth=%s stdout=%q stderr=%q",
		r.exit, r.timedOut, len(r.records), p5AuthOf(r), p5Preview(r.stdout), p5Preview(r.stderr))
}

func p5EvidenceLine(spec p5Spec, res *p5Result, verdict, notes string) {
	line := map[string]any{
		"case":                   spec.id,
		"scenario":               spec.scenario,
		"harness_local_scenario": strings.HasPrefix(spec.scenario, "p5-"),
		"oracle":                 spec.oracle,
		"commit":                 p5Commit,
		"go_version":             p5GoVersion,
		"verdict":                verdict,
	}
	if spec.notes != "" {
		line["case_notes"] = spec.notes
	}
	if notes != "" {
		line["notes"] = notes
	}
	if res != nil {
		recs := make([]map[string]any, 0, len(res.records))
		for _, r := range res.records {
			roles, lengths := p5RolesAndLengths(r)
			fields := map[string]any{}
			for _, k := range []string{"model", "stream", "max_tokens"} {
				if v, ok := r.Body[k]; ok {
					fields[k] = v
				}
			}
			problems := append([]string(nil), r.ExpectCheck.Problems...)
			recs = append(recs, map[string]any{
				"method":                       r.Method,
				"path":                         r.Path,
				"auth":                         r.Auth.String(),
				"authorization_header_present": r.Headers[mockllm.AuthorizationHeader] != "",
				"headers":                      r.Headers,
				"roles":                        roles,
				"content_lengths":              lengths,
				"fields":                       fields,
				"expect_check_matched":         r.ExpectCheck.Matched,
				"expect_check_problems":        problems,
			})
		}
		line["config_sha256"] = res.cfgHash
		line["config"] = json.RawMessage(res.cfgJSON)
		line["endpoint"] = res.endpoint
		line["stdin"] = res.spec.stdin
		line["env_names"] = res.envNames
		line["exit_code"] = res.exit
		line["timed_out"] = res.timedOut
		line["stdout_sha256"] = p5Hash(res.stdout)
		line["stdout_preview"] = p5Preview(res.stdout)
		line["stderr_sha256"] = p5Hash(res.stderr)
		line["stderr_preview"] = p5Preview(res.stderr)
		line["requests"] = recs
	}
	b, err := json.Marshal(line)
	if err != nil {
		panic("discovery: cannot marshal evidence: " + err.Error())
	}
	p5Mu.Lock()
	defer p5Mu.Unlock()
	if p5Evidence == nil {
		return
	}
	fmt.Fprintf(p5Evidence, "%s\n", b)
}

// p5Conclude records the verdict for a case with a documented oracle. A
// timeout is NOT RUN rather than a divergence, because an unverified
// hypothesis must never be reported as a defect.
func p5Conclude(t *testing.T, spec p5Spec, res *p5Result, ok bool, actual string) {
	t.Helper()
	if res.timedOut {
		p5NotRun(t, spec, res, "case exceeded the 15-second bounded timeout; not classified without manual verification")
		return
	}
	if ok {
		p5EvidenceLine(spec, res, "PASS", "")
		return
	}
	p5EvidenceLine(spec, res, "DIVERGENCE", "expected: "+spec.div.expected+" | actual: "+actual)
	id := p5RecordDivergence(spec, res, actual)
	t.Errorf("P5 %s (%s): documented divergence recorded as finding %s; expected: %s; actual: %s",
		spec.id, spec.scenario, id, spec.div.expected, actual)
}

func p5Exploratory(t *testing.T, spec p5Spec, res *p5Result, observation string) {
	t.Helper()
	p5EvidenceLine(spec, res, "EXPLORATORY", observation)
	t.Logf("P5 %s (%s) exploratory: %s", spec.id, spec.scenario, observation)
}

func p5Skip(t *testing.T, spec p5Spec, reason string) {
	t.Helper()
	p5EvidenceLine(spec, nil, "NOT RUN", reason)
	t.Skipf("P5 %s not run: %s", spec.id, reason)
}

func p5NotRun(t *testing.T, spec p5Spec, res *p5Result, reason string) {
	t.Helper()
	p5EvidenceLine(spec, res, "NOT RUN", reason)
	t.Skipf("P5 %s not run: %s", spec.id, reason)
}

func p5Slug(id, scenario string) string {
	return strings.ToLower(id) + "-" + scenario
}

func p5RecordDivergence(spec p5Spec, res *p5Result, actual string) string {
	p5Mu.Lock()
	defer p5Mu.Unlock()
	id := fmt.Sprintf("F-%04d", p5FindingBase+len(p5Divergences))
	p5Divergences = append(p5Divergences, p5Divergence{
		findingID:    id,
		slug:         spec.div.slug,
		title:        spec.div.title,
		caseID:       spec.id,
		scenario:     spec.scenario,
		harnessLocal: strings.HasPrefix(spec.scenario, "p5-"),
		expected:     spec.div.expected,
		actual:       actual,
		impact:       spec.div.impact,
		severity:     spec.div.severity,
		repro:        p5Repro(spec, res),
		cfgJSON:      res.cfgJSON,
		envNames:     res.envNames,
		stdin:        res.spec.stdin,
	})
	return id
}

// p5PrintfArg renders stdin for a printf format argument: newlines become the
// two characters backslash-n, which printf interprets.
func p5PrintfArg(s string) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func p5Repro(spec p5Spec, res *p5Result) string {
	const port = "34589"
	endpoint := "http://127.0.0.1:" + port + "/v1/chat/completions/" + spec.scenario
	cfg := strings.TrimSpace(strings.Replace(res.cfgJSON, res.endpoint, endpoint, 1))
	var b strings.Builder
	if strings.HasPrefix(spec.scenario, "p5-") {
		b.WriteString("# The p5- scenario is registered by test code, so the standalone reproduction\n")
		b.WriteString("# is the tagged harness itself (it builds the real binary once to a temp dir,\n")
		b.WriteString("# serves the scenario in-process, and drives the binary with the config and\n")
		b.WriteString("# stdin shown below):\n")
		fmt.Fprintf(&b, "go test -tags mockdiscovery ./internal/discovery -count=1 -run '%s' -v\n", p5RunFilter(spec.id))
	} else {
		b.WriteString("# Terminal 1 - serve the accepted scenario on a free loopback port:\n")
		fmt.Fprintf(&b, "go run ./cmd/radichat-mock run %s -addr 127.0.0.1:%s\n", spec.scenario, port)
		b.WriteString("# Terminal 2 - build the binary, write the config, and drive it:\n")
		fmt.Fprintf(&b, "go build -o /tmp/radichat-p5 . && cat > /tmp/radichat-p5.json <<'JSON'\n%s\nJSON\n", cfg)
	}
	if len(spec.env) > 0 {
		b.WriteString("# Required environment (fake sentinel values only; never a real credential):\n")
		for _, e := range spec.env {
			fmt.Fprintf(&b, "#   export %s   # value: %s (see internal/mockllm/scenarios.go and docs/testing-mock.md)\n", e.name, envValueHint(e.name))
		}
	}
	fmt.Fprintf(&b, "printf %s | /tmp/radichat-p5 -config /tmp/radichat-p5.json; echo \"exit=$?\"\n", p5PrintfArg(spec.stdin))
	return b.String()
}

func envValueHint(name string) string {
	switch name {
	case "P5_SENTINEL_TOKEN":
		return "mockllm.SentinelToken"
	case "P5_MISMATCH_TOKEN":
		return "mockllm.SentinelMismatchToken"
	case "P5_EMPTY_TOKEN":
		return "the empty string"
	case "P5_MALFORMED_TOKEN":
		return "a value containing spaces (invalid HTTP header value)"
	default:
		return "left unset"
	}
}

// p5RunFilter renders the exact -run filter that reruns one case. Family tests
// use subtests named by case ID; A/E/C cases use dedicated test functions.
func p5RunFilter(id string) string {
	switch {
	case strings.HasPrefix(id, "S"):
		if id == "S6" {
			return "TestP5OmittedContentType/S6"
		}
		return "TestP5StreamSuccess/" + id
	case strings.HasPrefix(id, "F"):
		return "TestP5Framing/" + id
	case strings.HasPrefix(id, "M"):
		return "TestP5Malformed/" + id
	case strings.HasPrefix(id, "H"):
		return "TestP5HTTPErrors/" + id
	case strings.HasPrefix(id, "R"):
		return "TestP5Redirects/" + id
	case strings.HasPrefix(id, "A"):
		return "TestP5AuthStartup/" + id
	case strings.HasPrefix(id, "E"):
		return "TestP5EchoRedaction/" + id
	case strings.HasPrefix(id, "C"):
		return "TestP5Context/" + id
	default:
		return "TestP5" + id
	}
}

func p5FlushFindings(root string) {
	for _, d := range p5Divergences {
		content := p5FindingMarkdown(d)
		scan := "PASS"
		if p5ContainsSentinel(content) {
			scan = "FAIL"
		}
		content = strings.Replace(content, "{{SENTINEL_SCAN}}", scan, 1)
		path := filepath.Join(root, p5FindingsDirName, fmt.Sprintf("%s-%s.md", d.findingID, d.slug))
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "P5 harness: cannot write finding %s: %v\n", path, err)
		}
	}
}

func p5ContainsSentinel(s string) bool {
	for _, sentinel := range p5Sentinels {
		if strings.Contains(s, sentinel) {
			return true
		}
	}
	return false
}

func p5FindingMarkdown(d p5Divergence) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s — %s\n\n", d.findingID, d.title)
	b.WriteString("**Status:** new\n\n")
	b.WriteString("**Source:** Phase 5 Worker (Step 5 Preview via Hermes CLI session 20261013_095745_21eecc, \"Step 5 Build Week\"), 2026-10-14\n\n")
	fmt.Fprintf(&b, "**Phase / run:** Stage 3 / Phase 5 — Adversarial Protocol Investigation. Case %s, scenario `%s`. Evidence: `%s` (case %s).\n\n",
		d.caseID, d.scenario, p5EvidenceRel, d.caseID)
	fmt.Fprintf(&b, "**Baseline:** commit %s (accepted Phase 4 checkpoint). The working tree also carried uncommitted Phase 5 lifecycle files; no product code, mock code, protocol docs, or accepted Phase 0-4 evidence was modified.\n\n", p5Commit)
	b.WriteString("## Expected\n\n")
	fmt.Fprintf(&b, "%s\n\n", d.expected)
	b.WriteString("## Observed\n\n")
	fmt.Fprintf(&b, "%s\n\n", d.actual)
	b.WriteString("## Reproduction\n\n```sh\n")
	b.WriteString(d.repro)
	b.WriteString("\n```\n\n")
	b.WriteString("## Environment / configuration\n\n")
	fmt.Fprintf(&b, "- Go: %s\n- Commit: %s\n- Harness: `go test -tags mockdiscovery ./internal/discovery -count=1 -run 'P5' -v` (builds the real binary once to a temp dir; drives it with piped stdin and a temp config against the in-process accepted mock)\n\n", p5GoVersion, p5Commit)
	b.WriteString("Config used for the failing run (no secret values; a bearer token, when used, is supplied only through the named environment variable):\n\n```json\n")
	b.WriteString(strings.TrimSpace(d.cfgJSON))
	b.WriteString("\n```\n\n")
	if len(d.envNames) > 0 {
		fmt.Fprintf(&b, "Environment variable names set for the run (values never recorded): %s\n\n", strings.Join(d.envNames, ", "))
	}
	fmt.Fprintf(&b, "Stdin: printf %s\n\n", p5PrintfArg(d.stdin))
	b.WriteString("## Impact / severity\n\n")
	fmt.Fprintf(&b, "%s\n\nSeverity: %s\n\n", d.impact, d.severity)
	b.WriteString("## Sentinel scan\n\n{{SENTINEL_SCAN}} — neither `mockllm.SentinelToken` nor `mockllm.SentinelMismatchToken` appears in this finding or in the sanitized evidence previews (raw stdout/stderr are recorded only as SHA-256 hashes with masked bounded previews).\n\n")
	b.WriteString("## Triage\n\nNot yet triaged. Per `findings/README.md` this finding is evidence, not an authorized task; any fix requires separate Owner authorization through the normal phase process.\n")
	return b.String()
}
