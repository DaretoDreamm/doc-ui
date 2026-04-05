package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/frkn/doc-ui/internal/crawler"
	"github.com/frkn/doc-ui/internal/database"
	"github.com/frkn/doc-ui/internal/llm"
	"github.com/frkn/doc-ui/internal/pipeline"
	"github.com/frkn/doc-ui/internal/wiki"
)

type Server struct {
	db       *database.DB
	crawler  *crawler.Crawler
	wiki     *wiki.Manager
	llm      llm.Provider
	pipeline *pipeline.Pipeline
	port     int
}

func New(db *database.DB, crawler *crawler.Crawler, wikiMgr *wiki.Manager, llmProvider llm.Provider, port int) *Server {
	var pl *pipeline.Pipeline
	if wikiMgr != nil {
		pl = pipeline.New(db, llmProvider, wikiMgr)
	}
	return &Server{
		db:       db,
		crawler:  crawler,
		wiki:     wikiMgr,
		llm:      llmProvider,
		pipeline: pl,
		port:     port,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	s.registerRoutes(mux)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("Server starting on http://0.0.0.0%s", addr)
	return http.ListenAndServe(addr, mux)
}
