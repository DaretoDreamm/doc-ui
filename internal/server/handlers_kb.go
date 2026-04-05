package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/frkn/doc-ui/internal/llm"
	"github.com/frkn/doc-ui/internal/models"
	"github.com/frkn/doc-ui/internal/pipeline"
	"github.com/frkn/doc-ui/internal/wiki"
	kbpages "github.com/frkn/doc-ui/templates/kbpages"
)

// --- Knowledge Base CRUD ---

func (s *Server) handleKBList(w http.ResponseWriter, r *http.Request) {
	kbs, err := s.db.ListKnowledgeBases()
	if err != nil {
		log.Printf("list kbs: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	kbpages.KBList(kbs, s.llm != nil).Render(r.Context(), w)
}

func (s *Server) handleKBCreate(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	description := strings.TrimSpace(r.FormValue("description"))
	colorPalette := strings.TrimSpace(r.FormValue("color_palette"))
	typography := strings.TrimSpace(r.FormValue("typography"))

	if name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}
	if colorPalette == "" {
		colorPalette = "ocean"
	}
	if typography == "" {
		typography = "modern"
	}

	slug := slugify(name)
	// Ensure unique slug
	base := slug
	counter := 1
	for s.db.SlugExists(slug) {
		slug = fmt.Sprintf("%s-%d", base, counter)
		counter++
	}

	basePath := filepath.Join(".", "data", "kb", slug)
	if err := wiki.InitKBDirectory(basePath); err != nil {
		log.Printf("init kb dir: %v", err)
		http.Error(w, "Failed to create directory", http.StatusInternalServerError)
		return
	}

	kb, err := s.db.CreateKnowledgeBase(name, slug, description, basePath, colorPalette, typography)
	if err != nil {
		log.Printf("create kb: %v", err)
		http.Error(w, "Failed to create knowledge base", http.StatusInternalServerError)
		return
	}

	// Sync initial files (index.md)
	s.wiki.SyncFromDisk(kb)

	kbpages.KBCard(*kb).Render(r.Context(), w)
}

func (s *Server) handleKBDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	if s.pipeline != nil {
		s.pipeline.StopCompile(id)
	}
	if err := s.db.DeleteKnowledgeBase(id); err != nil {
		log.Printf("delete kb %d: %v", id, err)
		http.Error(w, "Failed to delete", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleKBDashboard(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	kb, err := s.db.GetKnowledgeBase(id)
	if err != nil {
		http.Error(w, "Knowledge base not found", http.StatusNotFound)
		return
	}
	// Refresh counts
	s.db.UpdateKBCounts(kb.ID)
	kb, _ = s.db.GetKnowledgeBase(id)

	articles, _ := s.db.ListArticles(kb.ID)
	rawDocs, _ := s.db.ListRawDocuments(kb.ID)
	healthChecks, _ := s.db.ListHealthChecks(kb.ID)
	projects, _ := s.db.ListProjects()

	var pipelineProgress *models.PipelineProgress
	if s.pipeline != nil {
		pipelineProgress = s.pipeline.GetProgress(kb.ID)
	}

	kbpages.KBDashboard(kb, articles, rawDocs, healthChecks, pipelineProgress, s.llm != nil, projects).Render(r.Context(), w)
}

func (s *Server) handleKBSync(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	kb, err := s.db.GetKnowledgeBase(id)
	if err != nil {
		http.Error(w, "Knowledge base not found", http.StatusNotFound)
		return
	}
	count, err := s.wiki.SyncFromDisk(kb)
	if err != nil {
		http.Error(w, "Sync failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Trigger", "kb-updated")
	fmt.Fprintf(w, `<span class="text-emerald-400 text-xs font-mono">Synced %d articles</span>`, count)
}

// --- Articles ---

func (s *Server) handleKBArticles(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	kb, err := s.db.GetKnowledgeBase(id)
	if err != nil {
		http.Error(w, "Knowledge base not found", http.StatusNotFound)
		return
	}
	articles, _ := s.db.ListArticles(kb.ID)
	if len(articles) == 0 {
		http.Redirect(w, r, fmt.Sprintf("/kb/%d", id), http.StatusFound)
		return
	}
	// Show first article
	article, err := s.db.GetArticleByPath(kb.ID, articles[0].FilePath)
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/kb/%d", id), http.StatusFound)
		return
	}
	s.renderWikiArticle(w, r, kb, articles, article)
}

func (s *Server) handleKBArticle(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	path := r.PathValue("path")
	if path == "" {
		http.Redirect(w, r, fmt.Sprintf("/kb/%d/articles", id), http.StatusFound)
		return
	}
	// Add .md extension if missing
	if !strings.HasSuffix(path, ".md") {
		path += ".md"
	}

	kb, err := s.db.GetKnowledgeBase(id)
	if err != nil {
		http.Error(w, "Knowledge base not found", http.StatusNotFound)
		return
	}

	article, err := s.db.GetArticleByPath(kb.ID, path)
	if err != nil {
		http.Error(w, "Article not found", http.StatusNotFound)
		return
	}

	articles, _ := s.db.ListArticles(kb.ID)
	s.renderWikiArticle(w, r, kb, articles, article)
}

func (s *Server) renderWikiArticle(w http.ResponseWriter, r *http.Request, kb *models.KnowledgeBase, allArticles []models.Article, article *models.Article) {
	html, err := s.wiki.RenderArticle(kb, article)
	if err != nil {
		html = "<p>Error rendering article</p>"
	}

	backlinks, _ := s.db.GetBacklinksForArticle(article.ID)

	paletteCSS := models.GetPaletteCSS(kb.ColorPalette)
	typoCSS := models.GetTypographyCSS(kb.Typography)
	fontsLink := models.GetTypographyFontsLink(kb.Typography)

	kbpages.WikiArticle(kb, allArticles, article, html, backlinks, paletteCSS, typoCSS, fontsLink).Render(r.Context(), w)
}

func (s *Server) handleKBSearch(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		w.Write([]byte(""))
		return
	}
	results, err := s.db.SearchArticles(id, q, 12)
	if err != nil {
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}
	kbpages.KBSearchResults(id, results).Render(r.Context(), w)
}

// --- Raw Documents ---

func (s *Server) handleRawList(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	kb, err := s.db.GetKnowledgeBase(id)
	if err != nil {
		http.Error(w, "Knowledge base not found", http.StatusNotFound)
		return
	}
	docs, _ := s.db.ListRawDocuments(kb.ID)
	kbpages.RawDocList(kb, docs).Render(r.Context(), w)
}

func (s *Server) handleRawUpload(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	kb, err := s.db.GetKnowledgeBase(id)
	if err != nil {
		http.Error(w, "Knowledge base not found", http.StatusNotFound)
		return
	}

	// Parse multipart form (max 50MB)
	r.ParseMultipartForm(50 << 20)

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "File upload failed", http.StatusBadRequest)
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Read file failed", http.StatusInternalServerError)
		return
	}

	fileName := header.Filename
	mimeType := wiki.MimeTypeFromExt(fileName)

	if err := wiki.WriteRawDocument(kb.BasePath, fileName, content); err != nil {
		http.Error(w, "Save file failed", http.StatusInternalServerError)
		return
	}

	s.db.InsertRawDocument(kb.ID, fileName, fileName, mimeType, int64(len(content)))
	s.db.UpdateKBCounts(kb.ID)

	// Return updated raw doc list
	docs, _ := s.db.ListRawDocuments(kb.ID)
	kbpages.RawDocItems(kb, docs).Render(r.Context(), w)
}

