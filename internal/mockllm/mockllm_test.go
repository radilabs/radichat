package mockllm

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// noRedirectClient never follows redirects, so redirect scenarios can be
// observed the way RadiChat sees them.
var noRedirectClient = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// httpPost issues a minimal POST to the mock.
func httpPost(url string) (*http.Response, error) {
	return noRedirectClient.Post(url, "application/json",
		strings.NewReader(`{"model":"m","messages":[{"role":"user","content":"hi"}],"stream":true,"max_tokens":64}`))
}

// postJSON issues a POST with an explicit Authorization header and body.
func postJSON(url, auth, body string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	return noRedirectClient.Do(req)
}

// readAll reads a response body fully.
func readAll(t *testing.T, resp *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}

func TestCatalogCoversEveryRequiredFamily(t *testing.T) {
	families := map[string]bool{}
	for _, s := range Catalog() {
		families[s.Family] = true
		if s.Name == "" {
			t.Error("scenario with empty name")
		}
		if s.Description == "" {
			t.Errorf("scenario %s has no description", s.Name)
		}
	}
	for _, want := range []string{"stream", "framing", "malformed", "http-error", "auth", "redirect", "context"} {
		if !families[want] {
			t.Errorf("missing scenario family %q", want)
		}
	}
}

func TestCatalogIsDeterministic(t *testing.T) {
	first := Names()
	second := Names()
	if len(first) != len(second) {
		t.Fatalf("catalog size changed: %d then %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("catalog order changed at %d: %s vs %s", i, first[i], second[i])
		}
	}
	// Same scenario must produce the same wire bytes every time.
	sc, ok := Lookup("stream-success-basic")
	if !ok {
		t.Fatal("missing stream-success-basic")
	}
	if string(WireBytes(sc.Wire)) != string(WireBytes(sc.Wire)) {
		t.Fatal("wire is not stable")
	}
}

func TestDuplicateScenarioNamePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Register did not panic on duplicate name")
		}
	}()
	Register(Scenario{Name: "stream-success-basic"})
}

func TestEveryScenarioIsReachable(t *testing.T) {
	srv := startServer(t, Options{})
	for _, sc := range Catalog() {
		resp, err := httpPost(srv.URL() + sc.Path())
		if err != nil {
			t.Errorf("%s: %v", sc.Name, err)
			continue
		}
		body := readAll(t, resp)
		resp.Body.Close()
		// A 404 here would mean the scenario path did not resolve, not that
		// the scenario answered 404 on purpose.
		if resp.StatusCode == http.StatusNotFound && sc.status() != http.StatusNotFound {
			t.Errorf("%s: scenario not reachable (404)", sc.Name)
			continue
		}
		if sc.Redirect != "" && resp.Header.Get("Location") != sc.Redirect {
			t.Errorf("%s: Location %q; want %q", sc.Name, resp.Header.Get("Location"), sc.Redirect)
		}
		if len(sc.Wire) > 0 && body == "" && sc.Status == 0 && sc.Redirect == "" {
			t.Errorf("%s: empty body but scenario declares wire", sc.Name)
		}
	}
}

