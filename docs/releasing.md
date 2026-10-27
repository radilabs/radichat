# Releasing RadiChat

This document records four things: the evidence for the published `v0.1.0` binaries and container image, the checked-in OCI packaging procedure for `v0.1.1`, the publication of `v0.1.1` (tag, GitHub release, and GHCR and Docker Hub images), and the open items. The durable release record is [v0.1.1 release record](handoffs/release-v0.1.1.md). Any further tag, release, or image publication requires separate Owner authorization. `v0.1.0` and `latest` are not changed by this document.

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

## v0.1.1 OCI packaging and publication

This section describes the checked-in procedure, the publication steps, and the verified published state. Release publication for `v0.1.1` has been completed. The tag, GitHub release, and GHCR image are listed under "Published state" below.

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

### Publication procedure (as executed for v0.1.1)

The image was published from the local BuildKit archive, not rebuilt for publication. The steps are:

1. **Pin ORAS.** Use ORAS `v1.3.5`, `linux/amd64` archive `oras_1.3.5_linux_amd64.tar.gz`, SHA-256 `f48a73ea6d026a862c3919fc773da9a8c86fc259c2c737d5baed3b8b9614ad22`. The hash matches the official `oras_1.3.5_checksums.txt` for that release (**verified**). Check the archive against that hash before extracting it.
2. **Apply the outer ref-name adapter.** The BuildKit OCI exporter's outer `index.json` descriptor does not carry `org.opencontainers.image.ref.name`, which the registry copy needs for a tag. Make a temporary local OCI-layout copy of `dist/v0.1.1/radichat-v0.1.1.oci.tar`, and add that one annotation (the tag name `v0.1.1`) to the outer descriptor only. Do not change any blob. The nested multi-platform image index, its manifests, configs, and layers keep their digests. This is a descriptor annotation, not a rebuild, and the image content is unchanged.
3. **Copy to GHCR with ORAS.** With a temporary ORAS registry-config file that authenticates to GHCR, run:

   ```sh
   oras cp --no-tty --from-oci-layout --to-registry-config "$auth_file" \
     /path/to/adapted-layout:v0.1.1 \
     ghcr.io/radilabs/radichat:v0.1.1
   ```

   This is the command shape used for the publication. Keep the auth file outside the repository and remove it after use. Copy only to the `v0.1.1` tag. Do not push or retag `latest`.
4. **Verify the registry copy.** The registry index digest must equal the nested index digest of the archive, `sha256:10e843853f975998159e6110aeb55c5b9e1ec5db8edfd2f9d1ad59a9fc112505`. Pull the image back as an OCI layout and run `scripts/verify_oci.py` against the accepted Linux binaries and the pinned CA bundle. The check must exit 0.

Image execution was not tested. No OCI runtime was available, so the published image has structural, config, and payload verification only.

### Docker Hub publication (2026-10-27)

Owner authorization was granted to publish the existing verified OCI archive to Docker Hub. No image rebuild was performed. Skopeo `1.24.0` copied the archive with `--all --preserve-digests`:

```sh
skopeo copy --all --preserve-digests \
  oci-archive:dist/v0.1.1/radichat-v0.1.1.oci.tar \
  docker://docker.io/naorw/radichat:v0.1.1
```

Skopeo used the configured Docker Hub authentication. Keep credentials in Skopeo's credential store or an external auth file; do not place them in the repository or command history. Only the `v0.1.1` tag was targeted.

The registry copy was fetched back as an OCI archive with Skopeo and checked with `scripts/verify_oci.py`. The registry index digest remained `sha256:10e843853f975998159e6110aeb55c5b9e1ec5db8edfd2f9d1ad59a9fc112505`, matching the accepted archive and GHCR. The amd64 and arm64 platform manifests, image configs, layers, binary hashes, and CA bundle hash all matched. The pulled-back tar archive has its own transport archive hash; that is not the registry digest.

### Published state

Verified on 2026-10-24 after the GitHub and GHCR publication. Docker Hub publication and verification were added on 2026-10-27:

