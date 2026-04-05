package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/frkn/doc-ui/internal/database"
	"github.com/frkn/doc-ui/internal/llm"
	"github.com/frkn/doc-ui/internal/models"
	"github.com/frkn/doc-ui/internal/wiki"
)

// HealthChecker runs health checks on a knowledge base wiki.
type HealthChecker struct {
	db          *database.DB
	llmProvider llm.Provider
}

// NewHealthChecker creates a new HealthChecker.
func NewHealthChecker(db *database.DB, llmProvider llm.Provider) *HealthChecker {
	return &HealthChecker{db: db, llmProvider: llmProvider}
}

// RunChecks performs all health checks on a knowledge base.
func (h *HealthChecker) RunChecks(kb *models.KnowledgeBase) error {
	// Clear previous unresolved checks
	h.db.ClearHealthChecks(kb.ID)

	articles, err := h.db.ListArticles(kb.ID)
	if err != nil {
		return err
	}

	// 1. Orphan detection (no backlinks, not the index)
	for _, a := range articles {
		if a.FilePath == "index.md" {
			continue
		}
		if a.BacklinkCount == 0 {
			h.db.InsertHealthCheck(kb.ID, "orphan", "warning",
				fmt.Sprintf("Article '%s' has no incoming links from other articles", a.Title),
				a.ID)
		}
	}

	// 2. Dead wiki-links
	for _, a := range articles {
		content, err := wiki.ReadArticle(kb.BasePath, a.FilePath)
		if err != nil {
			continue
		}
		links := wiki.ExtractWikiLinks(content)
		titleSet := make(map[string]bool)
		for _, art := range articles {
			titleSet[strings.ToLower(art.Title)] = true
		}
		for _, link := range links {
			if !titleSet[strings.ToLower(link)] {
				h.db.InsertHealthCheck(kb.ID, "dead-link", "error",
					fmt.Sprintf("Article '%s' links to non-existent article '[[%s]]'", a.Title, link),
					a.ID)
			}
		}
	}

	// 3. Stale articles (articles that haven't been updated but have newer raw docs)
	rawDocs, _ := h.db.ListRawDocuments(kb.ID)
	for _, a := range articles {
		for _, d := range rawDocs {
			if d.Processed && d.CreatedAt.After(a.UpdatedAt) {
				h.db.InsertHealthCheck(kb.ID, "stale", "suggestion",
					fmt.Sprintf("Article '%s' may be outdated — raw document '%s' was added after it was last updated", a.Title, d.FileName),
					a.ID)
				break
			}
		}
	}

	// 4. LLM-powered checks (if provider available)
	if h.llmProvider != nil && len(articles) > 0 {
		h.runLLMChecks(kb, articles)
	}

	return nil
}

func (h *HealthChecker) runLLMChecks(kb *models.KnowledgeBase, articles []models.Article) {
	var summaries []string
	for _, a := range articles {
		summary := a.Summary
		if summary == "" {
			summary = "(no summary)"
		}
		summaries = append(summaries, fmt.Sprintf("**%s** [%s]: %s", a.Title, a.Category, summary))
	}

	prompt := HealthCheckPrompt(strings.Join(summaries, "\n"))
	resp, err := h.llmProvider.Chat(context.Background(), llm.ChatRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   2048,
		Temperature: 0.3,
	})
	if err != nil {
		log.Printf("health check LLM error: %v", err)
		return
	}

	var findings []struct {
		Type     string `json:"type"`
		Severity string `json:"severity"`
		Message  string `json:"message"`
		Article  string `json:"article"`
	}

	jsonStr := extractJSON(resp.Content)
	if err := json.Unmarshal([]byte(jsonStr), &findings); err != nil {
		log.Printf("health check parse error: %v", err)
		return
	}

	for _, f := range findings {
		var articleID int64
		if f.Article != "" {
			for _, a := range articles {
				if strings.EqualFold(a.Title, f.Article) {
					articleID = a.ID
					break
				}
			}
		}
		h.db.InsertHealthCheck(kb.ID, f.Type, f.Severity, f.Message, articleID)
	}
}