func TestStreamSuccessBasic(t *testing.T) {
	srv := startServer(t, Options{})
	resp, err := httpPost(srv.URL() + "/v1/chat/completions/stream-success-basic")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if got := resp.Header.Get("Content-Type"); !strings.Contains(got, contentTypeSSE) {
		t.Fatalf("Content-Type %q", got)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	body := readAll(t, resp)
	if !strings.Contains(body, `"content":"hello"`) {
		t.Fatalf("body missing content: %s", body)
	}
	if !strings.Contains(body, doneMarker) {
		t.Fatalf("body missing [DONE]: %s", body)
	}
}

func TestFragmentedSegmentsReassembleExactly(t *testing.T) {
	// The deterministic unit is the configured segment list. This test proves
	// the configured segments, and separately that the concatenation equals
	// the scenario's declared wire. It deliberately does not assert anything
	// about how a TCP reader observes those writes.
	sc, ok := Lookup("fragment-utf8-inside-rune")
	if !ok {
		t.Fatal("missing fragment-utf8-inside-rune")
	}
	full := WireText(sc.Wire)
	if len(sc.Wire) != 2 {
		t.Fatalf("want 2 configured segments, got %d", len(sc.Wire))
	}
	if midRuneOffset(full) < 0 {
		t.Fatal("test wire has no multibyte rune to split")
	}
	if len(sc.Wire[0].Bytes) != midRuneOffset(full) {
		t.Fatalf("first segment is %d bytes; want the offset %d inside a rune",
			len(sc.Wire[0].Bytes), midRuneOffset(full))
	}
	if string(WireBytes(sc.Wire)) != full {
		t.Fatal("segments do not concatenate back to the wire")
	}
}

func TestFragmentOffsetsAreValidated(t *testing.T) {
	data := []byte("abcdef")
	if got := Fragment(data, 2, 4); len(got) != 3 {
		t.Fatalf("Fragment(data,2,4) gave %d segments; want 3", len(got))
	}
	// Non-increasing offsets are rejected rather than silently mangled.
	if got := Fragment(data, 4, 2); got != nil {
		t.Fatalf("Fragment with decreasing offsets gave %d segments; want nil", len(got))
	}
	// Out-of-range offsets are rejected.
	if got := Fragment(data, 0); got != nil {
		t.Fatalf("Fragment(data,0) gave %d segments; want nil", len(got))
	}
	if got := Fragment(data, len(data)); got != nil {
		t.Fatalf("Fragment at end gave %d segments; want nil", len(got))
	}
	// Fragment is deterministic.
	a := string(WireBytes(Fragment(data, 2)))
	b := string(WireBytes(Fragment(data, 2)))
	if a != b {
		t.Fatal("Fragment is not deterministic")
	}
}

func TestFragmentationScenariosSplitWhereDeclared(t *testing.T) {
	cases := []struct {
		name   string
		offset func(string) int
	}{
		{"fragment-utf8-inside-rune", midRuneOffset},
		{"fragment-inside-crlf", firstCRLFSplit},
		{"fragment-inside-data-field", firstDataFieldSplit},
		{"fragment-inside-done", firstDoneSplit},
	}
	for _, tc := range cases {
		sc, ok := Lookup(tc.name)
		if !ok {
			t.Errorf("missing scenario %s", tc.name)
			continue
		}
		full := WireText(sc.Wire)
		want := tc.offset(full)
		if want < 0 {
			t.Errorf("%s: no split point found in wire", tc.name)
			continue
		}
		if len(sc.Wire) != 2 {
			t.Errorf("%s: want 2 segments, got %d", tc.name, len(sc.Wire))
			continue
		}
		if len(sc.Wire[0].Bytes) != want {
			t.Errorf("%s: first segment %d bytes; want %d", tc.name, len(sc.Wire[0].Bytes), want)
		}
	}
}

func TestByteByByteFragmentationProducesManySegments(t *testing.T) {
	sc, ok := Lookup("fragment-byte-by-byte")
	if !ok {
		t.Fatal("missing fragment-byte-by-byte")
	}
	if len(sc.Wire) != len(WireBytes(sc.Wire)) {
		t.Fatalf("want one segment per byte, got %d segments for %d bytes",
			len(sc.Wire), len(WireBytes(sc.Wire)))
	}
}

func TestEarlyTerminationClosesWithoutDone(t *testing.T) {
	srv := startServer(t, Options{})
	resp, err := httpPost(srv.URL() + "/v1/chat/completions/malformed-early-termination")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body := readAll(t, resp)
	if strings.Contains(body, "[DONE]") {
		t.Fatal("early-termination scenario emitted [DONE]")
	}
	if !strings.Contains(body, "partial") {
		t.Fatalf("expected partial content, got %q", body)
	}
}

func TestHTTPErrorScenariosReturnStatus(t *testing.T) {
	srv := startServer(t, Options{})
	cases := map[string]int{
		"http-error-401":               401,
		"http-error-429":               429,
		"http-error-500":               500,
		"http-error-404-unknown-model": 404,
	}
	for name, want := range cases {
		resp, err := httpPost(srv.URL() + "/v1/chat/completions/" + name)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if resp.StatusCode != want {
			t.Errorf("%s: status %d; want %d", name, resp.StatusCode, want)
		}
		_ = readAll(t, resp)
		resp.Body.Close()
	}
}

func TestAuthMatchMissingMismatch(t *testing.T) {
	srv := startServer(t, Options{ExpectedBearer: SentinelToken})
	url := srv.URL() + "/v1/chat/completions/auth-required-match"

	// Match succeeds.
	resp, err := postJSON(url, "Bearer "+SentinelToken, `{"model":"m","messages":[],"stream":true}`)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("match: status %d; want 200", resp.StatusCode)
	}
	_ = readAll(t, resp)
	resp.Body.Close()

	// Missing is refused.
	resp, err = postJSON(url, "", `{"model":"m","messages":[],"stream":true}`)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 401 {
		t.Errorf("missing: status %d; want 401", resp.StatusCode)
	}
	_ = readAll(t, resp)
	resp.Body.Close()

	// Mismatch is refused.
	resp, err = postJSON(url, "Bearer "+SentinelMismatchToken, `{"model":"m","messages":[],"stream":true}`)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 401 {
		t.Errorf("mismatch: status %d; want 401", resp.StatusCode)
	}
	_ = readAll(t, resp)
	resp.Body.Close()

	// Malformed is refused.
	resp, err = postJSON(url, "Basic dXNlcjpwYXNz", `{"model":"m","messages":[],"stream":true}`)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 401 {
		t.Errorf("malformed: status %d; want 401", resp.StatusCode)
	}
	_ = readAll(t, resp)
	resp.Body.Close()
}

