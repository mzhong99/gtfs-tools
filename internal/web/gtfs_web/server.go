package gtfs_web

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	database "tarediiran-industries.com/gtfs-services/internal/db"
)

type GtfsWebServer struct {
	db       *database.Database
	server   *http.Server
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

	httpServer := &http.Server{
		Addr:    listenAddr,
		Handler: router,
	}

	server := &GtfsWebServer{
		db:       db,
		server:   httpServer,
		renderer: renderer,
	}

	router.Get("/", func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, "/trains", http.StatusFound)
	})
	router.Get("/trains", server.handleTrainsPage)
	router.Get("/trains/partial", server.handleTrainsPartial)

	return server, nil
}

func (server *GtfsWebServer) startHosting() {
	err := server.server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func (server *GtfsWebServer) Serve(ctx context.Context) {
	log.Printf("listening on http://localhost:%s", server.server.Addr)

	go server.startHosting()
	<-ctx.Done()

	log.Println("Shutting down.")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server.server.Shutdown(shutdownCtx)
}
