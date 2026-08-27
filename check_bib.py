#!/usr/bin/env python3
"""Sample: check a bibliography string with `bibcheck text`.

Usage:
    python3 check_bib.py            # checks the built-in sample string
    python3 check_bib.py refs.txt   # checks a file instead
"""

import json
import subprocess
import sys

BIBCHECK = "./bibcheck"  # path to the binary built with `go build -o bibcheck .`

# The string your Python program would hold. Five real entries (four arXiv,
# one DOI), then one with a wrong arXiv ID, one fabricated with no identifier.
SAMPLE_BIB = (
    "[1] A. Vaswani et al., 2017, Attention Is All You Need, "
    "https://arxiv.org/abs/1706.03762 "
    "[2] K. He, X. Zhang, S. Ren, J. Sun, 2015, Deep Residual Learning for "
    "Image Recognition, arXiv:1512.03385 "
    "[3] J. Devlin, M. Chang, K. Lee, K. Toutanova, 2018, BERT: Pre-training "
    "of Deep Bidirectional Transformers for Language Understanding, "
    "arXiv:1810.04805 "
    "[4] D. P. Kingma and J. Ba, 2014, Adam: A Method for Stochastic "
    "Optimization, arXiv:1412.6980 "
    "[5] J. Jumper et al., 2021, Highly accurate protein structure prediction "
    "with AlphaFold, Nature 596, 583-589, "
    "https://doi.org/10.1038/s41586-021-03819-2 "
    "[6] J. Smith et al., 2020, A Study of Things, arXiv:2003.99999 "
    "[7] Q. Fictional and R. Imaginary, 2022, Deep Learning for Unicorn "
    "Detection, Journal of Made-Up Results 12(3), 45-67."
    "[8] A. Vaswani et al., 2017, Attention Is All You Need, "
    "https://arxiv.org/abs/1707.03762 " # Modify and see if it'll detect wrong arxiv.
    "[9] K. He, X. Zhang, S. Ren, J. Sun, 2015, Deep Residual Learning for " # Modify, to see will detect wrong names
    "https://arxiv.org/abs/1706.03762 "

)

POSITIVE = {"found", "matched"}
NEGATIVE = {"not-found", "no-match"}


def _arxiv_title(detail: str) -> str:
    """Extract the title from an arXiv detail string.

    Format: 'Author1, Author2. Title Here. published DATE. [updated DATE.]'
    """
    pub = detail.find(". published ")
    if pub == -1:
        return ""
    before = detail[:pub]
    dot = before.rfind(". ")
    if dot == -1:
        return ""
    return before[dot + 2:]


def _title_matches(detail: str, cited: str) -> bool:
    """True if the arXiv title words mostly appear in the cited text."""
    title = _arxiv_title(detail)
    if not title:
        return True
    words = [w.lower() for w in title.split() if len(w) > 2]
    if not words:
        return True
    cited_lower = cited.lower()
    hits = sum(1 for w in words if w in cited_lower)
    return hits / len(words) >= 0.5


def check(bib_string: str) -> list[dict]:
    result = subprocess.run(
        # --workers 1 keeps arXiv lookups sequential; parallel requests
        # trigger arXiv's rate limiting (HTTP 429)
        [BIBCHECK, "text", "--format", "json", "--workers", "1", "-"],
        input=bib_string,
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        raise RuntimeError(f"bibcheck failed: {result.stderr.strip()}")
    return json.loads(result.stdout)["entries"]


def verdict(entry: dict) -> tuple[str, str]:
    """Reduce an entry's per-source results to a one-word verdict."""
    statuses = {s["name"]: s["status"] for s in entry["sources"]}
    details = {s["name"]: s["detail"] for s in entry["sources"]}

    for name in ("arXiv", "OSTI", "Crossref", "Elsevier", "Online"):
        if statuses[name] in POSITIVE:
            if not _title_matches(details[name], entry["original_text"]):
                return "MISMATCH", f"{name} record does not match: {details[name][:100]}"
            return "VERIFIED", f"{name}: {details[name][:100]}"
    if statuses["DOI"] == "found":
        return "DOI-OK", "DOI resolves at doi.org (weak signal: content not compared)"

    negatives = [n for n, s in statuses.items() if s in NEGATIVE]
    errors = [f"{n}: {details[n][:80]}" for n, s in statuses.items() if s == "error"]
    if errors:
        # a lookup failed (e.g. arXiv HTTP 429 rate limiting), so a negative
        # from another source is not conclusive -- retry after ~90s
        return "RETRY", "; ".join(errors)
    if negatives:
        return "SUSPECT", f"no source could confirm it ({', '.join(negatives)} negative)"
    return "UNKNOWN", "no checkable identifier and no match"


def main() -> None:
    if len(sys.argv) > 1:
        with open(sys.argv[1]) as f:
            bib = f.read()
    else:
        bib = SAMPLE_BIB

    for entry in check(bib):
        print(entry)
        v, why = verdict(entry)
        print(f"[{entry['number']}] {v:8s} {entry['original_text'][:70]}")
        print(f"             {why}\n")


if __name__ == "__main__":
    main()
