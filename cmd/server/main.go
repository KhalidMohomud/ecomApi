// Command server is the entrypoint for the E-commerce API.
//
// Its only job is composition: load config, open the database,
// build every dependency in order (repositories -> services ->
// handlers), hand them to the router, and start/stop the server. It
// contains no business logic — that all lives in internal/.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KhalidMohomud/ecomApi/internal/config"
	"github.com/KhalidMohomud/ecomApi/internal/database"
	"github.com/KhalidMohomud/ecomApi/internal/domain/repository"
	"github.com/KhalidMohomud/ecomApi/internal/handler"
	"github.com/KhalidMohomud/ecomApi/internal/routes"
	"github.com/KhalidMohomud/ecomApi/internal/service"
	"github.com/KhalidMohomud/ecomApi/internal/utils"
)

// @title			E-commerce API
// @version		1.0
// @description	Production-ready E-commerce REST API built with Go, Gin, GORM, and PostgreSQL.
// @contact.name	API Support
// @contact.email	alizakifarah@gmail.com
// @license.name	MIT
// @host			localhost:8080
// @BasePath		/api/v1
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				Type "Bearer" followed by a space and the JWT access token, e.g. "Bearer eyJhbGciOi...".
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(); err != nil {
		slog.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

// run contains the actual startup logic. Splitting it out of main
// lets it return an error instead of calling os.Exit directly,
// which keeps deferred cleanup (db.Close, etc.) actually running.
func run() error {
	cfg, err := config.Load(".env")
	if err != nil {
		return err
	}

	// A context bound to the process lifetime, cancelled the moment
	// the OS asks us to shut down (Ctrl+C locally, SIGTERM in Docker
	// or on Railway). We pass this down so long-running operations
	// know when to stop.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Connect(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer func() {
		if err := database.Close(db); err != nil {
			slog.Error("error closing database connection", "error", err)
		}
	}()

	// Composition root: every dependency is constructed explicitly,
	// in dependency order, and passed to whatever needs it. Nothing
	// here is a package-level variable — if you needed a second
	// server instance in a test, this whole chain could run again
	// with different arguments and no shared state.
	userRepo := repository.NewUserRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	brandRepo := repository.NewBrandRepository(db)
	productRepo := repository.NewProductRepository(db)

	tokenManager := utils.NewTokenManager(cfg.Auth.JWTSecret, cfg.Auth.AccessTokenTTL)

	authService := service.NewAuthService(userRepo, refreshTokenRepo, tokenManager, cfg.Auth.RefreshTokenTTL)
	userService := service.NewUserService(userRepo, refreshTokenRepo)
	adminService := service.NewAdminService(userRepo, refreshTokenRepo)
	categoryService := service.NewCategoryService(categoryRepo, productRepo)
	brandService := service.NewBrandService(brandRepo)
	productService := service.NewProductService(productRepo, categoryRepo, brandRepo)

	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	adminHandler := handler.NewAdminHandler(adminService)
	categoryHandler := handler.NewCategoryHandler(categoryService)
	brandHandler := handler.NewBrandHandler(brandService)
	productHandler := handler.NewProductHandler(productService)

	router := routes.SetupRouter(cfg, tokenManager, routes.Handlers{
		Auth:     authHandler,
		User:     userHandler,
		Admin:    adminHandler,
		Category: categoryHandler,
		Brand:    brandHandler,
		Product:  productHandler,
	})

	srv := &http.Server{
		Addr:              ":" + cfg.App.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("starting server", "port", cfg.App.Port, "env", cfg.App.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}

	slog.Info("server shut down cleanly")
	return nil
}
