# Kubernetes Job example

The JSON manifests in `examples/kubernetes/` show one non-interactive RadiChat run inside a Kubernetes Job. RadiChat does not call the Kubernetes API. The examples were checked with `encoding/json` and the RadiChat config parser. No cluster was contacted. The `v0.1.1` image is published on GHCR at the digest in the Image section below. It has not been run in a container or in a cluster.

`kubectl logs` can mix stdout and stderr. A JSONL consumer must read stdout separately or filter out diagnostic lines.

## What the Job runs

The container is non-interactive: `stdin` and `tty` are false. Arguments are:

```text
-config /config/radichat.json -input /config/input.txt -output -
```

The ConfigMap `radichat-batch` provides `radichat.json`, `system-prompt.txt`, and `input.txt`, mounted read-only at `/config`. `system_prompt_file` is the absolute path `/config/system-prompt.txt`. Results are JSONL on stdout. Kubernetes log collection is outside RadiChat; the process does not need a writable filesystem for the result file.

The endpoint stays on the placeholder cluster host `llm.default.svc.cluster.local` and uses HTTPS, `https://llm.default.svc.cluster.local/v1/chat/completions`, because `bearer_token_env` is set. The operator must provision a certificate that this container trusts before the Job can complete a TLS handshake. RadiChat does not install a custom CA. This example does not show that handshake or claim a cluster run.

`bearer_token_env` names `RADICHAT_BEARER_TOKEN`. The Job injects that variable from the Secret `radichat-token`, key `token`. The token is not stored in the ConfigMap. Create the Secret before running the Job. If the variable is missing, RadiChat exits before any request.

## Pod constraints

- `restartPolicy: Never`
- `backoffLimit: 0` so a failed batch is not run again automatically
- `activeDeadlineSeconds: 120`
- `automountServiceAccountToken: false` on the Pod spec
- `seccompProfile: RuntimeDefault` on the Pod spec
- `runAsNonRoot` and UID/GID 65532
- read-only root filesystem
- `allowPrivilegeEscalation: false`
- all capabilities dropped
- config volume mounted read-only

## Image

The published `v0.1.1` image is:

```text
ghcr.io/radilabs/radichat@sha256:10e843853f975998159e6110aeb55c5b9e1ec5db8edfd2f9d1ad59a9fc112505
```

This is the GHCR OCI index digest for `v0.1.1`. It covers the `linux/amd64` and `linux/arm64` platform manifests. The release record is in [Releasing RadiChat](releasing.md) and [v0.1.1 release record](handoffs/release-v0.1.1.md).

**Manifest status:** `examples/kubernetes/job.json` uses the digest above, and `examples/kubernetes/kubernetes_test.go` asserts that exact reference. The manifest and this page must change together.

Do not treat this example as proof that the container image runs. The image has not been executed in a container or a cluster.

The `-input`, `-output`, and `system_prompt_file` features exist only in the `v0.1.1` image. The published `v0.1.0` image predates them, so this Job cannot run with that image.

## Files

| File | Role |
| --- | --- |
| `examples/kubernetes/configmap.json` | Config, system prompt, and input text |
| `examples/kubernetes/job.json` | Job that mounts the ConfigMap and writes JSONL to stdout |

`go test ./examples/kubernetes` checks the manifests against each other and parses the embedded config. It does not apply them.
