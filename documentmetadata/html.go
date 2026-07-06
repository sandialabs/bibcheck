// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package documentmetadata

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"strings"
	"unicode/utf8"

	xhtml "golang.org/x/net/html"
)

const maxVisibleBlockBytes = 8_000

// HTMLPrompt instructs the LLM how to interpret the selected HTML evidence.
const HTMLPrompt = `Extract the following from the provided website evidence:
- Title (string)
- Authors (array of string)
- Publication/Update Date (string, preferring YYYY-MM-DD, but YYYY-MM or YYYY is okay)

The input is a selected, possibly non-contiguous set of evidence from an HTML document. Labels describe where each excerpt came from. A RAW HTML FALLBACK section means that useful regions could not be identified reliably.

Use the following guidelines:
- Prefer user-visible information to hidden values such as HTML meta tags.
- The user wants the web page title, which may differ from the title of the whole website.
- Extract information about the document itself, not external links, references, navigation, or nearby unrelated content.
- Do not infer missing information merely because excerpts may be incomplete.
- Provide empty values when the requested information is not present.
- Produce JSON.
`

// Config controls how much selected and fallback HTML PrepareHTML returns.
type Config struct {
	SelectedTextBytes  int
	FallbackStartBytes int
	FallbackEndBytes   int
}

type excerpt struct {
	label string
	text  string
}

type visibleBlock struct {
	tag    string
	attrs  string
	text   strings.Builder
	markup strings.Builder
}

var blockTags = map[string]bool{
	"address": true, "article": true, "blockquote": true, "caption": true,
	"dd": true, "details": true, "div": true, "dt": true, "figcaption": true,
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"header": true, "li": true, "main": true, "p": true, "pre": true,
	"section": true, "summary": true, "td": true, "th": true, "time": true,
}

var evidenceTerms = []string{
	"author", "byline", "written by", "posted by", "published", "publication",
	"creator", "contributor", "maintainer", "owner", "updated", "modified",
	"revised", "released", "version", "date", "headline", "article-title", "citation_",
}

var metadataTerms = []string{
	"author", "byline", "title", "headline", "date", "published", "modified",
	"updated", "creator", "contributor", "revision", "citation", "dc.",
	"dcterms", "article:", "og:title",
}

// DefaultConfig returns the payload limits used by the metadata extraction clients.
func DefaultConfig() Config {
	return Config{
		SelectedTextBytes:  30_000,
		FallbackStartBytes: 20_000,
		FallbackEndBytes:   10_000,
	}
}