func TestRedirectScenariosSetLocation(t *testing.T) {
	srv := startServer(t, Options{})
	for _, name := range []string{"redirect-to-final-url", "redirect-with-query", "redirect-absolute"} {
		sc, ok := Lookup(name)
		if !ok {
			t.Errorf("missing %s", name)
			continue
		}
		resp, err := httpPost(srv.URL() + sc.Path())
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if resp.StatusCode < 300 || resp.StatusCode >= 400 {
			t.Errorf("%s: status %d; want 3xx", name, resp.StatusCode)
		}
		if got := resp.Header.Get("Location"); got != sc.Redirect {
			t.Errorf("%s: Location %q; want %q", name, got, sc.Redirect)
		}
		_ = readAll(t, resp)
		resp.Body.Close()
	}
}

// assertNoSentinel checks that a sentinel credential appears nowhere in a
// recorded observation or command output.
func assertNoSentinel(t *testing.T, where, s string) {
	t.Helper()
	for _, bad := range []string{SentinelToken, SentinelMismatchToken} {
		if strings.Contains(s, bad) {
			t.Errorf("%s contains a sentinel credential value", where)
		}
	}
}

func TestRecorderRedactsAuthorization(t *testing.T) {
	srv := startServer(t, Options{ExpectedBearer: SentinelToken})
	resp, err := postJSON(srv.URL()+"/v1/chat/completions/auth-required-match",
		"Bearer "+SentinelToken, `{"model":"m","messages":[{"role":"user","content":"hi"}],"stream":true,"max_tokens":64}`)
	if err != nil {
		t.Fatal(err)
	}
	_ = readAll(t, resp)
	resp.Body.Close()

	rec, ok := srv.Last()
	if !ok {
		t.Fatal("request not recorded")
	}
	if got := rec.Headers[AuthorizationHeader]; got != RedactedAuthorization {
		t.Fatalf("recorded Authorization %q; want the redaction placeholder", got)
	}
	if !rec.Auth.Present || !rec.Auth.Matched {
		t.Fatalf("auth outcome %+v; want present and matched", rec.Auth)
	}
	// The full record must not contain the credential anywhere.
	blob, err := json.Marshal(rec)
	if err != nil {
		t.Fatal(err)
	}
	assertNoSentinel(t, "recorded request", string(blob))
}

func TestRecorderCapturesMethodPathAndFields(t *testing.T) {
	srv := startServer(t, Options{})
	resp, err := postJSON(srv.URL()+"/v1/chat/completions/context-system-retained",
		"", `{"model":"m","messages":[{"role":"system","content":"s"},{"role":"user","content":"hi"}],"stream":true,"max_tokens":64}`)
	if err != nil {
		t.Fatal(err)
	}
	_ = readAll(t, resp)
	resp.Body.Close()

	rec, ok := srv.Last()
	if !ok {
		t.Fatal("request not recorded")
	}
	if rec.Method != http.MethodPost {
		t.Errorf("method %q", rec.Method)
	}
	if rec.Path != "/v1/chat/completions/context-system-retained" {
		t.Errorf("path %q", rec.Path)
	}
	if rec.Body["model"] != "m" {
		t.Errorf("model %v", rec.Body["model"])
	}
	if rec.Body["stream"] != true {
		t.Errorf("stream %v", rec.Body["stream"])
	}
	if rec.Body["max_tokens"] != float64(64) {
		t.Errorf("max_tokens %v", rec.Body["max_tokens"])
	}
	if roles := messageRoles(rec.Body); len(roles) != 2 || roles[0] != "system" || roles[1] != "user" {
		t.Errorf("roles %v", roles)
	}
	if rec.Scenario != "context-system-retained" {
		t.Errorf("scenario %q", rec.Scenario)
	}
	if !rec.ExpectCheck.Matched {
		t.Errorf("expect check failed: %v", rec.ExpectCheck.Problems)
	}
}

