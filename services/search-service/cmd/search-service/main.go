package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gateframe/search-service/internal/config"
	"github.com/gateframe/search-service/internal/handler"
	"github.com/gateframe/search-service/internal/middleware"
	"github.com/gateframe/search-service/internal/migrate"
	"github.com/gateframe/search-service/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.Load()
	gin.SetMode(cfg.GinMode)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	if err := migrate.Up(ctx, pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	repo := repository.New(pool)
	h := handler.New(repo)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.GET("/health", h.Health)

	api := r.Group("/api")
	api.Use(middleware.RequireInternalToken(cfg.InternalToken))
	api.Use(middleware.RequireIdentity())
	api.Use(middleware.LoadPermissionsFromGateway())
	{
		api.GET("/search", middleware.RequirePermission("search.read"), h.Search)
		api.POST("/search/documents", middleware.RequirePermission("search.manage"), h.CreateDocument)
		api.DELETE("/search/documents/:id", middleware.RequirePermission("search.manage"), h.DeleteDocument)
	}

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("search-service listening on %s", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
