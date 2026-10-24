# Releasing RadiChat

This document records three things: the evidence for the published `v0.1.0` binaries and container image, the checked-in local OCI packaging procedure for `v0.1.1`, and the local `v0.1.1` artifacts with their open items. It does not authorize a tag, GitHub release, GHCR push, or any change to `v0.1.0`. Those require separate Owner authorization.

Each statement is marked as **verified** (checked against a named source in this repository or a read-only registry or GitHub fetch), **reported** (taken from a source this repository does not contain), or **unknown**.

## v0.1.0 evidence

### Source and tag

- **Verified:** tag `v0.1.0` resolves to commit `043216c3e1cb6fa962d97ae1580de130d87db683` (`git rev-parse v0.1.0^{commit}`).
- **Verified:** the `v0.1.0` tree has no `Dockerfile`, `Containerfile`, `Makefile`, shell scripts, release workflow, or `goreleaser` configuration (`git ls-tree -r --name-only v0.1.0`).
- **Verified:** no release-builder commands are checked into the repository at that tag.

### Binaries

The GitHub release `v0.1.0` states that all binaries were built from tag `v0.1.0` at commit `043216c3e1cb6fa962d97ae1580de130d87db683` with Go 1.27.1, `CGO_ENABLED=0`, `-buildvcs=false`, `-trimpath`, and `-ldflags="-s -w"`. It also states that each target was built twice and compared byte-for-byte (**reported**).

| Artifact | SHA-256 (release `SHA256SUMS`, **verified**) |
| --- | --- |
| `radichat-linux-amd64` | `9c04bbee1effbceb43b6382ea7afccd65b4abfb5492c118e258d54586de0ab19` |
| `radichat-linux-arm64` | `161b6eec7ac8f20cb7573e989af3878c1dbe8ad91186b98d8ec7bf44347ac41b` |
| `radichat-darwin-amd64` | `d05fa075cae9dea81673c2dd103611977520cce4d6a44b6f14948b63eb8af65a` |
| `radichat-darwin-arm64` | `d2f8655c322ea77953d45c9dc7de926feaf84775b06084faa66834b5d1e37cd6` |

- **Reported:** `linux/amd64` was smoke-tested natively. The `linux/arm64` and `darwin/*` artifacts were cross-compiled and not runtime-tested. The macOS artifacts are unsigned and not notarized. RadiChat claims no macOS runtime support.

### Container image

- **Verified:** the image index `ghcr.io/radilabs/radichat:v0.1.0` has digest `sha256:2d8648b9f9d3c60273a7e432d0341a93e8e8fbcfe4c172617ddce3a07df98ecf` (`Docker-Content-Digest` header, read-only fetch, 2026-10-23). `latest` currently resolves to the same digest.
- **Verified:** the index is an OCI image index that lists two platform manifests:
  - `linux/amd64`: `sha256:169beada7b0ade8329e446faa7a69bb88b710fe1fbb604d3765f5ca398f60a54`
  - `linux/arm64`: `sha256:949ffe0a0f69b131b5110fd559ed133ade58be0388e12bbaefe61379dcadccf0`

  The arm64 digest is 64 hex characters. An earlier transcription had 63 and is wrong.
