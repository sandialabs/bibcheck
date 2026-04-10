# bibcheck

Catch errors in paper bibliographies.

> [!WARNING]
> This tool is not a substitute for your professional judgement!

> [!WARNING]
> This tool may make mistakes!

## Quick Start (macOS)

Download the appropriate binary from the [latest release](https://github.com/sandialabs/bibcheck/releases/latest)

1. Remove the quarantine bit (the binaries aren't signed or whatever)
2. Make executable
3. Run

```bash
xattr -d com.apple.quarantine bibcheck-darwin-arm64
chmod +x bibcheck-darwin-arm64
./bibcheck-darwin-arm64
```

## Quick Start (Linux)

Download the appropriate binary from the [latest release](https://github.com/sandialabs/bibcheck/releases/latest)

1. Make executable
2. Run

```bash
chmod +x bibcheck-linux-amd64
./bibcheck-linux-amd64
```

## Quick Start (Build from Source)

1. Install Go >= 1.24.0
2. Compile and run:
```bash
go run main.go
```

## Examples

**Analyze a whole document**

```bash
export SHIRTY_API_KEY=sk-...
go run main.go test/20231113_siefert_pmbs.pdf
```

```bash
go run main.go --shirty-api-key sk-... test/20231113_siefert_pmbs.pdf 
```
```
=== Entry 1 ===
[1] 2017. NVIDIA Tesla V100 GPU Architecture. https://images.nvidia.com/content/volta-architecture/pdf/volta-architecture-whitepaper.pdf
Detected URL: https://images.nvidia.com/content/volta-architecture/pdf/volta-architecture-whitepaper.pdf
Kind: website
direct URL access...
recieved application/pdf from https://images.nvidia.com/content/volta-architecture/pdf/volta-architecture-whitepaper.pdf
URL: ✓ LOOKS OKAY
     Both entries have the same title, with minor transcription-style differences (capitalization and spacing). The contributing organization in ENTRY 1 matches the implied author/organization in ENTRY 2.
=== Entry 2 ===
[2] 2018. HPL - A Portable Implementation of the High-Performance Linpack Benchmark for Distributed-Memory Computers. https://www.netlib.org/benchmark/hpl
Detected URL: https://www.netlib.org/benchmark/hpl
Kind: software_package
direct URL access...
recieved text/html; charset=UTF-8 from https://www.netlib.org/benchmark/hpl
URL: ✓ LOOKS OKAY
     The title field matches exactly between the two entries, but there is no author information in the second entry to compare with the first entry. However, based on the available data, it appears that both entries reference the same thing, which is the HPL benchmark.
=== Entry 3 ===
[3] 2020. NVIDIA A100 Tensor Core GPU Architecture. https://images.nvidia.com/aem-dam/en-zz/Solutions/data-center/nvidia-ampere-architecture-whitepaper.pdf
Detected URL: https://images.nvidia.com/aem-dam/en-zz/Solutions/data-center/nvidia-ampere-architecture-whitepaper.pdf
Kind: website
direct URL access...
recieved application/pdf from https://images.nvidia.com/aem-dam/en-zz/Solutions/data-center/nvidia-ampere-architecture-whitepaper.pdf
URL: ✓ LOOKS OKAY
     Both entries have the same title, which suggests they reference the same thing.
...
```

**Analyze a single entry**

```bash
export SHIRTY_API_KEY=sk-...
go run main.go test/20231113_siefert_pmbs.pdf --entry 14
```

```bash
go run main.go --shirty-api-key sk-... test/20231113_siefert_pmbs.pdf --entry 14
```
```
=== Entry 14 ===
[14] 2023. OSU Micro-benchmarks. http://mvapich.cse.ohio-state.edu/benchmarks/
Detected URL: http://mvapich.cse.ohio-state.edu/benchmarks/
Kind: website
direct URL access...
recieved text/html; charset=utf-8 from http://mvapich.cse.ohio-state.edu/benchmarks/
Website:
  URL:     http://mvapich.cse.ohio-state.edu/benchmarks/
  Title:   OSU Micro-benchmarks
  Authors: 
URL: NO MATCH
     The titles do not match, but the URL in ENTRY 2 suggests a connection to MVAPICH, which is present in the title of ENTRY 1.
```

**Interactive GUI (shirty only)**
```
export SHIRTY_API_KEY=sk-...
go run main.go serve
```
or
```
go run main.go serve sk-...
```
Then navigate to http://localhost:8080 in your browser

`OPENROUTER_API_KEY` and `SHIRTY_API_KEY` are used automatically when set. Command-line flags still override environment values.
`OPENROUTER_BASE_URL` and `SHIRTY_BASE_URL` are also supported.

## Features

* Extracts bibliography entries from PDF documents and analyzes them one-by-one
* Supports both CLI analysis and a lightweight web UI for uploaded PDFs
* Uses configured LLM backends for bibliography counting, entry extraction, metadata parsing, and optional result summarization
    * `SHIRTY_API_KEY` enables the Shirty-based pipeline
    * `OPENROUTER_API_KEY` enables the OpenRouter-based CLI pipeline for bibliography counting, entry extraction, and metadata parsing
* Verifies entries with direct lookups against
    * doi.org
    * arXiv
    * OSTI
    * Crossref
    * Elsevier Scopus search (when `ELSEVIER_API_KEY` is configured)
* Fetches and analyzes linked online resources when an entry points to a URL
    * HTML pages
    * PDF documents
* Includes bibliography-oriented CLI helpers
    * `bib` extracts the bibliography
    * `entry` extracts a single bibliography entry
    * `list-entries` lists numeric bibliography entry IDs

## "Search" strategy

For each extracted bibliography entry, bibcheck currently works in this order:

* DOI check
    * If a DOI is present, resolve it through doi.org to confirm that it exists
    * This does not stop the search, because DOI resolution alone does not provide enough metadata for comparison
* OSTI lookup
    * If an OSTI identifier is present, fetch the OSTI record directly
    * A successful OSTI match is treated as sufficient
* arXiv lookup
    * If an arXiv identifier is present, fetch the arXiv metadata directly
    * A successful arXiv match is treated as sufficient
* Elsevier search
    * If `ELSEVIER_API_KEY` is configured, parse authors, title, and publication venue, then query Elsevier
* Crossref bibliographic search
    * Query Crossref with the full bibliography entry text
    * Only accept a result when the top score is strong enough and not effectively tied with the next match
* Online resource lookup
    * If no database/source match was found, parse the entry as an online resource
    * Fetch the URL directly and extract metadata from HTML or PDF content for comparison

## Roadmap:

* docker/podman build instructions
* Allow user to provide email address (for crossref.org API)
* DBLP search
* OpenAlex search
* Web UI

## Contributing

### Running Tests

Run tests with a shirty key or an openrouter key

```bash
go test -v ./... -args --shirty-api-key="sk-..."
go test -v ./... -args --openrouter-api-key="sk-or-v1-..."
```

### Release Deployments

* Create fine-grained token with "Contents" repository permissions (write)
* Add it as an actions secret: `RELEASE_TOKEN`

## Acknowledgements

* Thank you to arXiv for use of its open access interoperability.
* Thank you to OSTI for providing a free API
* Thank you to Crossref for providing a free API
* Thank you to doi.org for providing a free API

## Roadmap

* OpenAlex
* If a URL is available, try that first, e.g. for 
```
[3] C. Bormann, M. Ersue, and A. Keranen, "Terminology for Constrained-Node Networks," RFC 7228, Internet Engineering Task Force, May 2014. [Online]. Available: https://tools.ietf.org/html/rfc7228
```
* offer a google scholar link when we can't find it, e.g.
```
https://scholar.google.com/scholar?hl=en&as_sdt=0%2C32&q=Quantum+information&btnG=
```
* offer a DOI URL when the DOI is found
* Show selected file in upload GUI
* extract alphanumeric bibliography entry
    * structured response asking for all bibliography IDs
* try other shirty LLMs
    * `openai/gpt-oss-120b`

## Licensing

To list third-party licenses:

save the following as `template.md`
```md
{{ range . }}
## {{ .Name }}

* Name: {{ .Name }}
* Version: {{ .Version }}
* License: [{{ .LicenseName }}]({{ .LicenseURL }})

```
{{ .LicenseText }}
```
{{ end }}
```

```bash
go install github.com/google/go-licenses/v2@latest

go-licenses report ./... --ignore github.com/sandialabs/bibcheck --template template.md > NOTICE
```
