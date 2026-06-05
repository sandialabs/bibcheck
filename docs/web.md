# Web UI Deployment

The web UI is a static Go WebAssembly app: the browser reads the selected PDF locally and calls Shirty or OpenRouter directly with the API key the user pastes into the page.

The webapp is static-only.
It does not persist PDF uploads or state, store API keys, or expose analysis API endpoints.
All state is held in browser memory for the current page session.

## Local Build

Build the WASM bundle from the repo root:

```bash
GOOS=js GOARCH=wasm go build -o web/static/app.wasm ./web/app
```

Copy the `wasm_exec.js` file that matches the Go toolchain used for the build:

```bash
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/static/wasm_exec.js
```

Serve the static directory locally:

```bash
go run . serve
```

Then open <http://localhost:8080>.

## Container Image

Uses a multi-stage container build:

1. A Go builder stage compiles `./web/app` to `app.wasm`.
2. The builder stage copies the matching Go `wasm_exec.js`.
3. A runtime stage serves only static files with Red Hat's OpenShift-oriented nginx image.

For OpenShift, use the Red Hat UBI nginx image `registry.access.redhat.com/ubi9/nginx-124:latest`.
This image is intended for OpenShift nginx applications and runs on port `8080`.


## OpenShift Notes

OpenShift commonly runs containers with an arbitrary non-root UID.
The runtime image must not require a fixed UID, root-owned writable nginx paths, or binding to privileged ports.

Use these defaults:

- Listen on `8080`, not `80`.
- Serve static files from `/opt/app-root/src`.
- Do not write logs or cache files into application directories.
- Keep static assets world-readable or group-readable.
- Do not bake API keys into the image. Users paste keys into the browser at run
  time.
- Do not add a backend proxy unless the browser-direct Shirty/OpenRouter access
  model changes.

The Red Hat nginx image also supports S2I, but this repository uses a Dockerfile
because the WASM bundle must be compiled before nginx serves it.

## Local Container Development


```bash
podman build -t bibcheck-wasm .
```

Run with the image's default user:

```bash
podman run --rm -p 8080:8080 bibcheck-wasm
```

Run with an arbitrary OpenShift-style UID:

```bash
podman run --rm --user 12345:0 -p 8080:8080 bibcheck-wasm
```

Verify:

```bash
curl -I http://localhost:8080/
curl -I http://localhost:8080/app.wasm
curl -I http://localhost:8080/wasm_exec.js
curl -I http://localhost:8080/style.css
```

Expected results:

- `/` returns `200 OK` and `text/html`.
- `/app.wasm` returns `200 OK` and `application/wasm`.
- `/wasm_exec.js` returns `200 OK` and JavaScript content.
- `/style.css` returns `200 OK` and CSS content.