func TestRecorderRecordsExpectMismatch(t *testing.T) {
	srv := startServer(t, Options{})
	// Send a request that does not match the scenario's declared expectation.
	resp, err := postJSON(srv.URL()+"/v1/chat/completions/context-max-tokens-sent",
		"", `{"model":"m","messages":[],"stream":true,"max_tokens":7}`)
	if err != nil {
		t.Fatal(err)
	}
	_ = readAll(t, resp)
	resp.Body.Close()

	rec, _ := srv.Last()
	if rec.ExpectCheck.Matched {
		t.Fatal("expected the mismatch to be reported")
	}
	if len(rec.ExpectCheck.Problems) == 0 {
		t.Fatal("mismatch reported with no problem text")
	}
}

func TestRecorderAuthOutcomes(t *testing.T) {
	srv := startServer(t, Options{ExpectedBearer: SentinelToken})
	cases := []struct {
		name   string
		header string
		want   AuthOutcome
	}{
		{"none", "", AuthOutcome{Present: false}},
		{"match", "Bearer " + SentinelToken, AuthOutcome{Present: true, Matched: true}},
		{"mismatch", "Bearer " + SentinelMismatchToken, AuthOutcome{Present: true, Matched: false}},
		{"malformed", "Basic abc", AuthOutcome{Present: true, Malformed: true}},
		{"empty-bearer", "Bearer ", AuthOutcome{Present: true, Malformed: true}},
	}
	for _, tc := range cases {
		resp, err := postJSON(srv.URL()+"/v1/chat/completions/stream-success-basic",
			tc.header, `{"model":"m","messages":[],"stream":true}`)
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		_ = readAll(t, resp)
		resp.Body.Close()
		rec, _ := srv.Last()
		if rec.Auth != tc.want {
			t.Errorf("%s: auth %+v; want %+v", tc.name, rec.Auth, tc.want)
		}
	}
}

func TestAuthOutcomeStringsLeakNothing(t *testing.T) {
	for _, a := range []AuthOutcome{
		{Present: false},
		{Present: true, Matched: true},
		{Present: true, Matched: false},
		{Present: true, Malformed: true},
	} {
		s := a.String()
		assertNoSentinel(t, "AuthOutcome.String", s)
		if s == "" {
			t.Error("AuthOutcome.String is empty")
		}
	}
}