// PrepareHTML tries to extract only HTML relevant to determining website metadata.
// If no useful regions are found, it returns raw HTML
func PrepareHTML(raw []byte, config Config) (prepared string) {
	inputBytes := len(raw)
	defer func() {
		if inputBytes == 0 {
			log.Printf("HTML metadata preprocessing: %d -> %d bytes", inputBytes, len(prepared))
			return
		}
		log.Printf(
			"HTML metadata preprocessing: %d -> %d bytes",
			inputBytes,
			len(prepared),
		)
	}()

	z := xhtml.NewTokenizer(strings.NewReader(string(raw)))
	var excerpts []excerpt
	var blocks []visibleBlock
	var active []*visibleBlock
	var title strings.Builder
	var script strings.Builder
	inTitle, inJSONLD := false, false
	skipDepth := 0
	useful := false

	for {
		tt := z.Next()
		if tt == xhtml.ErrorToken {
			if z.Err() != nil && z.Err() != io.EOF {
				log.Printf("HTML metadata preprocessing failed; using raw fallback: %v", z.Err())
				return rawFallback(raw, config.FallbackStartBytes, config.FallbackEndBytes)
			}
			break
		}

		switch tt {
		// Opening / self-closing tags
		case xhtml.StartTagToken, xhtml.SelfClosingTagToken:
			t := z.Token()
			tag := strings.ToLower(t.Data)
			attrs := attributes(t.Attr)
			if skipDepth > 0 {
				if tt == xhtml.StartTagToken {
					skipDepth++
				}
				continue
			}
			switch tag {
			// Drop these elements, usually irrelevant
			case "style", "noscript", "svg", "template":
				if tt == xhtml.StartTagToken {
					skipDepth = 1
				}
				continue
			// Inspect JSON-LD scripts, all others skipped
			case "script":
				if strings.Contains(strings.ToLower(attrValue(t.Attr, "type")), "ld+json") {
					inJSONLD = true
					script.Reset()
				} else if tt == xhtml.StartTagToken {
					skipDepth = 1
				}
				continue
			// The document title is always useful evidence, even without keywords.
			case "title":
				inTitle = true
				title.Reset()
			// Relevant HTML meta elements provide title, author, and date evidence.
			case "meta":
				name := firstNonempty(attrValue(t.Attr, "name"), attrValue(t.Attr, "property"), attrValue(t.Attr, "itemprop"))
				content := attrValue(t.Attr, "content")
				if containsAny(strings.ToLower(name), metadataTerms) && strings.TrimSpace(content) != "" {
					excerpts = append(excerpts, excerpt{"meta " + name, t.String()})
					useful = true
				}
			}
			if blockTags[tag] && tt == xhtml.StartTagToken {
				b := &visibleBlock{tag: tag, attrs: attrs}
				b.markup.WriteString(t.String())
				active = append(active, b)
			} else if len(active) > 0 && tag != "title" && tag != "meta" {
				b := active[len(active)-1]
				b.markup.WriteString(t.String())
				if attrs != "" {
					b.attrs += " " + attrs
				}
			}

		// Text is accumulated only for the currently active metadata regions and
		// visible block elements.
		case xhtml.TextToken:
			if skipDepth > 0 {
				continue
			}
			text := string(z.Text())
			if inJSONLD {
				script.WriteString(text)
				continue
			}
			if inTitle {
				title.WriteString(text)
			}
			if len(active) > 0 {
				b := active[len(active)-1]
				appendTextPrefix(&b.text, text, maxVisibleBlockBytes)
				appendTextPrefix(&b.markup, html.EscapeString(text), maxVisibleBlockBytes)
			}

		// Closing tags finalize JSON-LD, titles, and visible blocks.
		case xhtml.EndTagToken:
			t := z.Token()
			tag := strings.ToLower(t.Data)
			if skipDepth > 0 {
				skipDepth--
				continue
			}
			if tag == "script" && inJSONLD {
				if values := jsonLDEvidence(script.String()); values != "" {
					excerpts = append(excerpts, excerpt{"JSON-LD", `<script type="application/ld+json">` + html.EscapeString(values) + `</script>`})
					useful = true
				}
				inJSONLD = false
				continue
			}
			if tag == "title" {
				if text := normalize(title.String()); text != "" {
					excerpts = append(excerpts, excerpt{"HTML title", "<title>" + html.EscapeString(text) + "</title>"})
					useful = true
				}
				inTitle = false
			}
			for i := len(active) - 1; i >= 0; i-- {
				if active[i].tag == tag {
					active[i].markup.WriteString("</")
					active[i].markup.WriteString(tag)
					active[i].markup.WriteByte('>')
					blocks = append(blocks, *active[i])
					active = append(active[:i], active[i+1:]...)
					break
				}
			}
			if len(active) > 0 && !blockTags[tag] {
				markup := &active[len(active)-1].markup
				markup.WriteString("</")
				markup.WriteString(tag)
				markup.WriteByte('>')
			}
		}
	}

	// Ignore empty layout containers so they do not consume positions in the
	// context window used for legacy table-based documents.
	nonempty := blocks[:0]
	for i := range blocks {
		if normalize(blocks[i].text.String()) != "" {
			nonempty = append(nonempty, blocks[i])
		}
	}
	blocks = nonempty

	// Find blocks that seem likely to contain useful data, plus two neighbors.
	selected := make(map[int]bool)
	for i := range blocks {
		text := normalize(blocks[i].text.String())
		combined := strings.ToLower(blocks[i].attrs + " " + text)
		candidate := strings.HasPrefix(blocks[i].tag, "h") || blocks[i].tag == "time" || blocks[i].tag == "address" || containsAny(combined, evidenceTerms)
		if candidate && (text != "" || blocks[i].attrs != "") {
			for j := max(0, i-2); j <= min(len(blocks)-1, i+2); j++ {
				selected[j] = true
			}
			useful = true
		}
	}
	for i := range blocks {
		if selected[i] {
			markup := strings.TrimSpace(blocks[i].markup.String())
			if markup != "" && normalize(blocks[i].text.String()) != "" {
				excerpts = append(excerpts, excerpt{fmt.Sprintf("selected <%s>", blocks[i].tag), markup})
			}
		}
	}

	if !useful {
		return rawFallback(raw, config.FallbackStartBytes, config.FallbackEndBytes)
	}
	return render(config.SelectedTextBytes, excerpts)
}

