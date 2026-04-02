package server

import "net/http"

func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", s.handleHome)
	mux.HandleFunc("POST /projects", s.handleCreateProject)
	mux.HandleFunc("POST /projects/{id}/recrawl", s.handleRecrawl)
	mux.HandleFunc("POST /projects/{id}/stop", s.handleStopCrawl)
	mux.HandleFunc("DELETE /projects/{id}", s.handleDeleteProject)
	mux.HandleFunc("GET /projects/{id}/progress", s.handleCrawlProgress)
	mux.HandleFunc("GET /docs/{id}", s.handleDocsIndex)
	mux.HandleFunc("GET /docs/{id}/search", s.handleDocSearch)
	mux.HandleFunc("GET /docs/{id}/{path...}", s.handleDocsPage)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
}
