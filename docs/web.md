# Web UI Deployment

The web UI is a Go WebAssembly app served by the `bibcheck serve` command.
The browser reads the selected PDF locally and calls Shirty or OpenRouter
directly with the API key the user pastes into the page.

The server also exposes `GET /api/fetch?url=...` for online bibliography
resources. This endpoint lets the wasm app fetch HTML or PDF resources through
the same origin when the target website does not allow browser CORS requests.
It does not persist PDF uploads or state, store API keys, or expose analysis API
endpoints. All analysis state is held in browser memory for the current page
session.

## Network Exposure

Do not expose `bibcheck serve` directly to the public internet.

The `/api/fetch` endpoint is intentionally narrow, but it is still a server-side
URL fetcher. If the service is reachable by untrusted users, they could use it as
a free proxy. Deploy it only on trusted networks, behind authentication, or
behind an ingress policy that restricts access to intended users.

## Local Build

Build the WASM bundle from the repo root:

```bash
GOOS=js GOARCH=wasm go build -o web/static/app.wasm ./web/app
```

Copy the `wasm_exec.js` file that matches the Go toolchain used for the build:

```bash
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/static/wasm_exec.js
```

Serve the web UI locally:

```bash
go run . serve
```

Then open <http://localhost:8080>.

By default, `/api/fetch` reads at most 25 MiB from an upstream response. Adjust
that for larger PDFs if needed:

```bash
go run . serve --fetch-max-bytes 52428800
```

## Container Image

Uses a multi-stage container build:

1. A Go builder stage compiles the native `bibcheck` server binary.
2. The builder stage compiles `./web/app` to `app.wasm`.
3. The builder stage copies the matching Go `wasm_exec.js`.
4. A UBI minimal runtime stage runs `bibcheck serve` on port `8080`.

The runtime container serves static web assets from `/opt/bibcheck/web` and
handles `/api/fetch` in the same process.

## OpenShift Notes

OpenShift commonly runs containers with an arbitrary non-root UID.
The runtime image must not require a fixed UID, root-owned writable application
paths, or binding to privileged ports.

Use these defaults:

- Listen on `8080`, not `80`.
- Serve static files from `/opt/bibcheck/web`.
- Do not write logs or cache files into application directories.
- Keep static assets world-readable or group-readable.
- Do not bake API keys into the image. Users paste keys into the browser at run
  time.
- Restrict access to the service. Do not publish `/api/fetch` as an unauthenticated
  public endpoint.

## Local Container Development

```bash
podman build -f bare.Dockerfile -t bibcheck-wasm .
```

If your environment requires some specific SSL certificate, you will need to provide that certificate as `corpca.crt`, and then build `corpca.Dockerfile` instead:
```bash
podman build -f corpca.Dockerfile -t bibcheck-wasm .
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
curl -sS -D - "http://localhost:8080/api/fetch?url=https%3A%2F%2Fwww.hpcg-benchmark.org%2F" -o /tmp/bibcheck-fetch.html
```

Expected results:

- `/` returns `200 OK` and `text/html`.
- `/app.wasm` returns `200 OK` and `application/wasm`.
- `/wasm_exec.js` returns `200 OK` and JavaScript content.
- `/style.css` returns `200 OK` and CSS content.
- `/api/fetch?...` returns the upstream response when the server can reach the
  requested URL and the response is within the configured byte limit.
