package wiki

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
	"gopkg.in/yaml.v3"
)

var md goldmark.Markdown

func init() {
	md = goldmark.New(
		goldmark.WithExtensions(
			extension.GFM, // tables, strikethrough, task lists, autolinks
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(), // allow raw HTML in markdown
		),
	)
}

// RenderMarkdown converts markdown content to HTML.
func RenderMarkdown(source []byte) (string, error) {
	_, body := SplitFrontmatter(source)
	var buf bytes.Buffer
	if err := md.Convert([]byte(body), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Frontmatter represents parsed YAML frontmatter.
type Frontmatter struct {
	Title    string   `yaml:"title"`
	Category string   `yaml:"category"`
	Tags     []string `yaml:"tags"`
	Summary  string   `yaml:"summary"`
	Source   string   `yaml:"source"`
}

// SplitFrontmatter splits a markdown file into frontmatter YAML and body.
func SplitFrontmatter(content []byte) (Frontmatter, string) {
	s := string(content)
	var fm Frontmatter

	if !strings.HasPrefix(s, "---") {
		return fm, s
	}

	parts := strings.SplitN(s[3:], "---", 2)
	if len(parts) != 2 {
		return fm, s
	}

	yaml.Unmarshal([]byte(parts[0]), &fm)
	return fm, strings.TrimLeft(parts[1], "\n")
}

// BuildFrontmatter creates YAML frontmatter from fields.
func BuildFrontmatter(fm Frontmatter) string {
	var b strings.Builder
	b.WriteString("---\n")
	if fm.Title != "" {
		b.WriteString("title: " + fm.Title + "\n")
	}
	if fm.Category != "" {
		b.WriteString("category: " + fm.Category + "\n")
	}
	if len(fm.Tags) > 0 {
		b.WriteString("tags: [" + strings.Join(fm.Tags, ", ") + "]\n")
	}
	if fm.Summary != "" {
		b.WriteString("summary: " + fm.Summary + "\n")
	}
	if fm.Source != "" {
		b.WriteString("source: " + fm.Source + "\n")
	}
	b.WriteString("---\n\n")
	return b.String()
}

var wikiLinkRe = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

// ExtractWikiLinks finds all [[wiki-link]] references in markdown content.
func ExtractWikiLinks(content []byte) []string {
	matches := wikiLinkRe.FindAllSubmatch(content, -1)
	var links []string
	seen := make(map[string]bool)
	for _, m := range matches {
		link := string(m[1])
		// Handle [[title|display]] syntax
		if idx := strings.Index(link, "|"); idx >= 0 {
			link = link[:idx]
		}
		link = strings.TrimSpace(link)
		if !seen[link] {
			links = append(links, link)
			seen[link] = true
		}
	}
	return links
}

// ResolveWikiLinks converts [[wiki-link]] syntax to HTML <a> tags.
// linkMap maps article titles (lowercase) to their URL paths.
func ResolveWikiLinks(html string, linkMap map[string]string) string {
	return wikiLinkRe.ReplaceAllStringFunc(html, func(match string) string {
		inner := match[2 : len(match)-2]
		title := inner
		display := inner
		if idx := strings.Index(inner, "|"); idx >= 0 {
			title = inner[:idx]
			display = inner[idx+1:]
		}
		title = strings.TrimSpace(title)
		display = strings.TrimSpace(display)

		if href, ok := linkMap[strings.ToLower(title)]; ok {
			return `<a href="` + href + `" class="wiki-link">` + display + `</a>`
		}
		// Dead link - style differently
		return `<span class="wiki-link-dead" title="Article not found">` + display + `</span>`
	})
}

// ExtractTitle attempts to get the title from frontmatter, falling back to first H1.
func ExtractTitle(content []byte) string {
	fm, body := SplitFrontmatter(content)
	if fm.Title != "" {
		return fm.Title
	}
	// Look for first H1
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return ""
}

// WordCount counts words in markdown body (excluding frontmatter).
func WordCount(content []byte) int {
	_, body := SplitFrontmatter(content)
	// Strip markdown syntax roughly
	body = wikiLinkRe.ReplaceAllString(body, "$1")
	words := strings.Fields(body)
	return len(words)
}
