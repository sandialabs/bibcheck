// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
// Tests for HTML metadata evidence extraction.
package documentmetadata

import (
	"strings"
	"testing"
)

func TestPrepareHTMLHPCGBenchmarkPage(t *testing.T) {
	// hpcg-benchmark.org
	raw := []byte(`
<html>
<head>
  <title>HPCG Benchmark</title>
  <link rel="stylesheet" href="/assets/stylesheet.css" type="text/css">
  <script language="javascript" src="/assets/scripts/browserdetect_png.js"></script>
  <style type="text/css">.SectionTitle, h1 { font-size: 18px; } ` + strings.Repeat("noise", 2_000) + `</style>
</head>
<body marginwidth="0" marginheight="0" bgcolor="#FFFFFF">
  <center><div class="Frame"><table width="755"><tr>
    <td class="LeftColumn"><div class="NavSection">
      <div class="NavButton"><a href="/">Home</a></div>
      <div class="NavButton"><a href="/software/index.html">Software</a></div>
      <noscript><div class="NavButton">Submission Form</div></noscript>
    </div></td>
    <td class="PageContentCell"><div class="PageContent">
      <table width="100%"><tr><td class="Intro">
        <div class="SectionTitle"><h1>HPCG Benchmark</h1></div>
        <div class="IntroText">
          <p>The High Performance Conjugate Gradients (HPCG) Benchmark project is an effort to create a new metric for ranking HPC systems.</p>
          <p>HPCG is a complete, stand-alone code that measures the performance of basic operations in a unified code.</p>
        </div>
        <div class="SectionTitle">New HPCG results announced at SC25</div>
        <div class="IntroText"><p>The new HPCG Performance List was announced at the SC25 conference.</p></div>
      </td></tr></table>
    </div></td>
  </tr></table></div></center>
  <script>document.write('info@example.invalid');</script>
  <div class="fine">Jun 09 2022</div>
</body>
</html>`)

	got := PrepareHTML(raw, DefaultConfig())
	for _, want := range []string{
		"<!-- HTML title -->",
		"<title>HPCG Benchmark</title>",
		"<h1> HPCG Benchmark</h1>",
		"The High Performance Conjugate Gradients",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output does not contain %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"font-size", strings.Repeat("noise", 100), "browserdetect_png.js", "document.write"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("output contains head/script noise %q", unwanted)
		}
	}
	if strings.Contains(got, "RAW HTML FALLBACK") {
		t.Fatal("representative HPCG HTML unexpectedly used the raw fallback")
	}
}

