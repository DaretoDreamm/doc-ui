package wiki

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/frkn/doc-ui/internal/database"
	"github.com/frkn/doc-ui/internal/models"
)

// Manager provides high-level wiki operations combining filesystem and database.
type Manager struct {
	db *database.DB
}

// NewManager creates a new wiki Manager.
func NewManager(db *database.DB) *Manager {
	return &Manager{db: db}
}

// SyncFromDisk scans the wiki directory and upserts all articles into SQLite.
// Returns the number of articles synced.
func (m *Manager) SyncFromDisk(kb *models.KnowledgeBase) (int, error) {
	files, err := ListMarkdownFiles(kb.BasePath)
	if err != nil {
		return 0, fmt.Errorf("list files: %w", err)
	}

	count := 0
	for _, relPath := range files {
		content, err := ReadArticle(kb.BasePath, relPath)
		if err != nil {
			log.Printf("sync: skip %s: %v", relPath, err)
			continue
		}

		hash := ComputeHash(content)

		// Check if content changed
		existing, err := m.db.GetArticleByPath(kb.ID, relPath)
		if err == nil && existing.ContentHash == hash {
			continue // no change
		}

		fm, _ := SplitFrontmatter(content)
		title := fm.Title
		if title == "" {
			title = ExtractTitle(content)
		}
		if title == "" {
			// Use filename as title
			title = strings.TrimSuffix(relPath, ".md")
			if idx := strings.LastIndex(title, "/"); idx >= 0 {
				title = title[idx+1:]
			}
			title = strings.ReplaceAll(title, "-", " ")
			title = strings.Title(title)
		}

		summary := fm.Summary
		if summary == "" && len(content) > 200 {
			_, body := SplitFrontmatter(content)
			if len(body) > 200 {
				summary = body[:200] + "..."
			} else {
				summary = body
			}
		}

		tags := "[]"
		if len(fm.Tags) > 0 {
			b, _ := json.Marshal(fm.Tags)
			tags = string(b)
		}

		source := fm.Source
		if source == "" {
			source = "manual"
		}

		wc := WordCount(content)

		_, err = m.db.UpsertArticle(kb.ID, relPath, title, summary, string(content), hash, fm.Category, tags, source, wc)
		if err != nil {
			log.Printf("sync: upsert %s: %v", relPath, err)
			continue
		}
		count++
	}

	// Resolve backlinks after sync
	m.ResolveBacklinks(kb)
	m.db.UpdateKBCounts(kb.ID)

	return count, nil
}

// ResolveBacklinks scans all articles for [[wiki-links]] and updates the backlinks table.
func (m *Manager) ResolveBacklinks(kb *models.KnowledgeBase) {
	articles, err := m.db.ListArticles(kb.ID)
	if err != nil {
		return
	}

	// Build title->article map
	titleMap := make(map[string]*models.Article)
	for i := range articles {
		titleMap[strings.ToLower(articles[i].Title)] = &articles[i]
	}

	for _, article := range articles {
		content, err := ReadArticle(kb.BasePath, article.FilePath)
		if err != nil {
			continue
		}

		m.db.ClearBacklinksForArticle(article.ID)

		links := ExtractWikiLinks(content)
		for _, linkTitle := range links {
			target, ok := titleMap[strings.ToLower(linkTitle)]
			if !ok {
				continue
			}
			if target.ID == article.ID {
				continue // no self-links
			}
			m.db.UpsertBacklink(article.ID, target.ID, "")
		}
	}

	m.db.UpdateBacklinkCounts(kb.ID)
}

// BuildLinkMap creates a map from lowercase article title to URL path for wiki link resolution.
func (m *Manager) BuildLinkMap(kbID int64) map[string]string {
	articles, err := m.db.ListArticles(kbID)
	if err != nil {
		return nil
	}
	linkMap := make(map[string]string)
	for _, a := range articles {
		path := strings.TrimSuffix(a.FilePath, ".md")
		linkMap[strings.ToLower(a.Title)] = fmt.Sprintf("/kb/%d/articles/%s", kbID, path)
	}
	return linkMap
}

// RenderArticle reads an article from disk, renders markdown to HTML, and resolves wiki links.
func (m *Manager) RenderArticle(kb *models.KnowledgeBase, article *models.Article) (string, error) {
	content, err := ReadArticle(kb.BasePath, article.FilePath)
	if err != nil {
		// Fall back to database content
		content = []byte(article.Content)
	}

	html, err := RenderMarkdown(content)
	if err != nil {
		return "", err
	}

	linkMap := m.BuildLinkMap(kb.ID)
	html = ResolveWikiLinks(html, linkMap)

	return html, nil
}
