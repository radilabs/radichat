#!/usr/bin/env bash
set -euo pipefail

readonly ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly VERSION="v0.1.1"
readonly DIST="$ROOT/dist/$VERSION"
readonly TOOL_DIR="$ROOT/dist/oci-toolchain"
readonly CONTEXT="$ROOT/dist/oci-context"
readonly CA_SHA256="796f1389bfea13c0ed78f663b765baaff54a508e5483786bd6ece73c5e3e3cbe"
readonly BUILDKIT_VERSION="0.34.0"
readonly BUILDKIT_ARCHIVE_SHA256="14813b1fb2a40082114e6d386c6fc5e7dd5298176ce7ee1a18bc33e73e109ac3"
readonly ROOTLESSKIT_VERSION="3.3.0"
readonly ROOTLESSKIT_ARCHIVE_SHA256="0e757c8f80510d764df8c9a5f169805580b48d02329f0a200900e261ecec27ed"

fail() { printf 'error: %s\n' "$*" >&2; exit 1; }

[[ "$(uname -s)" == Linux && "$(uname -m)" == x86_64 ]] || fail "run this pinned local builder on linux/amd64"
for tool in curl tar sha256sum python3; do command -v "$tool" >/dev/null || fail "required host tool not found: $tool"; done
[[ -f "$DIST/radichat-linux-amd64" && -f "$DIST/radichat-linux-arm64" ]] || fail "missing v0.1.1 Linux release binaries in $DIST"
[[ -f "$DIST/SHA256SUMS" && -f "$DIST/BUILD_INFO.txt" ]] || fail "missing release checksum or build information"
(
  cd "$DIST"
  grep -E ' radichat-linux-(amd64|arm64)$' SHA256SUMS | sha256sum -c -
)
echo "$CA_SHA256  $ROOT/container/ca-certificates.crt" | sha256sum -c - >/dev/null || fail "CA bundle checksum mismatch"

readonly SOURCE_COMMIT="$(sed -n 's/^Source commit: //p' "$DIST/BUILD_INFO.txt")"
[[ "$SOURCE_COMMIT" =~ ^[0-9a-f]{40}$ ]] || fail "BUILD_INFO.txt must identify the 40-character binary source commit"

mkdir -p "$TOOL_DIR" "$TOOL_DIR/buildkit" "$TOOL_DIR/rootlesskit"
readonly BUILDKIT_TARBALL="$TOOL_DIR/buildkit-v${BUILDKIT_VERSION}.linux-amd64.tar.gz"
readonly ROOTLESSKIT_TARBALL="$TOOL_DIR/rootlesskit-v${ROOTLESSKIT_VERSION}.x86_64.tar.gz"
[[ -f "$BUILDKIT_TARBALL" ]] || curl -fsSL "https://github.com/moby/buildkit/releases/download/v${BUILDKIT_VERSION}/buildkit-v${BUILDKIT_VERSION}.linux-amd64.tar.gz" -o "$BUILDKIT_TARBALL"
echo "$BUILDKIT_ARCHIVE_SHA256  $BUILDKIT_TARBALL" | sha256sum -c -
python3 -c 'import shutil,sys; shutil.rmtree(sys.argv[1], ignore_errors=True)' "$TOOL_DIR/buildkit"
mkdir -p "$TOOL_DIR/buildkit"
tar -xzf "$BUILDKIT_TARBALL" -C "$TOOL_DIR/buildkit"

[[ -f "$ROOTLESSKIT_TARBALL" ]] || curl -fsSL "https://github.com/rootless-containers/rootlesskit/releases/download/v${ROOTLESSKIT_VERSION}/rootlesskit-x86_64.tar.gz" -o "$ROOTLESSKIT_TARBALL"
echo "$ROOTLESSKIT_ARCHIVE_SHA256  $ROOTLESSKIT_TARBALL" | sha256sum -c -
python3 -c 'import shutil,sys; shutil.rmtree(sys.argv[1], ignore_errors=True)' "$TOOL_DIR/rootlesskit"
mkdir -p "$TOOL_DIR/rootlesskit"
tar -xzf "$ROOTLESSKIT_TARBALL" -C "$TOOL_DIR/rootlesskit"
readonly BUILDKITD="$TOOL_DIR/buildkit/bin/buildkitd"
readonly BUILDKIT="$TOOL_DIR/buildkit/bin/buildctl"
readonly ROOTLESSKIT="$TOOL_DIR/rootlesskit/rootlesskit"
[[ -x "$BUILDKITD" && -x "$BUILDKIT" && -x "$ROOTLESSKIT" ]] || fail "pinned BuildKit or RootlessKit binary missing after extraction"
"$BUILDKITD" --version | grep -F "v${BUILDKIT_VERSION}" >/dev/null || fail "unexpected BuildKit version"
"$ROOTLESSKIT" --version | grep -F "${ROOTLESSKIT_VERSION}" >/dev/null || fail "unexpected RootlessKit version"
export PATH="$TOOL_DIR/buildkit/bin:$PATH"

