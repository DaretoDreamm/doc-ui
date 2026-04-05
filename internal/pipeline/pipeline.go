package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/frkn/doc-ui/internal/database"
	"github.com/frkn/doc-ui/internal/llm"
	"github.com/frkn/doc-ui/internal/models"
	"github.com/frkn/doc-ui/internal/wiki"
)

// Pipeline orchestrates the compilation of raw documents into wiki articles.
type Pipeline struct {
	db          *database.DB
	llmProvider llm.Provider
	wikiMgr     *wiki.Manager

	mu          sync.Mutex
	progressMap map[int64]*models.PipelineProgress
	cancelMap   map[int64]*atomic.Bool
}

// New creates a new Pipeline.
func New(db *database.DB, llmProvider llm.Provider, wikiMgr *wiki.Manager) *Pipeline {
	return &Pipeline{
		db:          db,
		llmProvider: llmProvider,
		wikiMgr:     wikiMgr,
		progressMap: make(map[int64]*models.PipelineProgress),
		cancelMap:   make(map[int64]*atomic.Bool),
	}
}

// GetProgress returns the current pipeline progress for a KB.
func (p *Pipeline) GetProgress(kbID int64) *models.PipelineProgress {
	p.mu.Lock()
	defer p.mu.Unlock()
	if prog, ok := p.progressMap[kbID]; ok {
		cp := *prog
		return &cp
	}
	return nil
}

func (p *Pipeline) setProgress(kbID int64, stage string, processed, total int, currentFile string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if prog, ok := p.progressMap[kbID]; ok {
		prog.Stage = stage
		prog.Processed = processed
		prog.Total = total
		prog.CurrentFile = currentFile
	}
}

func (p *Pipeline) setError(kbID int64, err string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if prog, ok := p.progressMap[kbID]; ok {
		prog.ErrorMsg = err
	}
}

// StartCompile begins the compilation pipeline for a knowledge base.
func (p *Pipeline) StartCompile(kb *models.KnowledgeBase) {
	p.mu.Lock()
	cancel := &atomic.Bool{}
	p.cancelMap[kb.ID] = cancel
	p.progressMap[kb.ID] = &models.PipelineProgress{
		KBID:  kb.ID,
		Stage: "starting",
	}
	p.mu.Unlock()

	p.db.UpdateKBStatus(kb.ID, "compiling")

	go func() {
		defer func() {
			p.mu.Lock()
			delete(p.progressMap, kb.ID)
			delete(p.cancelMap, kb.ID)
			p.mu.Unlock()
		}()

		if err := p.runPipeline(kb, cancel); err != nil {
			log.Printf("pipeline error for KB %d: %v", kb.ID, err)
			p.setError(kb.ID, err.Error())
			p.db.UpdateKBStatus(kb.ID, "idle")
			return
		}

		p.db.UpdateKBStatus(kb.ID, "idle")
		p.db.UpdateKBCounts(kb.ID)
	}()
}

// StopCompile cancels a running compilation.
func (p *Pipeline) StopCompile(kbID int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if cancel, ok := p.cancelMap[kbID]; ok {
		cancel.Store(true)
	}
}