- **Verified, per manifest:** each platform manifest has one `application/vnd.oci.image.layer.v1.tar+gzip` layer, and the fetched config digest matches its manifest.
- **Verified, amd64 config:** `Entrypoint ["/radichat"]`, `Cmd ["-config","/config/radichat.json"]`, `User "65532:65532"`, `WorkingDir "/config"`, `Env SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt`, `created` and the single history entry at `0001-01-01T00:00:00Z`, no `created_by`. Labels include `version=v0.1.0` and `revision=043216c3e1cb6fa962d97ae1580de130d87db683`, plus title, description, source, URL, and license labels.
- **Verified, arm64 config:** architecture, `User`, `Entrypoint`, `Cmd`, and `WorkingDir` match the amd64 config. Labels were not re-checked.
- **Verified, amd64 layer:** one layer, 3,456,453 bytes, digest `sha256:8c3a341c27d602b18b5a297d14646a655d6263ab22c8a374787d3cc3175b274a`. Entries: `config/` (owner 65532), `etc/`, `etc/ssl/`, `etc/ssl/certs/`, `etc/ssl/certs/ca-certificates.crt` (218,113 bytes), `radichat` (6,824,096 bytes, mode 0755).
- **Verified, arm64 layer:** one layer, 3,108,733 bytes, digest `sha256:d854a1101f35795d982adbf166b144781ca4391679096cc99fd547d2aca25299`. Entries: the same paths, with `radichat` at 6,291,616 bytes.
- **Verified, binary identity:** the extracted `/radichat` from the amd64 layer hashes to `9c04bbee…0ab19`, and the extracted arm64 binary hashes to `161b6eec…ac41b`. Both match the release `SHA256SUMS`.
- **Reported:** the release states the image runs as UID/GID `65532:65532`, contains only the static binary and a CA bundle, and was not run because no local OCI runtime was available. The arm64 payload was not runtime-tested.

### Not reproducible from the evidence

- **Unknown:** the exact tool, command, and options used to build and publish the `v0.1.0` OCI image. No checked-in artifact records them.
- **Unknown:** the source and checksum of the CA bundle used for `v0.1.0`.
- **Unknown:** whether the `v0.1.0` image was assembled by a standard builder or by hand. The evidence is consistent with hand assembly: one layer per platform, an empty history entry, and zero-epoch timestamps. This is an **inference**, not a proven fact.
- **Consequence:** the `v0.1.0` image cannot be rebuilt bit-for-bit from this repository. Its binaries are reproducible from the recorded flags, and its layer contents are verified. Its build recipe is not. Do not describe the `v0.1.0` image as reproducible.

## v0.1.1 local OCI packaging

This section describes the checked-in procedure and the local artifacts produced under it. The artifacts are local only. No registry, tag, or release action has been taken.

### Checked-in files

| Path | Role |
| --- | --- |
| `Containerfile` | Two-stage `FROM scratch` build. Stage `payload` copies `radichat-linux-${TARGETARCH}` to `/radichat` and `ca-certificates.crt` to `/etc/ssl/certs/ca-certificates.crt`, both owned `65532:65532`, and sets `USER 65532:65532` and `WORKDIR /config`. Stage `final` copies the payload root, sets `USER`, `WORKDIR /config`, `ENV SSL_CERT_FILE=…`, `ENTRYPOINT ["/radichat"]`, `CMD ["-config","/config/radichat.json"]`, and the labels `version="v0.1.1"` and `revision=${SOURCE_COMMIT}`. |
| `container/.dockerignore` | Excludes everything from the build context except `Containerfile`, the two Linux binaries, and `ca-certificates.crt`. The build script copies it into the generated context as `.dockerignore`. |
| `container/ca-certificates.crt` | Vendored CA bundle. Provenance is in `container/CA-BUNDLE.md`. |
| `scripts/build-oci-v0.1.1.sh` | Pinned rootless BuildKit build of the OCI archive, run twice from two fresh state roots, followed by the byte comparison and the verifier. |
| `scripts/verify_oci.py` | Offline verifier for the OCI archive. It checks the exact platforms, every descriptor and blob digest, the image config, each layer's paths, modes, and ownership, the binary hashes, and the CA bundle hash. |
| `scripts/test_verify_oci.py` | Unit tests for the verifier. |

### Pinned inputs

