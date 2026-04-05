package server

import "net/http"

func (s *Server) registerRoutes(mux *http.ServeMux) {
	// Existing doc-ui routes
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

	// Knowledge Base routes
	mux.HandleFunc("GET /kb", s.handleKBList)
	mux.HandleFunc("POST /kb", s.handleKBCreate)
	mux.HandleFunc("DELETE /kb/{id}", s.handleKBDelete)
	mux.HandleFunc("GET /kb/{id}", s.handleKBDashboard)
	mux.HandleFunc("POST /kb/{id}/sync", s.handleKBSync)

	// Articles
	mux.HandleFunc("GET /kb/{id}/articles", s.handleKBArticles)
	mux.HandleFunc("GET /kb/{id}/search", s.handleKBSearch)
	mux.HandleFunc("GET /kb/{id}/articles/{path...}", s.handleKBArticle)

	// Raw documents
	mux.HandleFunc("GET /kb/{id}/raw", s.handleRawList)
	mux.HandleFunc("POST /kb/{id}/raw/upload", s.handleRawUpload)
	mux.HandleFunc("POST /kb/{id}/raw/import-project", s.handleRawImportProject)

	// Pipeline
	mux.HandleFunc("POST /kb/{id}/compile", s.handleCompileStart)
	mux.HandleFunc("POST /kb/{id}/compile/stop", s.handleCompileStop)
	mux.HandleFunc("GET /kb/{id}/compile/progress", s.handleCompileProgress)

	// Chat / Q&A
	mux.HandleFunc("GET /kb/{id}/chat", s.handleKBChat)
	mux.HandleFunc("POST /kb/{id}/chat/ask", s.handleKBChatAsk)
	mux.HandleFunc("DELETE /kb/{id}/chat", s.handleKBChatClear)

	// Health
	mux.HandleFunc("GET /kb/{id}/health", s.handleKBHealth)
	mux.HandleFunc("POST /kb/{id}/health/run", s.handleHealthRun)
	mux.HandleFunc("POST /kb/{id}/health/{checkId}/resolve", s.handleHealthResolve)

	// Graph
	mux.HandleFunc("GET /kb/{id}/graph", s.handleKBGraph)
	mux.HandleFunc("GET /kb/{id}/graph.json", s.handleKBGraphData)
}
