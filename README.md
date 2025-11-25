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

1. Install go >= 1.23.0
2. Compile and run:
```bash
go run main.go
```

## Examples

**Analyze a whole document**

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
go run main.go serve sk-...
```
Then navigate to http://localhost:8080 in your browser

## Features

* Bibliographic entry extraction via Google's Gemini family of LLMs
* Metadata matching against
    * arxiv.org
    * osti.gov
    * crossref.org
    * api.elsevier.com
* DOI existence check against doi.org
* Web search via Perplexity

## Known Issues

* Common terms in the paper will confound Perplexity's search, where it won't surface the specific bib entry.
* Gemini will occasionaly be unable to extract a citation text.
  * Gemini Pro is much better, but 4x as expensive and ~4x slower
* Perplexity really wants to summarize the results.
* Some papers are easily found on Google Scholar, but Perplexity does not surface them.

## Roadmap:

* docker/podman build instructions
* Allow user to provide email address (for crossref.org API)

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

* Improve metadata printouts
* Special handling for citations of books
  * crossref is not good at these
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
* dev.elsevier.com

## Licensing

To list third-party licenses:

```bash
go install github.com/google/go-licenses/v2@latest

go-licenses report ./...
```