| Input | Pin | Source of the pin |
| --- | --- | --- |
| BuildKit | `v0.34.0`, `linux-amd64` tarball, SHA-256 `14813b1fb2a40082114e6d386c6fc5e7dd5298176ce7ee1a18bc33e73e109ac3` | `scripts/build-oci-v0.1.1.sh`. The local tarball matches this hash (**verified**, 2026-10-24). |
| RootlessKit | `v3.3.0`, `x86_64` tarball, SHA-256 `0e757c8f80510d764df8c9a5f169805580b48d02329f0a200900e261ecec27ed` | `scripts/build-oci-v0.1.1.sh`. The local tarball matches this hash (**verified**). |
| CA bundle | curl.se Mozilla CA snapshot, 118 certificates, 200,116 bytes, SHA-256 `796f1389bfea13c0ed78f663b765baaff54a508e5483786bd6ece73c5e3e3cbe` | `container/CA-BUNDLE.md`. The local file matches the hash and certificate count (**verified**). The header says "Certificate data from Mozilla as of: Tue Oct 20 13:32:40 2026 GMT." The upstream `cacert.pem.sha256` returned the same hash on 2026-10-24 (**verified**, read-only fetch). |
| Release binaries | `dist/v0.1.1/radichat-linux-amd64` and `radichat-linux-arm64`, built from source commit `b06dce63a2552d30d66ac4226b3ac957c28347fe` | `dist/v0.1.1/BUILD_INFO.txt` and `SHA256SUMS` (**verified** against the verifier output below). |

