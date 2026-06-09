package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gateframe/user-service/internal/config"
	"github.com/gateframe/user-service/internal/handler"
	"github.com/gateframe/user-service/internal/middleware"
	"github.com/gateframe/user-service/internal/migrate"
	"github.com/gateframe/user-service/internal/permversion"
	"github.com/gateframe/user-service/internal/repository"
	"github.com/gateframe/user-service/internal/service"
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
	var permStore permversion.Store = permversion.Noop{}
	if cfg.RedisURL != "" {
		redisStore, err := permversion.NewRedis(cfg.RedisURL)
		if err != nil {
			log.Fatalf("redis: %v", err)
		}
		permStore = redisStore
	}
	authSvc := service.NewAuthService(repo, cfg, permStore)
	userSvc := service.NewUserService(repo)
	rbacSvc := service.NewRBACService(repo, permStore)

	tenantSvc := service.NewTenantService(repo)
	dashboardSvc := service.NewDashboardService(repo)

	authH := handler.NewAuthHandler(authSvc)
	userH := handler.NewUserHandler(userSvc)
	rbacH := handler.NewRBACHandler(rbacSvc)
	tenantH := handler.NewTenantHandler(tenantSvc)
	dashboardH := handler.NewDashboardHandler(dashboardSvc)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	r.GET("/health", handler.Health)

	internal := r.Group("/internal/v1")
	internal.Use(middleware.RequireInternalToken(cfg.InternalToken))
	{
		internal.GET("/tenants/:id/status", tenantH.InternalStatus)
		internal.POST("/auth/oidc/resolve", authH.InternalOidcResolve)
	}

	api := r.Group("/api")
	api.Use(middleware.RequireInternalToken(cfg.InternalToken))
	{
		api.POST("/auth/login", authH.Login)

		authed := api.Group("")
		authed.Use(middleware.RequireIdentity())
		authed.Use(middleware.RequireActiveTenant(repo))
		authed.Use(handler.LoadPermissionsMiddleware(authSvc))
		{
			authed.GET("/auth/permissions", authH.Permissions)
			authed.POST("/auth/logout", authH.Logout)
			authed.POST("/auth/refresh", authH.Refresh)

			users := authed.Group("/users")
			users.GET("", middleware.RequirePermission("user.read"), userH.List)
			users.GET("/:id", middleware.RequirePermission("user.read"), userH.Get)
			users.POST("", middleware.RequirePermission("user.create"), userH.Create)
			users.PUT("/:id", middleware.RequirePermission("user.update"), userH.Update)
			users.DELETE("/:id", middleware.RequirePermission("user.delete"), userH.Delete)

			rbac := authed.Group("/rbac")
			rbac.GET("/roles", middleware.RequirePermission("rbac.read"), rbacH.ListRoles)
			rbac.GET("/permissions", middleware.RequirePermission("rbac.read"), rbacH.ListPermissions)
			rbac.GET("/role-permissions", middleware.RequirePermission("rbac.read"), rbacH.ListRolePermissions)
			rbac.GET("/roles/:id/permissions", middleware.RequirePermission("rbac.read"), rbacH.GetRolePermissions)
			rbac.PUT("/roles/:id/permissions", middleware.RequirePermission("rbac.manage"), rbacH.UpdateRolePermissions)

			tenants := authed.Group("/tenants")
			tenants.GET("", middleware.RequirePermission("tenant.read"), tenantH.List)
			tenants.POST("", middleware.RequirePermission("tenant.manage"), tenantH.Create)
			tenants.PUT("/:id", middleware.RequirePermission("tenant.manage"), tenantH.Update)

			authed.GET("/dashboard/stats", middleware.RequirePermission("user.read"), dashboardH.Stats)
		}
	}

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("user-service listening on %s", cfg.ListenAddr)
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
