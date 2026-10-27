# Container image

RadiChat publishes a public multi-platform OCI image at `ghcr.io/radilabs/radichat`. The `v0.1.0` image supports `linux/amd64` and `linux/arm64` and can be pulled without registry authentication:

```bash
docker pull ghcr.io/radilabs/radichat:v0.1.0
```

The immutable multi-platform digest is:

```text
ghcr.io/radilabs/radichat@sha256:2d8648b9f9d3c60273a7e432d0341a93e8e8fbcfe4c172617ddce3a07df98ecf
```

`latest` currently resolves to the same digest. Use the versioned tag or immutable digest when repeatability matters.

The `v0.1.0` image predates the v0.1.1 `system_prompt_file` key and the `-input` and `-output` flags. A config that uses those features fails against it. The `v0.1.1` image is described in [v0.1.1 published image](#v011-published-image) below.

## Configuration

The image reads `/config/radichat.json` by default. Create `radichat.local.json` from one of the tracked examples, edit it for your endpoint, and mount it read-only:

```bash
cp radichat.example.json radichat.local.json

docker run --rm -it \
  -v "$PWD/radichat.local.json:/config/radichat.json:ro" \
  ghcr.io/radilabs/radichat:v0.1.0
```

On Linux, an endpoint such as `http://127.0.0.1:8080` refers to the container itself. Add `--network host` when the model server listens on the host loopback interface:

```bash
docker run --rm -it --network host \
  -v "$PWD/radichat.local.json:/config/radichat.json:ro" \
  ghcr.io/radilabs/radichat:v0.1.0
```

Host networking behaves differently outside Linux. Point the config at an address reachable from the container when using Docker Desktop or another container environment.

## Authenticated remote endpoint

Keep the credential outside the config file. The config names the environment variable that contains it:

```bash
cp radichat.remote.example.json radichat.local.json
export RADICHAT_BEARER_TOKEN='<your-token>'

docker run --rm -it \
  -v "$PWD/radichat.local.json:/config/radichat.json:ro" \
  -e RADICHAT_BEARER_TOKEN \
  ghcr.io/radilabs/radichat:v0.1.0
```

The `-e RADICHAT_BEARER_TOKEN` form forwards the value from the current environment without placing it in the command. Do not bake credentials into images, put them in the JSON file, or include their values in command arguments.

## Image contents and verification

The minimal image contains the released static Linux binary and a CA certificate bundle for HTTPS. It runs as the unprivileged user and group `65532:65532`, uses `/config` as its working directory, and keeps TLS verification strict.

The published OCI index and both platform manifests were validated remotely. The exported linux/amd64 payload matched the released binary byte-for-byte, and that binary passed native Linux smoke tests before packaging. No local OCI runtime was available to execute the assembled container, and the linux/arm64 payload was not runtime-tested.

The paragraphs above describe the published `v0.1.0` image. Aside from the empty `/config` directory (owned by `65532`), each platform layer of that image contains only `/radichat` and `/etc/ssl/certs/ca-certificates.crt`, plus the parent directory entries `/etc/`, `/etc/ssl/`, and `/etc/ssl/certs/` that hold the CA bundle. The exact build recipe for `v0.1.0` is not recorded in the repository and cannot be recovered from the release evidence. See [Releasing RadiChat](releasing.md) for the recorded evidence and open items.

## v0.1.1 published image

The `v0.1.1` image is published at GHCR and Docker Hub. The full release record is [v0.1.1 release record](handoffs/release-v0.1.1.md).

- **Tags:** `ghcr.io/radilabs/radichat:v0.1.1`; `docker.io/naorw/radichat:v0.1.1`
- **Immutable digest (OCI index):** `sha256:10e843853f975998159e6110aeb55c5b9e1ec5db8edfd2f9d1ad59a9fc112505` (same in both registries)
- **Platform manifests:** `linux/amd64` `sha256:b7ac2e2b79ea8d0a6a10827700bbe7659d85a0e4c2918e482e514f2f81a1a8e3`; `linux/arm64` `sha256:807c9ef798bc83053e43253c4e177da6fd5885768ff23ad046ce3498ff46d9c1`
- **Embedded binaries** (verified against the registry-pulled payloads): `linux/amd64` `e9c5250904b5a7497d1a26b8da23440ed8e944e9279cb7ef1f5ca356997ce9b5`; `linux/arm64` `cf7a80f83dade0655847ee7f491d7ddc3a4f58daf0a02c67b8d0e4a10e110490`
- **Embedded CA bundle:** `796f1389bfea13c0ed78f663b765baaff54a508e5483786bd6ece73c5e3e3cbe`
- **Image revision label:** `b06dce63a2552d30d66ac4226b3ac957c28347fe`, the source commit of the binaries. The release tag commit `6645988e354076fede28070686b23667d0ff89ac` differs only in documentation and packaging files, and the binaries are the same.

Pull either registry's tag with:

```bash
docker pull ghcr.io/radilabs/radichat:v0.1.1
docker pull docker.io/naorw/radichat:v0.1.1
```

The image has the same non-root user (`65532:65532`), working directory (`/config`), entrypoint (`/radichat`), and default `CMD ["-config","/config/radichat.json"]` as `v0.1.0`. It has only `version` and `revision` labels; the title, description, source, URL, and license labels from `v0.1.0` are not present.

**Verification scope:** the registry-pulled OCI contents from GHCR and Docker Hub passed `scripts/verify_oci.py`, which checks the platform set, descriptor and blob digests, the image config, each layer's paths, modes, and ownership, and the binary and CA hashes. The image has **not been executed**. No OCI runtime was available, so this is structural, config, and payload verification only.

GHCR `latest` was **not** moved. It still resolves to the `v0.1.0` digest `sha256:2d8648b9f9d3c60273a7e432d0341a93e8e8fbcfe4c172617ddce3a07df98ecf`, and `v0.1.0` is unchanged. No Docker Hub `latest` tag was published. Use `v0.1.1` explicitly.