Binary build command recorded in `BUILD_INFO.txt` (**reported**; the binary build is not scripted in the repository):

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -trimpath -ldflags='-s -w' -o dist/v0.1.1/radichat-linux-amd64 .
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -buildvcs=false -trimpath -ldflags='-s -w' -o dist/v0.1.1/radichat-linux-arm64 .
```

The build toolchain for those binaries is `go1.27.1 linux/amd64`, per `BUILD_INFO.txt`.

### Procedure

Requirements: a Linux x86_64 host with `curl`, `tar`, `sha256sum`, and `python3`, and a kernel that permits the rootless BuildKit and RootlessKit setup used by the script. The script refuses any other host.

1. Check out the candidate commit and confirm a clean tree.
2. Build the two Linux binaries with the command above into `dist/v0.1.1/`, and write `dist/v0.1.1/SHA256SUMS` and `dist/v0.1.1/BUILD_INFO.txt` with the source commit.
3. Run `bash scripts/build-oci-v0.1.1.sh` (or `./scripts/build-oci-v0.1.1.sh`). The script:
   - checks the SHA-256 of the two Linux binaries against `SHA256SUMS`;
   - checks the CA bundle hash;
   - downloads and hash-checks the pinned BuildKit and RootlessKit tarballs when they are not already present in `dist/oci-toolchain/`;
   - assembles `dist/oci-context/` with exactly `Containerfile`, `.dockerignore`, `ca-certificates.crt`, and the two Linux binaries;
   - runs two independent rootless BuildKit builds, each with a fresh state root, `--opt platform=linux/amd64,linux/arm64`, `SOURCE_COMMIT` from `BUILD_INFO.txt`, `SOURCE_DATE_EPOCH=0`, and an OCI gzip exporter with `rewrite-timestamp=true` and `compatibility-version=30`;
   - compares the two archives with `cmp`;
   - runs `scripts/verify_oci.py` on the first archive.
4. Confirm the outputs in `dist/v0.1.1/`: `radichat-v0.1.1.oci.tar`, `radichat-v0.1.1.repeat.oci.tar`, `oci-verification.json`, `oci-archive.sha256`, and `oci-build.txt`.
5. Run the verifier unit tests from the repository root (not with `-I`, because the test imports `scripts.verify_oci`):

   ```sh
   python3 -m unittest scripts.test_verify_oci -v
   ```

6. To re-check the archive offline:

   ```sh
   python3 -I scripts/verify_oci.py \
     --archive dist/v0.1.1/radichat-v0.1.1.oci.tar \
     --amd64 dist/v0.1.1/radichat-linux-amd64 \
     --arm64 dist/v0.1.1/radichat-linux-arm64 \
     --ca-bundle container/ca-certificates.crt \
     --ca-sha256 796f1389bfea13c0ed78f663b765baaff54a508e5483786bd6ece73c5e3e3cbe \
     --revision b06dce63a2552d30d66ac4226b3ac957c28347fe
   ```

### Local artifacts, 2026-10-24

- **Verified:** the two archives are byte-identical. Both are 5,874,688 bytes with SHA-256 `38651f5a5dbd3d93b44cdb8c1353f63e0c7044af3fd6afb5bc65e209eb4ba493`.
- **Verified:** `scripts/verify_oci.py` exits 0 on the archive, with these results:
  - OCI index digest `sha256:10e843853f975998159e6110aeb55c5b9e1ec5db8edfd2f9d1ad59a9fc112505`. This is the digest of the local archive's multi-platform index. It is **not a registry digest** and has not been pushed.
  - amd64 manifest `sha256:b7ac2e2b79ea8d0a6a10827700bbe7659d85a0e4c2918e482e514f2f81a1a8e3`, two layers, binary SHA-256 `e9c5250904b5a7497d1a26b8da23440ed8e944e9279cb7ef1f5ca356997ce9b5`.
  - arm64 manifest `sha256:807c9ef798bc83053e43253c4e177da6fd5885768ff23ad046ce3498ff46d9c1`, two layers, binary SHA-256 `cf7a80f83dade0655847ee7f491d7ddc3a4f58daf0a02c67b8d0e4a10e110490`.
  - Both platforms carry the pinned CA bundle, the expected config, user `65532:65532`, working directory `/config`, entrypoint `/radichat`, and the default `CMD ["-config","/config/radichat.json"]`.
- **Verified:** the `v0.1.1` tag does not exist on the registry (`GET …/manifests/v0.1.1` returns 404). `latest` and `v0.1.0` still resolve to `sha256:2d8648b9…98ecf`.
- **Verified:** the `unittest` suite passes 7 of 7 when run from the repository root.
- **Not verified:** runtime execution of the image or of its binaries in a container. No local OCI runtime is installed, so the image is verified only structurally and by its config and payload. The arm64 binary is not runtime-tested.

The build script produced `oci-verification.json`, `oci-archive.sha256`, and `oci-build.txt` during the final successful run. It builds twice from fresh BuildKit state roots and requires the archives to compare byte-for-byte before writing those records.

### Open items for v0.1.1

- **Labels differ from v0.1.0.** The v0.1.1 image has only `version` and `revision` labels. The v0.1.0 image also carried title, description, source, URL, and license labels. The verifier checks only the first two. Restoring the labels changes the image config and therefore the digest, so it needs an Owner decision and a rebuild.
- **Timestamp inputs:** the build passes `SOURCE_DATE_EPOCH=0` and enables the OCI exporter's `rewrite-timestamp=true`. Two fresh-root builds produced byte-identical archives, and the verifier checks every blob digest. The resulting reproducibility evidence covers the checked-in procedure on this host.
- **The CA license is not stated in the bundle.** `container/CA-BUNDLE.md` says the bundle is "MPL-2.0 (same license as Mozilla's source certificate store)." The bundle file itself contains no license text. The MPL-2.0 statement is an external claim that was not verified in this session. Mozilla's root store and the curl.se page are the sources to check.
- **Ignored release notes are older than this local build.** `dist/v0.1.1/BUILD_INFO.txt`, `PUBLICATION_PLAN.md`, and `RELEASE_NOTES.md` predate the local archive and still describe the earlier state. They are ignored by Git and are not public evidence; use this tracked document and the checked-in verifier when reviewing the local image.
- **Runtime gap.** No container runtime has run the image. Runtime behavior of `linux/arm64` has not been tested on any host.
- **No registry digest.** Any digest recorded before a push is local only. Once the image is pushed, record the registry digest and verify it against the archive's index digest.
- **Watcher and Owner gates.** `tasks/release-oci-v0.1.1.md` has an empty execution record. Independent Watcher verification of the final archive and the repository changes has not been run, and no Owner acceptance has been given.

## Publication

Publication is a separate action that needs its own Owner authorization. The release procedure does not include any tag, release, or push step. After Owner authorization, push by tag, confirm the registry digest matches the archive's index digest, and do not change `latest` unless the Owner asks for it.