// appendTextPrefix appends text without allowing b to exceed limit bytes.
func appendTextPrefix(b *strings.Builder, text string, limit int) {
	remaining := limit - b.Len()
	if remaining <= 0 {
		return
	}
	if b.Len() > 0 {
		b.WriteByte(' ')
		remaining--
		if remaining <= 0 {
			return
		}
	}
	b.WriteString(validPrefix(text, remaining))
}

// attributes returns metadata-relevant HTML attributes as a string.
func attributes(attrs []xhtml.Attribute) string {
	var parts []string
	for _, a := range attrs {
		key := strings.ToLower(a.Key)
		if key == "class" || key == "id" || key == "itemprop" || key == "rel" || key == "datetime" {
			parts = append(parts, key+"="+a.Val)
		}
	}
	return strings.Join(parts, " ")
}

// attrValue returns an HTML attribute value using a case-insensitive key match.
func attrValue(attrs []xhtml.Attribute, key string) string {
	for _, a := range attrs {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

// firstNonempty returns the first non-empty string in values.
func firstNonempty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// containsAny reports whether s contains at least one of the terms.
func containsAny(s string, terms []string) bool {
	for _, term := range terms {
		if strings.Contains(s, term) {
			return true
		}
	}
	return false
}

// normalize decodes HTML entities and compresses whitespace.
func normalize(s string) string {
	return strings.Join(strings.Fields(html.UnescapeString(s)), " ")
}

// jsonLDEvidence extracts title, author, and publication-date fields from JSON-LD.
func jsonLDEvidence(raw string) string {
	var value any
	if json.Unmarshal([]byte(raw), &value) != nil {
		return ""
	}
	var found []string
	var walk func(any, []string)
	walk = func(v any, path []string) {
		switch v := v.(type) {
		case map[string]any:
			for key, child := range v {
				lower := strings.ToLower(key)
				if lower == "headline" || (lower == "name" && len(path) == 0) || lower == "author" || lower == "datepublished" || lower == "datemodified" {
					if encoded, err := json.Marshal(child); err == nil {
						found = append(found, key+": "+string(encoded))
					}
				}
				walk(child, append(path, lower))
			}
		case []any:
			for _, child := range v {
				walk(child, path)
			}
		}
	}
	walk(value, nil)
	return strings.Join(found, "\n")
}

// render formats unique HTML excerpts in discovery order and returns the first limit bytes.
func render(limit int, excerpts []excerpt) string {
	if limit <= 0 {
		return ""
	}
	var b strings.Builder
	seen := make(map[string]bool)
	for _, e := range excerpts {
		text := strings.TrimSpace(e.text)
		if text == "" {
			continue
		}
		key := strings.ToLower(text)
		if seen[key] {
			continue
		}
		seen[key] = true
		line := "<!-- " + e.label + " -->\n" + text + "\n\n"
		remaining := limit - b.Len()
		if remaining <= 0 {
			break
		}
		b.WriteString(validPrefix(line, remaining))
	}
	return b.String()
}

// rawFallback returns configured slices from the beginning and end of unprocessed HTML.
func rawFallback(raw []byte, startBytes, endBytes int) string {
	const header = "[RAW HTML FALLBACK: beginning]\n"
	const middle = "\n\n[RAW HTML FALLBACK: end]\n"
	startBytes = max(0, startBytes)
	endBytes = max(0, endBytes)
	if len(raw) <= startBytes+endBytes {
		return header + string(raw)
	}
	return header + validPrefix(string(raw), startBytes) + middle + validSuffix(string(raw), endBytes)
}

// validPrefix returns at most n bytes from the start of s without splitting UTF-8.
func validPrefix(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}

// validSuffix returns at most n bytes from the end of s without splitting UTF-8.
func validSuffix(s string, n int) string {
	if len(s) <= n {
		return s
	}
	start := len(s) - n
	for start < len(s) && !utf8.RuneStart(s[start]) {
		start++
	}
	return s[start:]
}
