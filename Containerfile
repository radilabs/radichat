FROM scratch AS payload

ARG TARGETARCH

COPY --chown=65532:65532 radichat-linux-${TARGETARCH} /radichat
COPY --chown=65532:65532 ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
USER 65532:65532
WORKDIR /config

FROM scratch

ARG SOURCE_COMMIT

LABEL version="v0.1.1" \
      revision="${SOURCE_COMMIT}"

COPY --from=payload / /

USER 65532:65532
WORKDIR /config
ENV SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt
ENTRYPOINT ["/radichat"]
CMD ["-config", "/config/radichat.json"]
