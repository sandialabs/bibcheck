FROM golang:1.24 AS build

WORKDIR /src

ENV GOMODCACHE=/tmp/gomodcache
ENV GOCACHE=/tmp/gocache

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN mkdir -p /out
RUN GOOS=js GOARCH=wasm go build -o /out/app.wasm ./web/app
RUN cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" /out/wasm_exec.js
RUN cp web/static/index.html web/static/style.css /out/
RUN chmod -R g=u /out

FROM registry.access.redhat.com/ubi9/nginx-124:latest

COPY deploy/nginx/nginx.conf "${NGINX_CONF_PATH}"
COPY --from=build --chown=1001:0 /out/ /opt/app-root/src/

EXPOSE 8080

USER 1001
CMD ["nginx", "-g", "daemon off;"]
