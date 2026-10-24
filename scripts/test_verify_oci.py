"""Standard-library tests for the local OCI archive verifier."""

import gzip
import hashlib
import io
import json
import tarfile
import tempfile
import unittest
from pathlib import Path

from scripts.verify_oci import VerificationError, verify_archive


REVISION = "0123456789abcdef0123456789abcdef01234567"
CA = b"pinned-ca-bundle\n"
BINARIES = {"amd64": b"amd64 executable", "arm64": b"arm64 executable"}
EXPECTED_PATHS = {
    "config/",
    "etc/",
    "etc/ssl/",
    "etc/ssl/certs/",
    "etc/ssl/certs/ca-certificates.crt",
    "radichat",
}


def sha(data):
    return "sha256:" + hashlib.sha256(data).hexdigest()


def layer_for(arch, extra_path=None, binary_mode=0o755, uid=65532, ca_bundle=CA):
    stream = io.BytesIO()
    with tarfile.open(fileobj=stream, mode="w", format=tarfile.USTAR_FORMAT) as tf:
        for name in sorted(EXPECTED_PATHS):
            data = BINARIES[arch] if name == "radichat" else ca_bundle if name.endswith(".crt") else b""
            info = tarfile.TarInfo(name)
            info.uid = info.gid = uid
            info.mode = binary_mode if name == "radichat" else 0o644 if data else 0o755
            info.size = len(data)
            if name.endswith("/"):
                info.type = tarfile.DIRTYPE
                info.size = 0
            tf.addfile(info, io.BytesIO(data) if data else None)
        if extra_path:
            info = tarfile.TarInfo(extra_path)
            info.uid = info.gid = 0
            info.mode = 0o644
            info.size = 0
            tf.addfile(info, io.BytesIO())
    return stream.getvalue()


def archive_bytes(extra_path=None, user="65532:65532", duplicate_arch=False, binary_mode=0o755, uid=65532, ca_bundle=CA):
    blobs = {}
    descriptors = []
    for arch in ("amd64", "arm64"):
        layer_tar = layer_for(arch, extra_path, binary_mode, uid, ca_bundle)
        compressed = gzip.compress(layer_tar, mtime=0)
        layer_digest = sha(compressed)
        config = {
            "architecture": arch,
            "os": "linux",
            "config": {
                "User": user,
                "WorkingDir": "/config",
                "Env": [
                    "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
                    "SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt",
                ],
                "Entrypoint": ["/radichat"],
                "Cmd": ["-config", "/config/radichat.json"],
                "Labels": {"version": "v0.1.1", "revision": REVISION},
            },
            "rootfs": {"type": "layers", "diff_ids": [sha(layer_tar)]},
            "history": [{"created": "1970-01-01T00:00:00Z", "created_by": "COPY"}],
        }
        config_bytes = json.dumps(config, sort_keys=True, separators=(",", ":")).encode()
        config_digest = sha(config_bytes)
        manifest = {
            "schemaVersion": 2,
            "mediaType": "application/vnd.oci.image.manifest.v1+json",
            "config": {"mediaType": "application/vnd.oci.image.config.v1+json", "digest": config_digest, "size": len(config_bytes)},
            "layers": [{"mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": layer_digest, "size": len(compressed)}],
        }
        manifest_bytes = json.dumps(manifest, sort_keys=True, separators=(",", ":")).encode()
        manifest_digest = sha(manifest_bytes)
        blobs[config_digest] = config_bytes
        blobs[layer_digest] = compressed
        blobs[manifest_digest] = manifest_bytes
        descriptors.append({
            "mediaType": "application/vnd.oci.image.manifest.v1+json",
            "digest": manifest_digest,
            "size": len(manifest_bytes),
            "platform": {"architecture": arch, "os": "linux"},
        })
    if duplicate_arch:
        descriptors[1]["platform"]["architecture"] = "amd64"
    image_index_bytes = json.dumps({
        "schemaVersion": 2,
        "mediaType": "application/vnd.oci.image.index.v1+json",
        "manifests": descriptors,
    }, sort_keys=True, separators=(",", ":")).encode()
    image_index_digest = sha(image_index_bytes)
    blobs[image_index_digest] = image_index_bytes
    index_bytes = json.dumps({
        "schemaVersion": 2,
        "mediaType": "application/vnd.oci.image.index.v1+json",
        "manifests": [{
            "mediaType": "application/vnd.oci.image.index.v1+json",
            "digest": image_index_digest,
            "size": len(image_index_bytes),
            "annotations": {"org.opencontainers.image.created": "1970-01-01T00:00:00Z"},
        }],
    }, sort_keys=True, separators=(",", ":")).encode()
    output = io.BytesIO()
    with tarfile.open(fileobj=output, mode="w") as archive:
        for name, data in [("oci-layout", b'{"imageLayoutVersion":"1.0.0"}'), ("index.json", index_bytes)]:
            info = tarfile.TarInfo(name)
            info.size = len(data)
            archive.addfile(info, io.BytesIO(data))
        for digest, data in blobs.items():
            name = "blobs/sha256/" + digest.split(":", 1)[1]
            info = tarfile.TarInfo(name)
            info.size = len(data)
            archive.addfile(info, io.BytesIO(data))
    return output.getvalue()


class VerifyOciTests(unittest.TestCase):
    def run_archive(self, payload):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "image.oci.tar"
            path.write_bytes(payload)
            return verify_archive(path, BINARIES, CA, hashlib.sha256(CA).hexdigest(), REVISION)

    def test_accepts_two_platforms_with_expected_config_and_contents(self):
        result = self.run_archive(archive_bytes())
        self.assertEqual(result["platforms"], ["linux/amd64", "linux/arm64"])
        self.assertEqual(len(result["index_digest"]), 71)
        self.assertEqual(set(result["manifests"]), {"amd64", "arm64"})

    def test_rejects_an_extra_payload_file(self):
        with self.assertRaisesRegex(VerificationError, "layer paths"):
            self.run_archive(archive_bytes(extra_path="etc/passwd"))

    def test_rejects_root_execution(self):
        with self.assertRaisesRegex(VerificationError, "User"):
            with tempfile.TemporaryDirectory() as tmp:
                path = Path(tmp) / "image.oci.tar"
                path.write_bytes(archive_bytes(user="0:0"))
                verify_archive(path, BINARIES, CA, hashlib.sha256(CA).hexdigest(), REVISION)

    def test_rejects_non_executable_binary(self):
        with self.assertRaisesRegex(VerificationError, "permission mismatch for radichat"):
            self.run_archive(archive_bytes(binary_mode=0o644))

    def test_rejects_wrong_payload_ownership(self):
        with self.assertRaisesRegex(VerificationError, "ownership mismatch"):
            self.run_archive(archive_bytes(uid=0))

    def test_rejects_unpinned_ca_bundle(self):
        with self.assertRaisesRegex(VerificationError, "CA bundle"):
            with tempfile.TemporaryDirectory() as tmp:
                path = Path(tmp) / "image.oci.tar"
                path.write_bytes(archive_bytes(ca_bundle=b"other CA data\n"))
                verify_archive(path, BINARIES, CA, hashlib.sha256(CA).hexdigest(), REVISION)

    def test_rejects_duplicate_platform(self):
        with self.assertRaisesRegex(VerificationError, "exactly linux/amd64 and linux/arm64"):
            self.run_archive(archive_bytes(duplicate_arch=True))


if __name__ == "__main__":
    unittest.main()
