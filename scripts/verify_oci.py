#!/usr/bin/env python3
"""Verify the structure and payload of a local OCI image archive."""

from __future__ import annotations

import argparse
import gzip
import hashlib
import io
import json
import re
import tarfile
from pathlib import Path


class VerificationError(Exception):
    """Raised when an OCI archive differs from the expected RadiChat image."""


def _fail(message: str) -> None:
    raise VerificationError(message)


def _read_json(data: bytes, label: str) -> dict:
    try:
        value = json.loads(data)
    except (UnicodeDecodeError, json.JSONDecodeError) as exc:
        _fail(f"invalid JSON in {label}: {exc}")
    if not isinstance(value, dict):
        _fail(f"{label} must be a JSON object")
    return value


def _sha(data: bytes) -> str:
    return "sha256:" + hashlib.sha256(data).hexdigest()


def _descriptor_blob(archive: tarfile.TarFile, descriptor: dict, label: str) -> bytes:
    digest = descriptor.get("digest", "")
    match = re.fullmatch(r"sha256:([0-9a-f]{64})", digest)
    if not match:
        _fail(f"{label} has unsupported or invalid digest {digest!r}")
    path = "blobs/sha256/" + match.group(1)
    try:
        member = archive.getmember(path)
    except KeyError:
        _fail(f"missing OCI blob {path} for {label}")
    if not member.isfile():
        _fail(f"OCI blob {path} for {label} is not a regular file")
    fileobj = archive.extractfile(member)
    if fileobj is None:
        _fail(f"unable to read OCI blob {path} for {label}")
    data = fileobj.read()
    if len(data) != descriptor.get("size"):
        _fail(f"size mismatch for {label}: descriptor={descriptor.get('size')} actual={len(data)}")
    if _sha(data) != digest:
        _fail(f"digest mismatch for {label}: descriptor={digest} actual={_sha(data)}")
    return data


