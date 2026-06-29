#!/usr/bin/env zsh

set -euo pipefail
unsetopt BG_NICE

readonly repo_root="${0:A:h:h}"
readonly build_dir="$(mktemp -d "${TMPDIR:-/tmp}/bibcheck.XXXXXXXX")"
server_pid=""

log() {
	print -u2 -r -- "[bibcheck] $*"
}

cleanup() {
	local exit_code=$?

	trap - EXIT HUP INT TERM
	if [[ -n "${server_pid}" ]] && kill -0 "${server_pid}" 2>/dev/null; then
		log "Stopping server (PID ${server_pid})..."
		kill -TERM "${server_pid}" 2>/dev/null || true
		wait "${server_pid}" 2>/dev/null || true
	fi
	log "Removing temporary build directory: ${build_dir}"
	rm -rf -- "${build_dir}"
	log "Cleanup complete."
	exit "${exit_code}"
}

trap cleanup EXIT
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM

log "Using temporary build directory: ${build_dir}"
mkdir -p "${build_dir}/web"
export GOCACHE="${build_dir}/go-cache"

cd "${repo_root}"
log "Building the server binary..."
CGO_ENABLED=0 go build -o "${build_dir}/bibcheck" .

log "Building the WebAssembly application..."
GOOS=js GOARCH=wasm go build -o "${build_dir}/web/app.wasm" ./web/app

log "Copying web assets..."
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" "${build_dir}/web/wasm_exec.js"
cp web/static/*.html web/static/*.css "${build_dir}/web/"

log "Compressing web assets..."
if command -v brotli >/dev/null 2>&1; then
	use_brotli=true
else
	use_brotli=false
	log "Brotli is not installed; creating gzip assets only."
fi

for file in "${build_dir}"/web/*.(wasm|html|css|js); do
	log "Compressing ${file:t}..."
	gzip -k -f "${file}"
	if ${use_brotli}; then
		brotli -5 -k -f "${file}"
	fi
done

log "Starting server at ${BIBCHECK_ADDR:-:8080} (press Ctrl-C to stop)..."
"${build_dir}/bibcheck" serve \
	--addr "${BIBCHECK_ADDR:-:8080}" \
	--web-dir "${build_dir}/web" \
	"$@" &
server_pid=$!
wait "${server_pid}"
