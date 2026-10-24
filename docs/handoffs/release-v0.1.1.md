# v0.1.1 release record

Date: 2026-10-24
Status: **published**

This is a release record, not a phase handoff. It records the published `v0.1.1` binaries and container image, the procedure that produced them, and the verification and limits that apply. No phase is opened or closed by it. Further tags, releases, or image publication require separate Owner authorization.

## Published release

- **Source tag:** `v0.1.1`, an annotated tag that peels to `6645988e354076fede28070686b23667d0ff89ac`. `origin/main` is at the same commit at publication.
- **GitHub release:** https://github.com/radilabs/radichat/releases/tag/v0.1.1. Not a draft and not a prerelease. Assets: `radichat-linux-amd64`, `radichat-linux-arm64`, `radichat-darwin-amd64`, `radichat-darwin-arm64`, and `SHA256SUMS`.
- **Binary source commit:** `b06dce63a2552d30d66ac4226b3ac957c28347fe`. No Go source or module file changed between that commit and the release commit. The commits after it are documentation and packaging only.
- **Build:** Go `1.27.1`, `CGO_ENABLED=0`, `-buildvcs=false`, `-trimpath`, `-ldflags='-s -w'`. Each target was built twice, and the local validation record reports the builds compared byte-for-byte.

### Tag immutability

Tag `v0.1.1` intentionally remains at `6645988e354076fede28070686b23667d0ff89ac`. It was not moved or recreated, and the published artifacts were not changed. That commit's `examples/kubernetes/job.json` and `docs/kubernetes.md` still contain the pre-publication placeholder `REPLACE_WITH_V0_1_1_DIGEST`.

The post-publication correction is carried on `main` as a follow-up, without moving or recreating `v0.1.1`. Current `main` contains the corrected digest in `examples/kubernetes/job.json`, while the immutable `v0.1.1` tag snapshot retains its placeholder. Readers who want the corrected example should use current `main`, not the tag.

### Binary checksums

These match the published `SHA256SUMS`. The published assets are byte-identical to the local build, and `sha256sum -c` passes on the downloaded assets.

| Artifact | Bytes | SHA-256 |
| --- | ---: | --- |
| `radichat-linux-amd64` | 6,860,960 | `e9c5250904b5a7497d1a26b8da23440ed8e944e9279cb7ef1f5ca356997ce9b5` |
| `radichat-linux-arm64` | 6,357,152 | `cf7a80f83dade0655847ee7f491d7ddc3a4f58daf0a02c67b8d0e4a10e110490` |
| `radichat-darwin-amd64` | 7,020,528 | `4f354a5e85975acaa5f6f19c957fa6e00cf74406cc7efb2b0167c7fe2f4d5422` |
| `radichat-darwin-arm64` | 6,488,834 | `bb0937e5b8668f6dc6ffd172a882ab52affa7dd1d4ffdaa16c337713f3592975` |

The macOS binaries are unsigned and not notarized. No macOS runtime support is claimed.

## Container image

- **Tag:** `ghcr.io/radilabs/radichat:v0.1.1`
- **Immutable digest (OCI index):** `sha256:10e843853f975998159e6110aeb55c5b9e1ec5db8edfd2f9d1ad59a9fc112505`. This is the registry digest. It matches the nested index digest of the local BuildKit archive.
- **Platform manifests:** `linux/amd64` `sha256:b7ac2e2b79ea8d0a6a10827700bbe7659d85a0e4c2918e482e514f2f81a1a8e3`; `linux/arm64` `sha256:807c9ef798bc83053e43253c4e177da6fd5885768ff23ad046ce3498ff46d9c1`.
- **Embedded binaries:** `linux/amd64` `e9c5250904b5a7497d1a26b8da23440ed8e944e9279cb7ef1f5ca356997ce9b5`; `linux/arm64` `cf7a80f83dade0655847ee7f491d7ddc3a4f58daf0a02c67b8d0e4a10e110490`.
- **Embedded CA bundle:** `796f1389bfea13c0ed78f663b765baaff54a508e5483786bd6ece73c5e3e3cbe` (curl.se Mozilla snapshot, 118 certificates, 200,116 bytes).
- **Image revision label:** `b06dce63a2552d30d66ac4226b3ac957c28347fe`.
- **Runtime:** the image was not executed in a container. No OCI runtime was available. Verification is structural, config, and payload only.
- **Image shape:** the same non-root user (`65532:65532`), working directory (`/config`), entrypoint (`/radichat`), and default `CMD ["-config","/config/radichat.json"]` as `v0.1.0`. Only `version` and `revision` labels are present; the `v0.1.0` descriptive labels are not.
- **`latest`:** not moved. It still resolves to `sha256:2d8648b9f9d3c60273a7e432d0341a93e8e8fbcfe4c172617ddce3a07df98ecf`, the same digest as `v0.1.0`. `v0.1.0` is unchanged.

