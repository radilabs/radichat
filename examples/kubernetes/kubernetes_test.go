package kubernetes

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/radilabs/radichat/internal/config"
)

func TestKubernetesManifests(t *testing.T) {
	t.Setenv("RADICHAT_BEARER_TOKEN", "example-token")
	cm := readManifest(t, "configmap.json")
	job := readManifest(t, "job.json")

	if cm["kind"] != "ConfigMap" || cm["apiVersion"] != "v1" {
		t.Fatalf("configmap kind %+v", cm["kind"])
	}
	if job["kind"] != "Job" || job["apiVersion"] != "batch/v1" {
		t.Fatalf("job kind %+v", job["kind"])
	}
	if dig(t, job, "spec", "backoffLimit") != 0.0 {
		t.Fatalf("backoffLimit %v", dig(t, job, "spec", "backoffLimit"))
	}
	deadline, _ := dig(t, job, "spec", "activeDeadlineSeconds").(float64)
	if deadline <= 0 {
		t.Fatalf("activeDeadlineSeconds %v", deadline)
	}
	if dig(t, job, "spec", "template", "spec", "restartPolicy") != "Never" {
		t.Fatalf("restartPolicy %v", dig(t, job, "spec", "template", "spec", "restartPolicy"))
	}

	pod := asMap(t, dig(t, job, "spec", "template", "spec"))
	if pod["automountServiceAccountToken"] != false {
		t.Fatalf("automountServiceAccountToken %v", pod["automountServiceAccountToken"])
	}
	if dig(t, pod, "securityContext", "runAsNonRoot") != true {
		t.Fatal("pod runAsNonRoot")
	}
	if dig(t, pod, "securityContext", "runAsUser") != 65532.0 {
		t.Fatalf("pod uid %v", dig(t, pod, "securityContext", "runAsUser"))
	}
	containers := asSlice(t, pod["containers"])
	if len(containers) != 1 {
		t.Fatalf("containers %d", len(containers))
	}
	ctr := asMap(t, containers[0])
	image, _ := ctr["image"].(string)
	if !strings.Contains(image, "REPLACE_WITH_V0_1_1_DIGEST") {
		t.Fatalf("image %q", image)
	}
	if ctr["stdin"] != false || ctr["tty"] != false {
		t.Fatalf("stdin/tty %+v %+v", ctr["stdin"], ctr["tty"])
	}
	sec := asMap(t, ctr["securityContext"])
	if sec["runAsNonRoot"] != true || sec["runAsUser"] != 65532.0 {
		t.Fatalf("container security %+v", sec)
	}
	if sec["readOnlyRootFilesystem"] != true || sec["allowPrivilegeEscalation"] != false {
		t.Fatalf("filesystem/privilege %+v", sec)
	}
	drop := asSlice(t, dig(t, sec, "capabilities", "drop"))
	if len(drop) != 1 || drop[0] != "ALL" {
		t.Fatalf("capabilities %v", drop)
	}

	args := stringSlice(t, ctr["args"])
	wantArgs := []string{"-config", "/config/radichat.json", "-input", "/config/input.txt", "-output", "-"}
	if strings.Join(args, " ") != strings.Join(wantArgs, " ") {
		t.Fatalf("args %v", args)
	}
	mounts := asSlice(t, ctr["volumeMounts"])
	if len(mounts) != 1 {
		t.Fatalf("mounts %d", len(mounts))
	}
	mount := asMap(t, mounts[0])
	if mount["mountPath"] != "/config" || mount["readOnly"] != true || mount["name"] != "config" {
		t.Fatalf("mount %+v", mount)
	}
	volumes := asSlice(t, pod["volumes"])
	if len(volumes) != 1 {
		t.Fatalf("volumes %d", len(volumes))
	}
	vol := asMap(t, volumes[0])
	if dig(t, vol, "configMap", "name") != "radichat-batch" {
		t.Fatalf("volume %+v", vol)
	}

	env := asSlice(t, ctr["env"])
	if len(env) != 1 {
		t.Fatalf("env %d", len(env))
	}
	ref := dig(t, asMap(t, env[0]), "valueFrom", "secretKeyRef")
	secretRef := asMap(t, ref)
	if secretRef["name"] != "radichat-token" || secretRef["key"] != "token" {
		t.Fatalf("secret ref %+v", secretRef)
	}
	if _, ok := asMap(t, env[0])["value"]; ok {
		t.Fatal("token value is inline")
	}

	data := asMap(t, cm["data"])
	for _, key := range []string{"radichat.json", "system-prompt.txt", "input.txt"} {
		if _, ok := data[key].(string); !ok {
			t.Fatalf("missing configmap key %s", key)
		}
	}
	blob, _ := json.Marshal(cm)
	if strings.Contains(string(blob), "secretKeyRef") || strings.Contains(string(blob), "example-token") {
		t.Fatal("configmap contains secret material")
	}
	cfg, err := config.Parse("/config/radichat.json", []byte(data["radichat.json"].(string)))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SystemPromptFile != "/config/system-prompt.txt" {
		t.Fatalf("prompt path %q", cfg.SystemPromptFile)
	}
	if cfg.BearerTokenEnv != "RADICHAT_BEARER_TOKEN" {
		t.Fatalf("env name %q", cfg.BearerTokenEnv)
	}
	if cfg.BearerTokenEnv != "" && !strings.HasPrefix(cfg.Endpoint, "https://") {
		t.Fatalf("bearer token configured with non-HTTPS endpoint %q", cfg.Endpoint)
	}
	if !strings.Contains(cfg.Endpoint, "llm.default.svc.cluster.local") {
		t.Fatalf("endpoint host changed: %q", cfg.Endpoint)
	}
	promptName := strings.TrimPrefix(cfg.SystemPromptFile, "/config/")
	if _, ok := data[promptName]; !ok {
		t.Fatalf("prompt %q is not a ConfigMap key", promptName)
	}
	if !strings.Contains(strings.Join(args, " "), "/config/input.txt") {
		t.Fatal("args do not read the mounted input")
	}
	if _, ok := data["input.txt"]; !ok {
		t.Fatal("input.txt missing")
	}
}

func readManifest(t *testing.T, name string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func dig(t *testing.T, m map[string]any, path ...string) any {
	t.Helper()
	var cur any = m
	for _, p := range path {
		obj, ok := cur.(map[string]any)
		if !ok {
			t.Fatalf("not an object at %s in %v", p, path)
		}
		cur = obj[p]
	}
	return cur
}

func asMap(t *testing.T, v any) map[string]any {
	t.Helper()
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("not a map: %#v", v)
	}
	return m
}

func asSlice(t *testing.T, v any) []any {
	t.Helper()
	s, ok := v.([]any)
	if !ok {
		t.Fatalf("not a slice: %#v", v)
	}
	return s
}

func stringSlice(t *testing.T, v any) []string {
	t.Helper()
	raw := asSlice(t, v)
	out := make([]string, len(raw))
	for i, item := range raw {
		s, ok := item.(string)
		if !ok {
			t.Fatalf("arg %d %#v", i, item)
		}
		out[i] = s
	}
	return out
}