rm -rf "$CONTEXT"
mkdir -m 700 -p "$CONTEXT"
install -m 0644 "$ROOT/Containerfile" "$CONTEXT/Containerfile"
install -m 0644 "$ROOT/container/.dockerignore" "$CONTEXT/.dockerignore"
install -m 0644 "$ROOT/container/ca-certificates.crt" "$CONTEXT/ca-certificates.crt"
install -m 0755 "$DIST/radichat-linux-amd64" "$CONTEXT/radichat-linux-amd64"
install -m 0755 "$DIST/radichat-linux-arm64" "$CONTEXT/radichat-linux-arm64"
expected_context=$'Containerfile\nca-certificates.crt\nradichat-linux-amd64\nradichat-linux-arm64'
actual_context="$(find "$CONTEXT" -mindepth 1 -maxdepth 1 -type f ! -name .dockerignore -printf '%f\n' | LC_ALL=C sort)"
[[ "$actual_context" == "$expected_context" ]] || fail "build context contains unexpected files: $actual_context"

build_once() {
  local ordinal="$1" output="$2" state socket log daemon_pid ready
  state="$(mktemp -d "$DIST/.buildkit-state.XXXXXX")"
  socket="/tmp/radichat-buildkit-$PPID-$ordinal.sock"
  log="$DIST/buildkitd-$ordinal.log"
  rm -f "$socket"
  "$ROOTLESSKIT" --net=host --copy-up=/etc "$BUILDKITD" \
    --rootless \
    --oci-worker-snapshotter=native \
    --oci-worker-no-process-sandbox \
    --root "$state" \
    --addr "unix://$socket" >"$log" 2>&1 &
  daemon_pid=$!
  cleanup_daemon() {
    kill "$daemon_pid" 2>/dev/null || true
    wait "$daemon_pid" 2>/dev/null || true
    rm -f "$socket"
    "$ROOTLESSKIT" --net=host --copy-up=/etc python3 -c \
      'import shutil,sys; shutil.rmtree(sys.argv[1], ignore_errors=True)' "$state" || true
  }
  trap cleanup_daemon RETURN
  ready=false
  for _ in $(seq 1 60); do
    if "$BUILDKIT" --addr "unix://$socket" debug workers >/dev/null 2>&1; then ready=true; break; fi
    if ! kill -0 "$daemon_pid" 2>/dev/null; then break; fi
    sleep 1
  done
  [[ "$ready" == true ]] || { cat "$log" >&2; fail "rootless BuildKit did not start"; }
  "$BUILDKIT" --addr "unix://$socket" build \
    --frontend dockerfile.v0 \
    --local context="$CONTEXT" \
    --local dockerfile="$CONTEXT" \
    --opt filename=Containerfile \
    --opt platform=linux/amd64,linux/arm64 \
    --opt build-arg:SOURCE_COMMIT="$SOURCE_COMMIT" \
    --opt build-arg:SOURCE_DATE_EPOCH=0 \
    --output "type=oci,dest=$output,compression=gzip,compression-level=6,force-compression=true,oci-mediatypes=true,rewrite-timestamp=true,compatibility-version=30"
  trap - RETURN
  cleanup_daemon
}

readonly ARCHIVE1="$DIST/radichat-v0.1.1.oci.tar"
readonly ARCHIVE2="$DIST/radichat-v0.1.1.repeat.oci.tar"
build_once 1 "$ARCHIVE1"
build_once 2 "$ARCHIVE2"
cmp -s "$ARCHIVE1" "$ARCHIVE2" || fail "independent clean-root builds did not produce byte-identical OCI archives"

python3 "$ROOT/scripts/verify_oci.py" \
  --archive "$ARCHIVE1" \
  --amd64 "$DIST/radichat-linux-amd64" \
  --arm64 "$DIST/radichat-linux-arm64" \
  --ca-bundle "$ROOT/container/ca-certificates.crt" \
  --ca-sha256 "$CA_SHA256" \
  --revision "$SOURCE_COMMIT" | tee "$DIST/oci-verification.json"
sha256sum "$ARCHIVE1" "$ARCHIVE2" | tee "$DIST/oci-archive.sha256"
printf 'BuildKit %s; RootlessKit %s; context entries: Containerfile, CA bundle, and the two Linux binaries.\n' \
  "$BUILDKIT_VERSION" "$ROOTLESSKIT_VERSION" | tee "$DIST/oci-build.txt"
printf 'Reproducible OCI build and verification passed. Index digest is shown above.\n'
