# RadiChat v0.1.1 OCI packaging — Owner-authorized release preparation

## Authorization

On 2026-10-24, the Owner authorized implementation of the minimal reproducible OCI packaging procedure proposed in `docs/releasing.md`, local multi-platform image builds and verification, documentation corrections, and necessary preparation commits. This is release preparation outside the accepted Stage 4 / Phase 6 contract; it does not authorize a new product phase, tag, GitHub release, GHCR push, changes to the v0.1.0 image, or product-code changes.

## Goal

Produce a reproducible local OCI image archive for the existing RadiChat v0.1.1 Linux amd64 and arm64 release binaries, preserve the `/config/radichat.json` default, and provide durable build and verification instructions for Owner release review.

## Scope

- Add a checked-in `FROM scratch` Containerfile.
- Stage only the two existing static release binaries and a pinned public CA bundle into the build context.
- Pin and record builder inputs and CA provenance.
- Build both target platforms to a local OCI archive and reproduce the archive with an independent second build.
- Inspect archive index, manifests, image config, layer contents, binary checksums, file modes and ownership, and non-root configuration.
- Have the Writer finalize release documentation and the independent Watcher verify the frozen artifacts and evidence.

## Exclusions

- RadiChat application code or accepted Phase 6 contract changes.
- Any mutation, retag, or republication of `ghcr.io/radilabs/radichat:v0.1.0`.
- v0.1.1 tag creation, GitHub release, binary upload, GHCR push, or moving `latest`.
- Adding credentials, development tools, repository files, `.git/`, `.factory/`, local configuration, or reports to the image build context.

## Acceptance criteria

1. Containerfile uses `FROM scratch`, contains the existing Linux binaries and pinned CA bundle only, and preserves entrypoint, default config argument, working directory, TLS bundle path, labels, and UID/GID `65532:65532`.
2. A checked-in restricted-context recipe and reproducible commands document exact builder version/checksum, CA bundle source/version/checksum, input binary hashes, output OCI archive digest, and verification steps.
3. Two independent local builds from the same pinned inputs produce byte-identical OCI archives.
4. Offline archive inspection confirms exactly linux/amd64 and linux/arm64, verifies all descriptor/blob digests, each binary hash matches its release checksum, the expected CA bundle is present, no other payload files exist, file ownership/modes are correct, and image config has the expected non-root execution and default configuration settings.
5. No product source or Phase 6 contract changes, no v0.1.0 image mutation, and no tag/release/publication action occurs.
6. Writer documentation review is complete, and the independent Watcher returns explicit PASS/FAIL on the actual final archive, reproducibility evidence, and repository changes.

## Execution record

### Candidate and inputs

- Candidate source commit for the released binaries: `b06dce63a2552d30d66ac4226b3ac957c28347fe` (recorded in the existing ignored `dist/v0.1.1/BUILD_INFO.txt`).
- Builder: BuildKit `v0.34.0`, archive SHA-256 `14813b1fb2a40082114e6d386c6fc5e7dd5298176ce7ee1a18bc33e73e109ac3`; RootlessKit `v3.3.0`, archive SHA-256 `0e757c8f80510d764df8c9a5f169805580b48d02329f0a200900e261ecec27ed`.
- CA bundle SHA-256: `796f1389bfea13c0ed78f663b765baaff54a508e5483786bd6ece73c5e3e3cbe`.
- Release binary SHA-256 values: amd64 `e9c5250904b5a7497d1a26b8da23440ed8e944e9279cb7ef1f5ca356997ce9b5`; arm64 `cf7a80f83dade0655847ee7f491d7ddc3a4f58daf0a02c67b8d0e4a10e110490`.

### Commands and results

- `bash scripts/build-oci-v0.1.1.sh` — PASS. Builds both platforms twice in separate fresh BuildKit state roots and requires byte-identical OCI archives.
- Archive SHA-256 for both outputs: `38651f5a5dbd3d93b44cdb8c1353f63e0c7044af3fd6afb5bc65e209eb4ba493` (5,874,688 bytes each).
- Local multi-platform image index digest: `sha256:10e843853f975998159e6110aeb55c5b9e1ec5db8edfd2f9d1ad59a9fc112505`. This is an OCI archive index digest, not a registry digest.
- Platform manifest digests: amd64 `sha256:b7ac2e2b79ea8d0a6a10827700bbe7659d85a0e4c2918e482e514f2f81a1a8e3`; arm64 `sha256:807c9ef798bc83053e43253c4e177da6fd5885768ff23ad046ce3498ff46d9c1`.
- `python3 -I scripts/verify_oci.py --archive dist/v0.1.1/radichat-v0.1.1.oci.tar --amd64 dist/v0.1.1/radichat-linux-amd64 --arm64 dist/v0.1.1/radichat-linux-arm64 --ca-bundle container/ca-certificates.crt --ca-sha256 796f1389bfea13c0ed78f663b765baaff54a508e5483786bd6ece73c5e3e3cbe --revision b06dce63a2552d30d66ac4226b3ac957c28347fe` — PASS. Verifies both platforms, all OCI descriptor/blob hashes, exact payload paths, binary and CA bytes, ownership and modes, and non-root image config including `/config/radichat.json` default.
- `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest scripts.test_verify_oci -v` — PASS, 7 tests.
- `bash -n scripts/build-oci-v0.1.1.sh` and `git diff --check` — PASS.
- Context check — PASS. The generated context contains only the Containerfile, `.dockerignore`, two release binaries, and CA bundle. The OCI payload contains only `/radichat`, the CA bundle, and required directories.

### Role outcomes and limits

- Writer completed the release documentation audit. Updated `docs/releasing.md`, `docs/container.md`, `docs/kubernetes.md`, and `docs/README.md`; no product code or Phase 6 contract changed.
- Independent Watcher verification: PASS on packaging candidate `00b2c99c4a1f3b7619e961e778cecf540ad92ffa` for the actual archive, manifest/config/payload checks, pinned builder and CA inputs, and independent byte-for-byte rebuild. The Watcher identified one stale statement in `docs/kubernetes.md`; it was corrected in `be18552a999adcd94c0e0b691a630538efc67481`. Fresh independent Watcher follow-up on `be18552a999adcd94c0e0b691a630538efc67481`: PASS for that documentation correction and confirmation that packaging files and actual archives are unchanged. Detailed evidence is in the ignored local Watcher report `.factory/reports/release-prep-v0.1.1-watcher-02.md` and in tracked build instructions/results above.
- Runtime smoke: not performed because this host has no OCI runtime. Image non-root execution is verified from config and layer ownership/modes; actual process execution is not claimed. The arm64 binary was not runtime-tested.
- The v0.1.1 image has only `version` and `revision` labels; the additional descriptive labels on v0.1.0 are not present. The documented configuration and execution behavior, including the default config path, is preserved.
- No v0.1.1 tag, GitHub release, binary upload, GHCR push, or modification of the v0.1.0 image occurred. No registry digest exists yet.

The release remains unpublished until separately authorized.
