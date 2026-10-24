# Pinned CA bundle

`ca-certificates.crt` is the Mozilla root CA store converted to PEM and served by the curl project. The checked-in snapshot was retrieved on 2026-10-24 from <https://curl.se/ca/cacert.pem>; curl.se reported that this snapshot was updated by Mozilla on 2026-10-20 and lists 118 certificates.

```text
SHA-256 796f1389bfea13c0ed78f663b765baaff54a508e5483786bd6ece73c5e3e3cbe
Size    200116 bytes
License MPL-2.0 (same license as Mozilla's source certificate store)
```

The checksum is published alongside the download at <https://curl.se/ca/cacert.pem.sha256>. The exact PEM snapshot is vendored here so later downloads or local trust-store customizations cannot silently change the image. Check the file hash before building. Refreshing the CA snapshot is a deliberate reviewed change that must update this record and the OCI build evidence.
