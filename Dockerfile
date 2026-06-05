FROM golang:1.24 AS build

WORKDIR /src

ENV GOMODCACHE=/tmp/gomodcache
ENV GOCACHE=/tmp/gocache

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN mkdir -p /out
RUN CGO_ENABLED=0 go build -o /out/bibcheck .
RUN GOOS=js GOARCH=wasm go build -o /out/app.wasm ./web/app
RUN cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" /out/wasm_exec.js
RUN cp web/static/index.html web/static/style.css /out/
RUN chmod -R g=u /out

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

COPY --from=build --chown=1001:0 /out/bibcheck /usr/local/bin/bibcheck
COPY --from=build --chown=1001:0 /out/app.wasm /opt/bibcheck/web/app.wasm
COPY --from=build --chown=1001:0 /out/wasm_exec.js /opt/bibcheck/web/wasm_exec.js
COPY --from=build --chown=1001:0 /out/index.html /opt/bibcheck/web/index.html
COPY --from=build --chown=1001:0 /out/style.css /opt/bibcheck/web/style.css

EXPOSE 8080

USER 1001
CMD ["bibcheck", "serve", "--addr", ":8080", "--web-dir", "/opt/bibcheck/web"]