def verify_archive(
    archive_path: Path,
    binaries: dict[str, bytes],
    ca_bundle: bytes,
    ca_sha256: str,
    revision: str,
) -> dict:
    """Check exact platforms, image config, filesystem payload, and digests."""
    if hashlib.sha256(ca_bundle).hexdigest() != ca_sha256:
        _fail("pinned CA bundle file does not match the expected SHA-256")
    if set(binaries) != {"amd64", "arm64"}:
        _fail("verifier requires exactly amd64 and arm64 expected binaries")

    try:
        archive = tarfile.open(archive_path, mode="r:*")
    except (OSError, tarfile.TarError) as exc:
        _fail(f"cannot open OCI archive {archive_path}: {exc}")
    with archive:
        members = archive.getmembers()
        names = [item.name.removeprefix("./") for item in members]
        if len(names) != len(set(names)):
            _fail("OCI archive contains duplicate paths")
        allowed = {"oci-layout", "index.json"}
        for name in names:
            if name in allowed or name in {"blobs", "blobs/sha256"}:
                continue
            if name.startswith("blobs/sha256/") and re.fullmatch(r"blobs/sha256/[0-9a-f]{64}", name):
                continue
            if name.endswith("/") and name[:-1] in {"blobs", "blobs/sha256"}:
                continue
            _fail(f"unexpected OCI archive path {name!r}")

        for path in ("oci-layout", "index.json"):
            try:
                member = archive.getmember(path)
            except KeyError:
                _fail(f"OCI archive is missing {path}")
            fileobj = archive.extractfile(member)
            if fileobj is None:
                _fail(f"cannot read {path}")
            data = fileobj.read()
            if path == "oci-layout":
                layout = _read_json(data, path)
                if layout.get("imageLayoutVersion") != "1.0.0":
                    _fail("OCI layout version must be 1.0.0")
            else:
                index_bytes = data
                index = _read_json(data, path)

        if index.get("schemaVersion") != 2 or index.get("mediaType") != "application/vnd.oci.image.index.v1+json":
            _fail("index.json must be an OCI image index (schemaVersion 2)")
        layout_descriptors = index.get("manifests")
        if not isinstance(layout_descriptors, list) or len(layout_descriptors) != 1:
            _fail("OCI archive index must point to exactly one multi-platform image index")
        layout_descriptor = layout_descriptors[0]
        if layout_descriptor.get("mediaType") != "application/vnd.oci.image.index.v1+json":
            _fail("OCI archive index must point to a multi-platform OCI image index")
        image_index_bytes = _descriptor_blob(archive, layout_descriptor, "multi-platform image index")
        referenced_blobs = {"blobs/sha256/" + layout_descriptor["digest"].split(":", 1)[1]}
        image_index = _read_json(image_index_bytes, "multi-platform image index")
        if image_index.get("schemaVersion") != 2 or image_index.get("mediaType") != "application/vnd.oci.image.index.v1+json":
            _fail("referenced image index must be an OCI image index (schemaVersion 2)")
        descriptors = image_index.get("manifests")
        if not isinstance(descriptors, list):
            _fail("multi-platform image index manifests must be an array")
        by_arch = {}
        for descriptor in descriptors:
            platform = descriptor.get("platform", {})
            key = (platform.get("os"), platform.get("architecture"))
            if key not in {("linux", "amd64"), ("linux", "arm64")} or key[1] in by_arch:
                _fail("index must contain exactly linux/amd64 and linux/arm64")
            by_arch[key[1]] = descriptor
        if set(by_arch) != {"amd64", "arm64"}:
            _fail("index must contain exactly linux/amd64 and linux/arm64")

        result = {
            "index_digest": layout_descriptor["digest"],
            "layout_index_digest": _sha(index_bytes),
            "archive_digest": "sha256:" + hashlib.sha256(archive_path.read_bytes()).hexdigest(),
            "platforms": ["linux/amd64", "linux/arm64"],
            "manifests": {},
        }
        expected_paths = {
            "config/",
            "etc/",
            "etc/ssl/",
            "etc/ssl/certs/",
            "etc/ssl/certs/ca-certificates.crt",
            "radichat",
        }
        expected_env = [
            "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
            "SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt",
        ]
        for arch in ("amd64", "arm64"):
            descriptor = by_arch[arch]
            if descriptor.get("mediaType") != "application/vnd.oci.image.manifest.v1+json":
                _fail(f"{arch} descriptor is not an OCI image manifest")
            manifest_bytes = _descriptor_blob(archive, descriptor, f"{arch} manifest")
            referenced_blobs.add("blobs/sha256/" + descriptor["digest"].split(":", 1)[1])
            manifest = _read_json(manifest_bytes, f"{arch} manifest")
            if manifest.get("schemaVersion") != 2 or manifest.get("mediaType") != "application/vnd.oci.image.manifest.v1+json":
                _fail(f"{arch} manifest has an unexpected schema or media type")
            config_descriptor = manifest.get("config", {})
            if config_descriptor.get("mediaType") != "application/vnd.oci.image.config.v1+json":
                _fail(f"{arch} config is not an OCI image config")
            config_bytes = _descriptor_blob(archive, config_descriptor, f"{arch} config")
            referenced_blobs.add("blobs/sha256/" + config_descriptor["digest"].split(":", 1)[1])
            image = _read_json(config_bytes, f"{arch} image config")
            config = image.get("config", {})
            expected_config = {
                "User": "65532:65532",
                "WorkingDir": "/config",
                "Env": expected_env,
                "Entrypoint": ["/radichat"],
                "Cmd": ["-config", "/config/radichat.json"],
                "Labels": {"version": "v0.1.1", "revision": revision},
            }
            for key, value in expected_config.items():
                if config.get(key) != value:
                    _fail(f"{arch} config {key} mismatch: expected {value!r}, got {config.get(key)!r}")
            if image.get("os") != "linux" or image.get("architecture") != arch:
                _fail(f"{arch} config OS/architecture mismatch")
            layers = manifest.get("layers", [])
            if not layers or any(layer.get("mediaType") != "application/vnd.oci.image.layer.v1.tar+gzip" for layer in layers):
                _fail(f"{arch} must contain one or more OCI gzip tar layers")
            diff_ids = image.get("rootfs", {}).get("diff_ids", [])
            if len(diff_ids) != len(layers):
                _fail(f"{arch} rootfs diff ID count does not match layer count")
            final_paths = set()
            content = {}
            for layer_number, layer_descriptor in enumerate(layers, 1):
                referenced_blobs.add("blobs/sha256/" + layer_descriptor["digest"].split(":", 1)[1])
                compressed = _descriptor_blob(archive, layer_descriptor, f"{arch} layer {layer_number}")
                try:
                    raw_layer = gzip.decompress(compressed)
                except OSError as exc:
                    _fail(f"{arch} layer {layer_number} is not valid gzip: {exc}")
                if diff_ids[layer_number - 1] != _sha(raw_layer):
                    _fail(f"{arch} rootfs diff ID mismatch in layer {layer_number}")
                try:
                    layer = tarfile.open(fileobj=io.BytesIO(raw_layer), mode="r:")
                except tarfile.TarError as exc:
                    _fail(f"{arch} layer {layer_number} is not a valid tar: {exc}")
                with layer:
                    layer_members = layer.getmembers()
                    layer_paths = [
                        member.name.removeprefix("./").rstrip("/") + "/" if member.isdir()
                        else member.name.removeprefix("./")
                        for member in layer_members
                    ]
                    if len(layer_paths) != len(set(layer_paths)):
                        _fail(f"{arch} layer {layer_number} contains duplicate paths")
                    for member in layer_members:
                        member_path = member.name.removeprefix("./").rstrip("/")
                        normalized_path = member_path + "/" if member.isdir() else member_path
                        if normalized_path not in expected_paths:
                            _fail(f"{arch} layer paths include unexpected entry {normalized_path!r}")
                        if not member.isdir() and not member.isfile():
                            _fail(f"{arch} layer contains unsupported file type: {member.name}")
                        if member.uid != 65532 or member.gid != 65532:
                            _fail(f"{arch} layer ownership mismatch for {member.name}: {member.uid}:{member.gid}")
                        expected_mode = 0o755 if member.isdir() or member_path == "radichat" else 0o644
                        if member.mode != expected_mode:
                            _fail(f"{arch} layer permission mismatch for {member.name}: {member.mode:o}")
                        final_paths.add(normalized_path)
                        if member.isfile():
                            fileobj = layer.extractfile(member)
                            if fileobj is None:
                                _fail(f"cannot read {arch} layer file {member.name}")
                            content[member_path] = fileobj.read()
            if final_paths != expected_paths:
                _fail(f"{arch} layer paths mismatch: expected {sorted(expected_paths)}, got {sorted(final_paths)}")
            if content.get("radichat") != binaries[arch]:
                _fail(f"{arch} image binary does not match the supplied release binary")
            if content.get("etc/ssl/certs/ca-certificates.crt") != ca_bundle:
                _fail(f"{arch} image CA bundle does not match the pinned CA snapshot")
            result["manifests"][arch] = {
                "manifest_digest": descriptor["digest"],
                "layer_digests": [layer["digest"] for layer in layers],
                "binary_sha256": hashlib.sha256(binaries[arch]).hexdigest(),
                "ca_sha256": hashlib.sha256(ca_bundle).hexdigest(),
            }
        archive_blob_paths = {name for name in names if name.startswith("blobs/sha256/")}
        if archive_blob_paths != referenced_blobs:
            _fail("OCI archive contains unreferenced blobs or is missing referenced blobs")
        return result


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--archive", type=Path, required=True)
    parser.add_argument("--amd64", type=Path, required=True)
    parser.add_argument("--arm64", type=Path, required=True)
    parser.add_argument("--ca-bundle", type=Path, required=True)
    parser.add_argument("--ca-sha256", required=True)
    parser.add_argument("--revision", required=True)
    args = parser.parse_args()
    try:
        result = verify_archive(
            args.archive,
            {"amd64": args.amd64.read_bytes(), "arm64": args.arm64.read_bytes()},
            args.ca_bundle.read_bytes(),
            args.ca_sha256,
            args.revision,
        )
    except (OSError, VerificationError) as exc:
        parser.error(str(exc))
    print(json.dumps(result, indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
