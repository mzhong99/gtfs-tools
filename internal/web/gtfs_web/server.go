package gtfs_web

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	database "tarediiran-industries.com/gtfs-services/internal/db"
)

type GtfsWebServer struct {
	listenAddr string
	db         *database.Database

	router   chi.Router
	renderer *Renderer
}

func NewGtfsWebServer(ctx context.Context, listenAddr string, databaseUrl string) (*GtfsWebServer, error) {
	db, err := database.NewDatabaseConnection(ctx, databaseUrl)
	if err != nil {
		return nil, err
	}
	renderer, err := NewRenderer()
	if err != nil {
		return nil, err
	}

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	server := &GtfsWebServer{
		db:         db,
		listenAddr: listenAddr,
		router:     router,
		renderer:   renderer,
	}

	server.router.Get("/", func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, "/trains", http.StatusFound)
	})
	server.router.Get("/trains", server.handleTrainsPage)

	return server, nil
}

func (server *GtfsWebServer) Serve(ctx context.Context) {
	log.Printf("listening on http://localhost:%s", server.listenAddr)
	log.Fatal(http.ListenAndServe(server.listenAddr, server.router))
}
