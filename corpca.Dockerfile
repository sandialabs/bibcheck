FROM golang:1.24 AS build

WORKDIR /src

ENV GOMODCACHE=/tmp/gomodcache
ENV GOCACHE=/tmp/gocache

RUN apt-get update && \
    apt-get install -y --no-install-recommends brotli && \
    rm -rf /var/lib/apt/lists/*

COPY corpca.crt /usr/local/share/ca-certificates/corpca.crt
RUN update-ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN mkdir -p /out && \
    CGO_ENABLED=0 go build -o /out/bibcheck . && \
    GOOS=js GOARCH=wasm go build -o /out/app.wasm ./web/app && \
    cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" /out/wasm_exec.js && \
    cp web/static/index.html web/static/style.css web/static/footer.css /out/ && \
    for file in /out/*.wasm /out/*.html /out/*.css /out/*.js; do gzip -k -f "$file" && brotli -k -f "$file"; done && \
    chmod -R g=u /out

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

COPY corpca.crt /etc/pki/ca-trust/source/anchors/corpca.crt
RUN update-ca-trust

COPY --from=build --chown=1001:0 /out/bibcheck /usr/local/bin/bibcheck
COPY --from=build --chown=1001:0 /out/app.wasm* /opt/bibcheck/web/
COPY --from=build --chown=1001:0 /out/wasm_exec.js* /opt/bibcheck/web/
COPY --from=build --chown=1001:0 /out/index.html* /opt/bibcheck/web/
COPY --from=build --chown=1001:0 /out/style.css* /opt/bibcheck/web/
COPY --from=build --chown=1001:0 /out/footer.css* /opt/bibcheck/web/

EXPOSE 8080

USER 1001
CMD ["bibcheck", "serve", "--addr", ":8080", "--web-dir", "/opt/bibcheck/web"]
