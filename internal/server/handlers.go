package server

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/frkn/doc-ui/internal/models"
	"github.com/frkn/doc-ui/templates/components"
	"github.com/frkn/doc-ui/templates/doctemplates"
	"github.com/frkn/doc-ui/templates/pages"
)

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	projects, err := s.db.ListProjects()
	if err != nil {
		log.Printf("list projects: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	pages.Home(projects).Render(r.Context(), w)
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	url := strings.TrimSpace(r.FormValue("url"))
	template := strings.TrimSpace(r.FormValue("template"))
	colorPalette := strings.TrimSpace(r.FormValue("color_palette"))

	if name == "" || url == "" || template == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		http.Error(w, "URL must start with http:// or https://", http.StatusBadRequest)
		return
	}

	validTemplate := false
	for _, t := range models.Templates {
		if t.ID == template {
			validTemplate = true
			break
		}
	}
	if !validTemplate {
		http.Error(w, "Invalid template", http.StatusBadRequest)
		return
	}

	if colorPalette == "" {
		colorPalette = "ocean"
	}
	typography := strings.TrimSpace(r.FormValue("typography"))
	if typography == "" {
		typography = "modern"
	}

	project, err := s.db.CreateProject(name, url, template, colorPalette, typography)
	if err != nil {
		log.Printf("create project: %v", err)
		http.Error(w, "Failed to create project", http.StatusInternalServerError)
		return
	}

	// Reuse pages if this URL was already crawled
	existing, err := s.db.GetDoneProjectByURL(url)
	if err == nil && existing.ID != project.ID {
		count, copyErr := s.db.CopyPages(existing.ID, project.ID)
		if copyErr == nil && count > 0 {
			s.db.UpdateProjectStatus(project.ID, "done", count, "")
			project, _ = s.db.GetProject(project.ID)
			components.ProjectCard(*project).Render(r.Context(), w)
			return
		}
	}

	s.crawler.StartCrawl(project)
	components.ProjectCard(*project).Render(r.Context(), w)
}

func (s *Server) handleRecrawl(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	project, err := s.db.GetProject(id)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}
	// Stop any running crawl, wipe old pages, reset status
	s.crawler.StopCrawl(id)
	s.db.DeletePages(id)
	s.db.UpdateProjectStatus(id, "pending", 0, "")
	// Start fresh crawl
	project.Status = "pending"
	project.PageCount = 0
	s.crawler.StartCrawl(project)
	// Return full card with crawling state
	components.ProjectCard(*project).Render(r.Context(), w)
}

func (s *Server) handleStopCrawl(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	s.crawler.StopCrawl(id)
	// Give it a moment to finalize
	time.Sleep(500 * time.Millisecond)
	project, err := s.db.GetProject(id)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}
	components.CrawlStatus(*project).Render(r.Context(), w)
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	s.crawler.StopCrawl(id)
	if err := s.db.DeleteProject(id); err != nil {
		log.Printf("delete project %d: %v", id, err)
		http.Error(w, "Failed to delete", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleCrawlProgress(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	project, err := s.db.GetProject(id)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}
	components.CrawlStatus(*project).Render(r.Context(), w)
}

func (s *Server) handleDocSearch(w http.ResponseWriter, r *http.Request) {
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
	results, err := s.db.SearchPages(id, q, 12)
	if err != nil {
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}
	components.SearchResults(id, results).Render(r.Context(), w)
}

func (s *Server) handleDocsIndex(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	project, err := s.db.GetProject(id)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}
	allPages, err := s.db.GetPages(id)
	if err != nil || len(allPages) == 0 {
		http.Error(w, "No pages found", http.StatusNotFound)
		return
	}
	s.renderDoc(w, r, project, allPages, &allPages[0])
}

func (s *Server) handleDocsPage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	path := r.PathValue("path")
	if path == "" {
		http.Redirect(w, r, fmt.Sprintf("/docs/%d", id), http.StatusFound)
		return
	}
	project, err := s.db.GetProject(id)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}
	allPages, err := s.db.GetPages(id)
	if err != nil || len(allPages) == 0 {
		http.Error(w, "No pages found", http.StatusNotFound)
		return
	}
	page, err := s.db.GetPageByPath(id, path)
	if err == sql.ErrNoRows {
		path = strings.TrimSuffix(path, "/")
		page, err = s.db.GetPageByPath(id, path)
	}
	if err != nil {
		http.Error(w, "Page not found", http.StatusNotFound)
		return
	}
	s.renderDoc(w, r, project, allPages, page)
}

func (s *Server) renderDoc(w http.ResponseWriter, r *http.Request, project *models.Project, allPages []models.Page, current *models.Page) {
	var prevPage, nextPage *models.Page
	for i, p := range allPages {
		if p.ID == current.ID {
			if i > 0 {
				prevPage = &allPages[i-1]
			}
			if i < len(allPages)-1 {
				nextPage = &allPages[i+1]
			}
			break
		}
	}

	paletteCSS := models.GetPaletteCSS(project.ColorPalette)
	typoCSS := models.GetTypographyCSS(project.Typography)
	fontsLink := models.GetTypographyFontsLink(project.Typography)

	switch project.Template {
	case "sidebar":
		doctemplates.Sidebar(project, allPages, current, prevPage, nextPage, paletteCSS, typoCSS, fontsLink).Render(r.Context(), w)
	case "modern":
		doctemplates.Modern(project, allPages, current, prevPage, nextPage, paletteCSS, typoCSS, fontsLink).Render(r.Context(), w)
	default:
		doctemplates.Minimal(project, allPages, current, prevPage, nextPage, paletteCSS, typoCSS, fontsLink).Render(r.Context(), w)
	}
}
