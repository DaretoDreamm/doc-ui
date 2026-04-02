package crawler

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/microcosm-cc/bluemonday"
)

var sanitizer = bluemonday.UGCPolicy()

func init() {
	sanitizer.AllowAttrs("class").Globally()
	sanitizer.AllowAttrs("id").Globally()
	sanitizer.AllowElements("pre", "code", "h1", "h2", "h3", "h4", "h5", "h6",
		"p", "ul", "ol", "li", "a", "strong", "em", "blockquote", "table",
		"thead", "tbody", "tr", "th", "td", "img", "br", "hr", "div", "span",
		"dl", "dt", "dd", "figure", "figcaption", "details", "summary")
	sanitizer.AllowAttrs("href", "target").OnElements("a")
	sanitizer.AllowAttrs("src", "srcset", "alt", "width", "height", "loading").OnElements("img")
	sanitizer.AllowElements("picture", "source", "video", "audio")
	sanitizer.AllowAttrs("src", "srcset", "type", "media").OnElements("source")
}

func extractTitle(doc *goquery.Document) string {
	// Try <title> tag first, split on common separators
	if t := strings.TrimSpace(doc.Find("title").First().Text()); t != "" {
		// Strip site name suffixes like " | Android Developers" or " - React"
		for _, sep := range []string{" | ", " - ", " — ", " · "} {
			if idx := strings.Index(t, sep); idx > 0 {
				t = strings.TrimSpace(t[:idx])
				break
			}
		}
		if t != "" {
			return t
		}
	}

	// Try heading selectors
	selectors := []string{
		"article h1",
		"main h1",
		".devsite-article h1",
		"h1",
	}
	for _, sel := range selectors {
		h1 := doc.Find(sel).First()
		if h1.Length() == 0 {
			continue
		}
		// Get direct text, ignoring nested banner elements
		h1.Find("aside, .banner, .collections, [class*='organize']").Remove()
		if t := strings.TrimSpace(h1.Text()); t != "" {
			// Truncate overly long titles
			if len(t) > 100 {
				t = t[:100]
			}
			return t
		}
	}
	return "Untitled"
}

func extractContent(doc *goquery.Document) string {
	// Remove unwanted elements
	doc.Find("nav, header, footer, aside, script, style, noscript, .sidebar, .menu, .navigation, .nav, .toc, .breadcrumb, [role='navigation'], [role='banner'], [role='complementary'], .devsite-nav, .devsite-banner, .devsite-header, .devsite-toc").Remove()

	// Try content selectors in priority order
	selectors := []string{
		".devsite-article-body",
		".devsite-article",
		"main",
		"article",
		"[role='main']",
		".content",
		".main-content",
		"#content",
		"#main-content",
		".documentation",
		".doc-content",
		".markdown-body",
		".post-content",
		".entry-content",
	}

	for _, sel := range selectors {
		s := doc.Find(sel).First()
		if s.Length() > 0 {
			html, err := s.Html()
			if err == nil && strings.TrimSpace(html) != "" {
				return sanitizer.Sanitize(html)
			}
		}
	}

	// Fallback to body
	html, err := doc.Find("body").Html()
	if err != nil {
		return ""
	}
	return sanitizer.Sanitize(html)
}