## Procedure

Build inputs are pinned in `scripts/build-oci-v0.1.1.sh`:

- BuildKit `v0.34.0` (`linux-amd64` tarball, SHA-256 `14813b1fb2a40082114e6d386c6fc5e7dd5298176ce7ee1a18bc33e73e109ac3`).
- RootlessKit `v3.3.0` (`x86_64` tarball, SHA-256 `0e757c8f80510d764df8c9a5f169805580b48d02329f0a200900e261ecec27ed`).
- CA bundle: SHA-256 `796f1389bfea13c0ed78f663b765baaff54a508e5483786bd6ece73c5e3e3cbe`. Provenance is in `container/CA-BUNDLE.md`.

The local BuildKit build ran twice from fresh state roots, and the two archives were byte-identical: 5,874,688 bytes, SHA-256 `38651f5a5dbd3d93b44cdb8c1353f63e0c7044af3fd6afb5bc65e209eb4ba493`. `scripts/verify_oci.py` passed on that archive.

Publication used ORAS `v1.3.5`:

- The `linux/amd64` ORAS archive was checked against the official checksum file. SHA-256 `f48a73ea6d026a862c3919fc773da9a8c86fc259c2c737d5baed3b8b9614ad22`.
- The BuildKit OCI archive's outer `index.json` descriptor lacked `org.opencontainers.image.ref.name`, which the registry copy needs for a tag. A temporary local OCI-layout copy was made. Only that one outer descriptor annotation was added. No blob changed, so the nested multi-platform index and its digest are preserved. This was an annotation adapter, not a rebuild.
- The adapted layout was copied to `ghcr.io/radilabs/radichat:v0.1.1`. The exact ORAS invocation was not captured in the repository; its transcript is the authority for the flags used.

## Verification

- **Published assets:** the downloaded assets match `SHA256SUMS`, and the published `SHA256SUMS` is byte-identical to the local file.
- **Registry:** `v0.1.1` resolves to the index digest above. The platform manifests match the values listed.
- **Registry-pulled copy:** `scripts/verify_oci.py` exits 0 against the accepted Linux binaries and the pinned CA bundle. The verification copy's SHA-256 is `793c59cf48071e9854b97cdd6cc76d6afc633256d2f819507e9e6b33b35799a2`. It is scratch evidence, not a release artifact.
- **Tests:** `python3 -m unittest scripts.test_verify_oci -v` passes 7 of 7 from the repository root.
- **Independent review:** the Watcher record in commit `6645988` (`tasks/release-oci-v0.1.1.md`) covers the packaging candidate `00b2c99c4a1f3b7619e961e778cecf540ad92ffa` and the follow-up candidate `be18552a999adcd94c0e0b691a630538efc67481`. This record does not restate those verdicts beyond what that record says.

## Known limitations

- The `v0.1.1` image has not been run. The `linux/arm64` binary and image, and all darwin binaries, have not been runtime-tested.
- The image has reduced labels compared with `v0.1.0`. Restoring them would change the digest and needs a rebuild and a new release decision.
- `SOURCE_DATE_EPOCH=0` is passed as a build argument that the Containerfile does not declare. Timestamp normalization is assumed from the exporter's `rewrite-timestamp` option. This is not separately verified.
- The MPL-2.0 license claim for the CA bundle is external to the bundle file. It was not verified here.
- The exact ORAS invocation is not recorded in the repository.
- Runtime behavior limits from the batch documentation apply: SIGINT and SIGTERM cannot interrupt a process blocked opening a FIFO for `-input`, and an unknown `/command` as the last input line exits 1 at EOF.
- The Kubernetes example is documentation and an example only. It needs an operator-provided CA trusted by the container for its HTTPS endpoint, and it has not been run in a cluster.

## Documentation updated by publication

- `docs/releasing.md`, `docs/container.md`, `docs/kubernetes.md`, `docs/README.md`, `README.md`, `TASKS.md`, `tasks/README.md`, `tasks/release-oci-v0.1.1.md`, `examples/kubernetes/job.json`, and `examples/kubernetes/kubernetes_test.go`.
- `docs/handoffs/phase-6.md` and the Phase 0–5 handoffs are accepted historical records and were not changed.
- The GitHub release body was not edited by this documentation pass. `dist/v0.1.1/RELEASE_NOTES.md` is ignored local publication material and is outside the repository.

## Next

No phase is authorized. Any further tag, release, image publication, or `latest` move needs separate Owner authorization.