- **Source tag:** annotated tag `v0.1.1` peels to `6645988e354076fede28070686b23667d0ff89ac`. `origin/main` is at the same commit.
- **GitHub release:** `https://github.com/radilabs/radichat/releases/tag/v0.1.1`. Not a draft and not a prerelease. It has four binary assets and `SHA256SUMS`. The published `SHA256SUMS` is byte-identical to the local file. The four binaries are byte-identical to the local build, and their SHA-256 values match `SHA256SUMS`.
- **GHCR tag `v0.1.1`:** OCI index digest `sha256:10e843853f975998159e6110aeb55c5b9e1ec5db8edfd2f9d1ad59a9fc112505`. This is the immutable digest, and it matches the local archive's nested index digest.
- **Docker Hub tag `v0.1.1`:** `docker.io/naorw/radichat:v0.1.1`, OCI index digest `sha256:10e843853f975998159e6110aeb55c5b9e1ec5db8edfd2f9d1ad59a9fc112505`. Published with Skopeo `1.24.0` from the existing verified archive using `--all --preserve-digests`; no rebuild was performed.
- **Platform manifests:** `linux/amd64` `sha256:b7ac2e2b79ea8d0a6a10827700bbe7659d85a0e4c2918e482e514f2f81a1a8e3`; `linux/arm64` `sha256:807c9ef798bc83053e43253c4e177da6fd5885768ff23ad046ce3498ff46d9c1`.
- **Docker Hub pull-back verification:** `skopeo copy --all --preserve-digests docker://docker.io/naorw/radichat:v0.1.1 oci-archive:/tmp/radichat-dockerhub-v0.1.1.oci.tar:v0.1.1`, followed by the `scripts/verify_oci.py` check against the released Linux binaries and pinned CA bundle, exited 0. The copied archive contained exactly `linux/amd64` and `linux/arm64` and reported binary hashes `e9c5250904b5a7497d1a26b8da23440ed8e944e9279cb7ef1f5ca356997ce9b5` and `cf7a80f83dade0655847ee7f491d7ddc3a4f58daf0a02c67b8d0e4a10e110490`, respectively.
- **Registry-pulled contents:** `scripts/verify_oci.py` exits 0 on the pulled copy. It reports the embedded `linux/amd64` binary `e9c5250904b5a7497d1a26b8da23440ed8e944e9279cb7ef1f5ca356997ce9b5`, the embedded `linux/arm64` binary `cf7a80f83dade0655847ee7f491d7ddc3a4f58daf0a02c67b8d0e4a10e110490`, and the embedded CA bundle `796f1389bfea13c0ed78f663b765baaff54a508e5483786bd6ece73c5e3e3cbe`. The verification copy's SHA-256 is `793c59cf48071e9854b97cdd6cc76d6afc633256d2f819507e9e6b33b35799a2`. It is scratch evidence, not a release artifact.
- **Image revision label:** `b06dce63a2552d30d66ac4226b3ac957c28347fe`, the source commit of the binaries. The tag commit `6645988` differs only in documentation and packaging files; no Go source or module file changed between the two.
- **`latest` was not moved.** It still resolves to `sha256:2d8648b9f9d3c60273a7e432d0341a93e8e8fbcfe4c172617ddce3a07df98ecf`, the same digest as `v0.1.0`. `v0.1.0` is unchanged.
- **Execution:** not tested. The published image has not been run in a container.

### Local build record, 2026-10-24 (before publication)