func (s *Server) handleRawImportProject(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	kb, err := s.db.GetKnowledgeBase(id)
	if err != nil {
		http.Error(w, "Knowledge base not found", http.StatusNotFound)
		return
	}

	projectID, err := strconv.ParseInt(r.FormValue("project_id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	project, err := s.db.GetProject(projectID)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	pages, err := s.db.GetPages(projectID)
	if err != nil || len(pages) == 0 {
		http.Error(w, "No pages to import", http.StatusBadRequest)
		return
	}

	projectSlug := slugify(project.Name)
	imported := 0

	for _, page := range pages {
		if strings.TrimSpace(page.Content) == "" {
			continue
		}

		// Build page slug from path or title
		pageSlug := page.Path
		if pageSlug == "" {
			pageSlug = slugify(page.Title)
		}
		pageSlug = strings.ReplaceAll(pageSlug, "/", "-")
		pageSlug = strings.ReplaceAll(pageSlug, "\\", "-")
		if pageSlug == "" {
			pageSlug = fmt.Sprintf("page-%d", page.ID)
		}

		// 1. Write as wiki article (immediately browsable)
		fm := wiki.BuildFrontmatter(wiki.Frontmatter{
			Title:    page.Title,
			Category: project.Name,
			Source:   "imported",
		})
		articleContent := fm + page.Content
		articlePath := fmt.Sprintf("imported/%s/%s.md", projectSlug, pageSlug)
		if err := wiki.WriteArticle(kb.BasePath, articlePath, []byte(articleContent)); err != nil {
			log.Printf("import wiki article %s: %v", page.Title, err)
			continue
		}

		// 2. Also save to raw/ for potential LLM re-processing later
		rawFileName := projectSlug + "_" + pageSlug + ".html"
		wiki.WriteRawDocument(kb.BasePath, rawFileName, []byte(page.Content))
		mimeType := wiki.MimeTypeFromExt(rawFileName)
		s.db.InsertRawDocument(kb.ID, rawFileName, rawFileName, mimeType, int64(len(page.Content)))

		imported++
	}

	// Sync wiki files into SQLite so articles appear immediately
	s.wiki.SyncFromDisk(kb)
	s.db.UpdateKBCounts(kb.ID)
	log.Printf("Imported %d pages from project '%s' into KB '%s'", imported, project.Name, kb.Name)

	// Redirect to dashboard so user sees the articles
	w.Header().Set("HX-Redirect", fmt.Sprintf("/kb/%d", kb.ID))
	w.WriteHeader(http.StatusOK)
}

// --- Pipeline ---

func (s *Server) handleCompileStart(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	if s.pipeline == nil || s.llm == nil {
		http.Error(w, "LLM not configured — set LLM_PROVIDER and LLM_API_KEY env vars", http.StatusBadRequest)
		return
	}
	kb, err := s.db.GetKnowledgeBase(id)
	if err != nil {
		http.Error(w, "Knowledge base not found", http.StatusNotFound)
		return
	}
	s.pipeline.StartCompile(kb)
	kbpages.CompileProgress(kb.ID, &models.PipelineProgress{KBID: kb.ID, Stage: "starting"}).Render(r.Context(), w)
}

func (s *Server) handleCompileStop(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	if s.pipeline != nil {
		s.pipeline.StopCompile(id)
	}
	fmt.Fprintf(w, `<span class="text-zinc-400 text-xs font-mono">Compilation stopped</span>`)
}

func (s *Server) handleCompileProgress(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	kb, err := s.db.GetKnowledgeBase(id)
	if err != nil {
		http.Error(w, "Knowledge base not found", http.StatusNotFound)
		return
	}

	var progress *models.PipelineProgress
	if s.pipeline != nil {
		progress = s.pipeline.GetProgress(id)
	}

	if progress == nil {
		// Pipeline finished
		kbpages.CompileDone(kb).Render(r.Context(), w)
		return
	}
	kbpages.CompileProgress(kb.ID, progress).Render(r.Context(), w)
}

// --- Chat / Q&A ---

func (s *Server) handleKBChat(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	kb, err := s.db.GetKnowledgeBase(id)
	if err != nil {
		http.Error(w, "Knowledge base not found", http.StatusNotFound)
		return
	}
	messages, _ := s.db.ListChatMessages(kb.ID, 50)
	articles, _ := s.db.ListArticles(kb.ID)

	paletteCSS := models.GetPaletteCSS(kb.ColorPalette)
	typoCSS := models.GetTypographyCSS(kb.Typography)
	fontsLink := models.GetTypographyFontsLink(kb.Typography)

	kbpages.KBChat(kb, messages, articles, s.llm != nil, paletteCSS, typoCSS, fontsLink).Render(r.Context(), w)
}

func (s *Server) handleKBChatAsk(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	if s.llm == nil {
		http.Error(w, "LLM not configured", http.StatusBadRequest)
		return
	}

	kb, err := s.db.GetKnowledgeBase(id)
	if err != nil {
		http.Error(w, "Knowledge base not found", http.StatusNotFound)
		return
	}

	question := strings.TrimSpace(r.FormValue("question"))
	if question == "" {
		http.Error(w, "Question is required", http.StatusBadRequest)
		return
	}

	// Save user message
	s.db.InsertChatMessage(kb.ID, "user", question)

	// Build system prompt with wiki context
	articles, _ := s.db.ListArticles(kb.ID)
	var articleSummary strings.Builder
	for _, a := range articles {
		summary := a.Summary
		if summary == "" {
			summary = "(no summary)"
		}
		articleSummary.WriteString(fmt.Sprintf("- **%s** [%s]: %s\n", a.Title, a.FilePath, summary))
	}

	systemPrompt := pipeline.QASystemPrompt(articleSummary.String())

	// Build conversation history
	history, _ := s.db.ListChatMessages(kb.ID, 20)
	var messages []llm.Message
	for _, m := range history {
		messages = append(messages, llm.Message{Role: m.Role, Content: m.Content})
	}

	// SSE streaming response
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	var fullResponse strings.Builder
	ctx := r.Context()

	err = s.llm.ChatStream(ctx, llm.ChatRequest{
		SystemPrompt: systemPrompt,
		Messages:     messages,
		MaxTokens:    4096,
		Temperature:  0.5,
	}, func(chunk string) {
		fullResponse.WriteString(chunk)
		// Escape for SSE
		escaped := strings.ReplaceAll(chunk, "\n", "\\n")
		fmt.Fprintf(w, "data: %s\n\n", escaped)
		flusher.Flush()
	})

	if err != nil {
		fmt.Fprintf(w, "data: [ERROR] %s\n\n", err.Error())
		flusher.Flush()
	}

	// Save assistant response
	if fullResponse.Len() > 0 {
		s.db.InsertChatMessage(kb.ID, "assistant", fullResponse.String())
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func (s *Server) handleKBChatClear(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	s.db.ClearChatHistory(id)
	w.WriteHeader(http.StatusOK)
}

// --- Health Checks ---

func (s *Server) handleKBHealth(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	kb, err := s.db.GetKnowledgeBase(id)
	if err != nil {
		http.Error(w, "Knowledge base not found", http.StatusNotFound)
		return
	}
	checks, _ := s.db.ListHealthChecks(kb.ID)
	kbpages.KBHealth(kb, checks, s.llm != nil).Render(r.Context(), w)
}

func (s *Server) handleHealthRun(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	kb, err := s.db.GetKnowledgeBase(id)
	if err != nil {
		http.Error(w, "Knowledge base not found", http.StatusNotFound)
		return
	}

	checker := pipeline.NewHealthChecker(s.db, s.llm)
	if err := checker.RunChecks(kb); err != nil {
		http.Error(w, "Health check failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	checks, _ := s.db.ListHealthChecks(kb.ID)
	kbpages.HealthCheckResults(checks).Render(r.Context(), w)
}

func (s *Server) handleHealthResolve(w http.ResponseWriter, r *http.Request) {
	checkID, err := strconv.ParseInt(r.PathValue("checkId"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	s.db.ResolveHealthCheck(checkID)
	w.WriteHeader(http.StatusOK)
}

// --- Graph ---

func (s *Server) handleKBGraph(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	kb, err := s.db.GetKnowledgeBase(id)
	if err != nil {
		http.Error(w, "Knowledge base not found", http.StatusNotFound)
		return
	}
	kbpages.KBGraph(kb).Render(r.Context(), w)
}

func (s *Server) handleKBGraphData(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	articles, backlinks, err := s.db.GetGraphData(id)
	if err != nil {
		http.Error(w, "Failed to get graph data", http.StatusInternalServerError)
		return
	}

	type Node struct {
		ID       int64  `json:"id"`
		Title    string `json:"title"`
		Category string `json:"category"`
		Size     int    `json:"size"`
		Path     string `json:"path"`
	}
	type Edge struct {
		Source int64 `json:"source"`
		Target int64 `json:"target"`
	}
	type GraphData struct {
		Nodes []Node `json:"nodes"`
		Edges []Edge `json:"edges"`
	}

	data := GraphData{}
	for _, a := range articles {
		data.Nodes = append(data.Nodes, Node{
			ID:       a.ID,
			Title:    a.Title,
			Category: a.Category,
			Size:     a.WordCount,
			Path:     strings.TrimSuffix(a.FilePath, ".md"),
		})
	}
	for _, b := range backlinks {
		data.Edges = append(data.Edges, Edge{
			Source: b.SourceArticleID,
			Target: b.TargetArticleID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// --- Helpers ---

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

