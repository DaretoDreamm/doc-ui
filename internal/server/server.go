package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/frkn/doc-ui/internal/crawler"
	"github.com/frkn/doc-ui/internal/database"
)

type Server struct {
	db      *database.DB
	crawler *crawler.Crawler
	port    int
}

func New(db *database.DB, crawler *crawler.Crawler, port int) *Server {
	return &Server{
		db:      db,
		crawler: crawler,
		port:    port,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	s.registerRoutes(mux)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("Server starting on http://0.0.0.0%s", addr)
	return http.ListenAndServe(addr, mux)
}