func TestPrepareHTMLGitHubRepositoryPage(t *testing.T) {
	// github.com
	raw := []byte(`
<html lang="en">
<head>
  <title>GitHub - argonne-lcf/alcf-mpi-benchmarks · GitHub</title>
  <meta name="description" content="Contribute to argonne-lcf/alcf-mpi-benchmarks development by creating an account on GitHub.">
  <meta name="twitter:title" content="GitHub - argonne-lcf/alcf-mpi-benchmarks">
  <meta property="og:site_name" content="GitHub">
  <meta property="og:title" content="GitHub - argonne-lcf/alcf-mpi-benchmarks">
  <style>` + strings.Repeat(".application-noise{}", 2_000) + `</style>
  <script type="application/javascript">` + strings.Repeat("window.webpackNoise=true;", 2_000) + `</script>
</head>
<body>
  <header><nav><a href="/features">Platform</a><a href="/pricing">Pricing</a></nav></header>
  <div id="repository-container-header" class="hide-full-screen">
    <div class="d-flex flex-wrap flex-items-center wb-break-word f3 text-normal">
      <span class="author flex-self-stretch" itemprop="author">
        <a class="url fn" rel="author" href="/argonne-lcf">argonne-lcf</a>
      </span>
      <span class="color-fg-muted">/</span>
      <strong itemprop="name"><a href="/argonne-lcf/alcf-mpi-benchmarks">alcf-mpi-benchmarks</a></strong>
      <span class="Label">Public</span>
    </div>
  </div>
  <main id="repo-content-pjax-container">
    <div class="Box">
      <h2>Repository files navigation</h2>
      <div class="markdown-heading"><h2>README</h2></div>
      <article class="markdown-body entry-content container-lg">
        <p>ALCF MPI benchmarks suite consists of five independent programs: mmps, pingpong, aggregate, bisection, and collectives.</p>
        <p>These programs measure messaging rate, communication latency, bandwidth, and collective operations.</p>
      </article>
    </div>
  </main>
  <footer>© GitHub, Inc.</footer>
</body>
</html>`)

	got := PrepareHTML(raw, DefaultConfig())
	for _, want := range []string{
		"GitHub - argonne-lcf/alcf-mpi-benchmarks",
		`itemprop="author"`,
		`rel="author"`,
		"argonne-lcf",
		"alcf-mpi-benchmarks",
		"ALCF MPI benchmarks suite consists of five independent programs",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output does not contain %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{strings.Repeat("application-noise", 100), strings.Repeat("webpackNoise", 100)} {
		if strings.Contains(got, unwanted) {
			t.Errorf("output contains GitHub asset noise %q", unwanted[:20])
		}
	}
	if strings.Contains(got, "RAW HTML FALLBACK") {
		t.Fatal("representative GitHub HTML unexpectedly used the raw fallback")
	}
}

func TestPrepareHTMLNetlibHPLPage(t *testing.T) {
	// netlib.org
	raw := []byte(`
<HTML><HEAD><TITLE>
HPL - A Portable Implementation of the High-Performance
Linpack Benchmark for Distributed-Memory Computers
</TITLE></HEAD>
<BODY BGCOLOR="WHITE" TEXT="#000000" LINK="#0000ff">
<HR NOSHADE>
<TABLE WIDTH="100%" BORDER="0"><TR>
  <TD ALIGN="CENTER"><H3>HPL - A Portable Implementation of the High-Performance Linpack Benchmark for Distributed-Memory Computers</H3></TD>
  <TD ALIGN="LEFT"><A HREF="http://icl.cs.utk.edu"><IMG SRC="2-273x48.jpg" ALT="ICL"></A></TD>
</TR></TABLE>
<TABLE WIDTH="100%" BORDER="0"><TR>
  <TD ALIGN="LEFT">Version 2.3</TD>
  <TD ALIGN="CENTER">
    <A HREF="http://www.cs.utk.edu/~petitet">A. Petitet</A>,
    <A HREF="http://www.cs.utk.edu/~rwhaley">R. C. Whaley</A>,
    <A HREF="http://www.netlib.org/utk/people/JackDongarra">J. Dongarra</A>,
    <A HREF="mailto:cleary1@llnl.gov">A. Cleary</A>
  </TD>
  <TD ALIGN="CENTER">December 2, 2018</TD>
</TR></TABLE>
<P>The reference implementation of HPL is now maintained in a <A HREF="https://github.com/icl-utk-edu/hpl/">GitHub repo</A>.</P>
<P>HPL is a software package that solves a dense linear system in double precision arithmetic on distributed-memory computers.</P>
<ADDRESS>Innovative Computing Laboratory<BR>last revised December 2, 2018<BR></ADDRESS>
<PRE>
file    hpl-2.3.tar.gz
for     HPL 2.3 - A Portable Implementation of the High-Performance Linpack Benchmark
by      Antoine Petitet, Clint Whaley, Jack Dongarra, Andy Cleary, Piotr Luszczek
Updated: December 2, 2018
</PRE>
</BODY></HTML>`)

	got := PrepareHTML(raw, DefaultConfig())
	for _, want := range []string{
		"HPL - A Portable Implementation of the High-Performance",
		"Version 2.3",
		"A. Petitet",
		"R. C. Whaley",
		"J. Dongarra",
		"A. Cleary",
		"December 2, 2018",
		"<td align=\"CENTER\">",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output does not contain %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "RAW HTML FALLBACK") {
		t.Fatal("representative Netlib HPL HTML unexpectedly used the raw fallback")
	}
}

func TestPrepareHTMLNERSCPerlmutterArchitecturePage(t *testing.T) {
	// docs.nersc.gov.
	raw := []byte(`<!doctype html><html lang=en class=no-js><head>` +
		`<meta charset=utf-8><meta name=viewport content="width=device-width,initial-scale=1">` +
		`<meta name=description content="NERSC Documentation"><meta name=author content=NERSC>` +
		`<meta name=generator content="mkdocs-1.5.3, mkdocs-material-9.2.7">` +
		`<title>Architecture - NERSC Documentation</title>` +
		`<link rel=stylesheet href=../../../assets/stylesheets/main.min.css>` +
		`<style>` + strings.Repeat(`.md-nav{display:block}`, 2_000) + `</style>` +
		`<script>` + strings.Repeat(`window.__md_scope=true;`, 2_000) + `</script></head>` +
		`<body><header class="md-header"><nav><div class=md-header__title>` +
		`<span>NERSC Documentation</span><span>Architecture</span></div></nav></header>` +
		`<main class=md-main><div class="md-main__inner md-grid">` +
		`<div class="md-sidebar md-sidebar--primary"><nav class=md-nav>` +
		`<ul><li>Getting Started at NERSC</li><li>Running Jobs</li><li>Managing Data</li></ul>` +
		`</nav></div><div class=md-content data-md-component=content>` +
		`<article class="md-content__inner md-typeset">` +
		`<h1 id=perlmutter-architecture>Perlmutter Architecture<a class=headerlink href=#perlmutter-architecture>&para;</a></h1>` +
		`<p>Perlmutter is a HPE Cray EX supercomputer, named in honor of Saul Perlmutter.</p>` +
		`<p>Perlmutter is a heterogeneous system comprised of CPU-only and GPU-accelerated nodes.</p>` +
		`<h2 id=system-specifications>System Specifications<a class=headerlink href=#system-specifications>&para;</a></h2>` +
		`<table><thead><tr><th>Partition</th><th># of nodes</th><th>CPU</th><th>GPU</th></tr></thead>` +
		`<tbody><tr><td>GPU</td><td>1536</td><td>AMD EPYC 7763</td><td>NVIDIA A100</td></tr></tbody></table>` +
		`</article></div></div></main><footer class=md-footer>NERSC footer links</footer></body></html>`)

	got := PrepareHTML(raw, DefaultConfig())
	for _, want := range []string{
		"Architecture - NERSC Documentation",
		`<meta name="author" content="NERSC">`,
		`<h1 id="perlmutter-architecture">`,
		"Perlmutter Architecture",
		"Perlmutter is a HPE Cray EX supercomputer",
		"System Specifications",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output does not contain %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{strings.Repeat(".md-nav", 100), strings.Repeat("window.__md_scope", 100)} {
		if strings.Contains(got, unwanted) {
			t.Errorf("output contains MkDocs asset noise %q", unwanted[:20])
		}
	}
	if strings.Contains(got, "RAW HTML FALLBACK") {
		t.Fatal("representative NERSC documentation HTML unexpectedly used the raw fallback")
	}
}