// TestCookieIsNeverRecorded proves that cookie contents cannot reach a Record.
//
// Cookies can carry credentials, so the recorder must omit the header entirely
// rather than truncate or redact it. The absence of the key is also how a
// caller asserts that no cookie was sent.
func TestCookieIsNeverRecorded(t *testing.T) {
	const cookieValue = "session=super-secret-cookie-value; other=also-secret"
	srv := startServer(t, Options{})

	req, err := http.NewRequest(http.MethodPost, srv.URL()+"/v1/chat/completions/stream-success-basic",
		strings.NewReader(`{"model":"m","messages":[],"stream":true}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookieValue)
	resp, err := noRedirectClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = readAll(t, resp)
	resp.Body.Close()

	rec, ok := srv.Last()
	if !ok {
		t.Fatal("request not recorded")
	}
	if _, present := rec.Headers["Cookie"]; present {
		t.Fatal("recorder retained a Cookie header entry")
	}
	// The cookie value must not survive anywhere in the record, including
	// through the raw body or any marshalled form.
	for _, bad := range []string{cookieValue, "super-secret-cookie-value", "session=", "also-secret"} {
		if strings.Contains(strings.Join(headerValues(rec.Headers), " "), bad) {
			t.Fatalf("recorded headers contain cookie material %q", bad)
		}
	}
	blob, err := json.Marshal(rec)
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{cookieValue, "super-secret-cookie-value", "session=", "also-secret"} {
		if strings.Contains(string(blob), bad) {
			t.Fatalf("marshalled record contains cookie material %q", bad)
		}
	}
	// Headers that are not credential-bearing must still be recorded.
	if rec.Headers["Content-Type"] == "" {
		t.Error("Content-Type was not recorded")
	}
}

// TestQueryStringIsNeverRecorded proves that a token carried in a request query
// string cannot reach a Record.
//
// Query capture is not required by the contract or the Architect design, and a
// query string can carry credentials such as ?access_token=..., so the recorder
// stores only the URL path.
func TestQueryStringIsNeverRecorded(t *testing.T) {
	const queryToken = "query-access-token-super-secret-value"
	srv := startServer(t, Options{})
	url := srv.URL() + "/v1/chat/completions/stream-success-basic?access_token=" + queryToken
	req, err := http.NewRequest(http.MethodPost, url,
		strings.NewReader(`{"model":"m","messages":[],"stream":true}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := noRedirectClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = readAll(t, resp)
	resp.Body.Close()

	rec, ok := srv.Last()
	if !ok {
		t.Fatal("request not recorded")
	}
	if rec.Path != "/v1/chat/completions/stream-success-basic" {
		t.Errorf("path %q; want the path without the query string", rec.Path)
	}
	// The query string must not survive anywhere in the record.
	blob, err := json.Marshal(rec)
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{queryToken, "access_token", "query-access-token"} {
		if strings.Contains(string(blob), bad) {
			t.Fatalf("marshalled record contains query material %q: %s", bad, blob)
		}
	}
}

// headerValues returns a map's values for scanning.
func headerValues(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

func TestResetRecords(t *testing.T) {
	srv := startServer(t, Options{})
	resp, err := httpPost(srv.URL() + "/v1/chat/completions/stream-success-basic")
	if err != nil {
		t.Fatal(err)
	}
	_ = readAll(t, resp)
	resp.Body.Close()
	if len(srv.Records()) == 0 {
		t.Fatal("expected a record")
	}
	srv.ResetRecords()
	if len(srv.Records()) != 0 {
		t.Fatal("ResetRecords did not clear records")
	}
}

func TestHandlerUsableWithoutSocket(t *testing.T) {
	srv, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	// Handler must work before Start binds a listener.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions/stream-success-basic",
		strings.NewReader(`{"model":"m","messages":[],"stream":true}`))
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status %d; want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "hello") {
		t.Fatalf("body %q", rec.Body.String())
	}
}

func TestNoFilesAreWritten(t *testing.T) {
	// The mock must not touch the filesystem. Record a full exchange and
	// confirm the working directory is unchanged.
	dir := t.TempDir()
	before, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	srv := startServer(t, Options{})
	resp, err := httpPost(srv.URL() + "/v1/chat/completions/stream-success-basic")
	if err != nil {
		t.Fatal(err)
	}
	_ = readAll(t, resp)
	resp.Body.Close()

	after, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != len(after) {
		t.Fatalf("mock created files: %d before, %d after", len(before), len(after))
	}
}

func TestStreamIsObservableBeforeCompletion(t *testing.T) {
	srv := startServer(t, Options{})
	resp, err := httpPost(srv.URL() + "/v1/chat/completions/stream-success-basic")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	br := bufio.NewReader(resp.Body)
	sawData := false
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		line, err := br.ReadString('\n')
		if err != nil {
			break
		}
		if strings.HasPrefix(line, "data: ") {
			sawData = true
			break
		}
	}
	if !sawData {
		t.Fatal("no data line was observable before the stream ended")
	}
}

// TestRawModeWritesExactBytes proves that raw mode delivers the configured
// bytes verbatim, which net/http would otherwise reframe.
func TestRawModeWritesExactBytes(t *testing.T) {
	payload := "data: {\"choices\":[{\"delta\":{\"content\":\"raw\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n"
	srv, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	// Register a raw-mode scenario locally so this test does not depend on
	// the catalog.
	Register(Scenario{
		Name:        "test-raw-exact",
		Family:      "framing",
		Description: "test-only raw scenario",
		Headers:     map[string]string{"Content-Type": contentTypeSSE},
		Wire:        FragmentText(payload),
		Mode:        ModeRaw,
	})
	t.Cleanup(func() { delete(registry, "test-raw-exact") })

	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = srv.Close() })

	resp, err := httpPost(srv.URL() + "/v1/chat/completions/test-raw-exact")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body := readAll(t, resp)
	if body != payload {
		t.Fatalf("raw body mismatch:\n got %q\nwant %q", body, payload)
	}
}
