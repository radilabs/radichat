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