- **Verified:** the two archives are byte-identical. Both are 5,874,688 bytes with SHA-256 `38651f5a5dbd3d93b44cdb8c1353f63e0c7044af3fd6afb5bc65e209eb4ba493`.
- **Verified:** `scripts/verify_oci.py` exits 0 on the archive, with these results:
  - OCI index digest `sha256:10e843853f975998159e6110aeb55c5b9e1ec5db8edfd2f9d1ad59a9fc112505`. This is the digest of the local archive's multi-platform index. At the time of this pre-publication check it was **not a registry digest** and had not been pushed. That is historical; the same digest is now the published GHCR index digest (see "Published state" above).
  - amd64 manifest `sha256:b7ac2e2b79ea8d0a6a10827700bbe7659d85a0e4c2918e482e514f2f81a1a8e3`, two layers, binary SHA-256 `e9c5250904b5a7497d1a26b8da23440ed8e944e9279cb7ef1f5ca356997ce9b5`.
  - arm64 manifest `sha256:807c9ef798bc83053e43253c4e177da6fd5885768ff23ad046ce3498ff46d9c1`, two layers, binary SHA-256 `cf7a80f83dade0655847ee7f491d7ddc3a4f58daf0a02c67b8d0e4a10e110490`.
  - Both platforms carry the pinned CA bundle, the expected config, user `65532:65532`, working directory `/config`, entrypoint `/radichat`, and the default `CMD ["-config","/config/radichat.json"]`.
- **Verified at the time of this pre-publication check:** the `v0.1.1` tag returned 404 from the registry (`GET …/manifests/v0.1.1`), and `latest` and `v0.1.0` resolved to `sha256:2d8648b9…98ecf`. These statements are historical. The `v0.1.1` tag now exists and resolves to the published digest; `latest` and `v0.1.0` are unchanged. See "Published state" above.
- **Verified:** the `unittest` suite passes 7 of 7 when run from the repository root.
- **Not verified:** runtime execution of the image or of its binaries in a container. No local OCI runtime is installed, so the image is verified only structurally and by its config and payload. The arm64 binary is not runtime-tested.

The build script produced `oci-verification.json`, `oci-archive.sha256`, and `oci-build.txt` during the final successful run. It builds twice from fresh BuildKit state roots and requires the archives to compare byte-for-byte before writing those records.

### Open items for v0.1.1

- **Labels differ from v0.1.0.** The v0.1.1 image has only `version` and `revision` labels. The v0.1.0 image also carried title, description, source, URL, and license labels. The verifier checks only the first two. Restoring the labels changes the image config and therefore the digest, so it needs an Owner decision and a rebuild.
- **Timestamp inputs:** the build passes `SOURCE_DATE_EPOCH=0` and enables the OCI exporter's `rewrite-timestamp=true`. Two fresh-root builds produced byte-identical archives, and the verifier checks every blob digest. The resulting reproducibility evidence covers the checked-in procedure on this host.
- **The CA license is not stated in the bundle.** `container/CA-BUNDLE.md` says the bundle is "MPL-2.0 (same license as Mozilla's source certificate store)." The bundle file itself contains no license text. The MPL-2.0 statement is an external claim that was not verified in this session. Mozilla's root store and the curl.se page are the sources to check.
- **Ignored release files.** `dist/v0.1.1/RELEASE_NOTES.md` was updated for the published state. `BUILD_INFO.txt` and `PUBLICATION_PLAN.md` still predate publication. They are ignored by Git and are not public evidence. Use this tracked document and the release record instead.
- **Runtime gap.** No container runtime has run the published image. Runtime behavior of `linux/arm64` has not been tested on any host.
- **Registry digest recorded.** The GHCR digest is in "Published state" above. Digests recorded before the push were local only.
- **Gates and evidence.** The Owner authorized and performed the `v0.1.1` publication on 2026-10-24. The Watcher verification records are in `tasks/release-oci-v0.1.1.md` and in commit `6645988`. They cover packaging candidate `00b2c99c4a1f3b7619e961e778cecf540ad92ffa` and the follow-up candidate `be18552a999adcd94c0e0b691a630538efc67481`. This document does not restate those verdicts beyond what those records say.
- **Kubernetes manifest.** `examples/kubernetes/job.json` uses the published immutable digest. Its test asserts the exact image reference. The example has not been run in a cluster.

## Publication

The `v0.1.1` publication is complete. Its state is recorded under "Published state" above and in the [v0.1.1 release record](handoffs/release-v0.1.1.md).

Any further tag, GitHub release, or image publication needs its own Owner authorization. Do not move `latest` unless the Owner asks for it. Verify any new registry digest against the archive's nested index digest before recording it.