func (p *Pipeline) runPipeline(kb *models.KnowledgeBase, cancel *atomic.Bool) error {
	if p.llmProvider == nil {
		return fmt.Errorf("no LLM provider configured")
	}

	ctx := context.Background()

	// Step 1: Get unprocessed raw documents
	rawDocs, err := p.db.ListUnprocessedRawDocuments(kb.ID)
	if err != nil {
		return fmt.Errorf("list raw docs: %w", err)
	}

	if len(rawDocs) == 0 {
		// Nothing to compile, just sync existing files
		p.setProgress(kb.ID, "syncing", 0, 0, "")
		p.wikiMgr.SyncFromDisk(kb)
		return nil
	}

	total := len(rawDocs)

	// Step 2: Summarize each raw document
	p.setProgress(kb.ID, "summarizing", 0, total, "")
	for i, doc := range rawDocs {
		if cancel.Load() {
			return fmt.Errorf("cancelled")
		}

		p.setProgress(kb.ID, "summarizing", i, total, doc.FileName)

		content, err := wiki.ReadRawDocument(kb.BasePath, doc.FilePath)
		if err != nil {
			log.Printf("pipeline: skip %s: %v", doc.FileName, err)
			continue
		}

		prompt := SummarizePrompt(string(content), doc.FileName)
		resp, err := p.llmProvider.Chat(ctx, llm.ChatRequest{
			Messages:    []llm.Message{{Role: "user", Content: prompt}},
			MaxTokens:   1024,
			Temperature: 0.3,
		})
		if err != nil {
			log.Printf("pipeline: summarize %s: %v", doc.FileName, err)
			continue
		}

		p.db.MarkRawProcessed(doc.ID, resp.Content)
	}

	// Step 3: Extract concepts
	p.setProgress(kb.ID, "extracting", 0, total, "")

	// Gather all summaries
	allDocs, _ := p.db.ListRawDocuments(kb.ID)
	var summaries []string
	for _, d := range allDocs {
		if d.Summary != "" {
			summaries = append(summaries, fmt.Sprintf("## %s\n%s", d.FileName, d.Summary))
		}
	}

	existingArticles, _ := p.db.ListArticles(kb.ID)
	var existingConcepts []string
	for _, a := range existingArticles {
		existingConcepts = append(existingConcepts, a.Title)
	}

	combinedSummary := strings.Join(summaries, "\n\n")
	conceptPrompt := ExtractConceptsPrompt(combinedSummary, existingConcepts)
	conceptResp, err := p.llmProvider.Chat(ctx, llm.ChatRequest{
		Messages:    []llm.Message{{Role: "user", Content: conceptPrompt}},
		MaxTokens:   2048,
		Temperature: 0.3,
	})
	if err != nil {
		return fmt.Errorf("extract concepts: %w", err)
	}

	var concepts []struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	// Extract JSON from response (may have surrounding text)
	jsonStr := extractJSON(conceptResp.Content)
	json.Unmarshal([]byte(jsonStr), &concepts)

	// Step 4: Generate articles for new concepts
	p.setProgress(kb.ID, "generating", 0, len(concepts), "")
	for i, concept := range concepts {
		if cancel.Load() {
			return fmt.Errorf("cancelled")
		}

		p.setProgress(kb.ID, "generating", i, len(concepts), concept.Title)

		// Check if article already exists
		slug := slugify(concept.Title)
		filePath := "concepts/" + slug + ".md"
		existing, err := p.db.GetArticleByPath(kb.ID, filePath)
		if err == nil && existing != nil {
			// Update existing article
			prompt := UpdateArticlePrompt(existing.Content, combinedSummary)
			resp, err := p.llmProvider.Chat(ctx, llm.ChatRequest{
				Messages:    []llm.Message{{Role: "user", Content: prompt}},
				MaxTokens:   4096,
				Temperature: 0.3,
			})
			if err != nil {
				log.Printf("pipeline: update %s: %v", concept.Title, err)
				continue
			}
			wiki.WriteArticle(kb.BasePath, filePath, []byte(resp.Content))
		} else {
			// Generate new article
			var relatedTitles []string
			for _, a := range existingArticles {
				relatedTitles = append(relatedTitles, a.Title)
			}

			prompt := WriteArticlePrompt(concept.Title, concept.Description, summaries, relatedTitles)
			resp, err := p.llmProvider.Chat(ctx, llm.ChatRequest{
				Messages:    []llm.Message{{Role: "user", Content: prompt}},
				MaxTokens:   4096,
				Temperature: 0.5,
			})
			if err != nil {
				log.Printf("pipeline: generate %s: %v", concept.Title, err)
				continue
			}
			wiki.WriteArticle(kb.BasePath, filePath, []byte(resp.Content))
		}
	}

	// Step 5: Sync filesystem to database
	p.setProgress(kb.ID, "syncing", 0, 0, "")
	p.wikiMgr.SyncFromDisk(kb)

	// Step 6: Update index
	p.setProgress(kb.ID, "indexing", 0, 0, "index.md")
	updatedArticles, _ := p.db.ListArticles(kb.ID)
	var articleList []string
	for _, a := range updatedArticles {
		articleList = append(articleList, fmt.Sprintf("- %s (category: %s, path: %s)", a.Title, a.Category, a.FilePath))
	}

	indexPrompt := CompileIndexPrompt(articleList)
	indexResp, err := p.llmProvider.Chat(ctx, llm.ChatRequest{
		Messages:    []llm.Message{{Role: "user", Content: indexPrompt}},
		MaxTokens:   4096,
		Temperature: 0.3,
	})
	if err == nil {
		wiki.WriteArticle(kb.BasePath, "index.md", []byte(indexResp.Content))
		// Re-sync to pick up index changes
		p.wikiMgr.SyncFromDisk(kb)
	}

	return nil
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		if r == ' ' || r == '_' {
			return '-'
		}
		return -1
	}, s)
	// Collapse multiple dashes
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}

func extractJSON(s string) string {
	// Find first [ and last ]
	start := strings.Index(s, "[")
	end := strings.LastIndex(s, "]")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return "[]"
}
