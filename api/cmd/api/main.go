package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/vasi1796/doit/internal/auth"
	"github.com/vasi1796/doit/internal/config"
	"github.com/vasi1796/doit/internal/domain"
	"github.com/vasi1796/doit/internal/eventstore"
	"github.com/vasi1796/doit/internal/handler"
	"github.com/vasi1796/doit/internal/middleware"
	"github.com/vasi1796/doit/internal/projection"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger := newLogger(cfg.LogLevel, cfg.LogFormat)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := connectDB(ctx, cfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer pool.Close()

	if cfg.DevMode {
		logger.Warn().Msg("DEV_MODE is enabled — do not use in production")
	}

	r := newRouter(pool, logger, cfg)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: r,
	}

	go func() {
		logger.Info().Int("port", cfg.Port).Str("log_level", cfg.LogLevel).Msg("starting server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("server failed")
		}
	}()

	<-ctx.Done()
	logger.Info().Msg("shutting down server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("server shutdown error")
	}

	logger.Info().Msg("server stopped")
}

func newLogger(level, format string) zerolog.Logger {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}

	var output = os.Stdout
	if format == "console" {
		writer := zerolog.ConsoleWriter{Out: output}
		return zerolog.New(writer).Level(lvl).With().Timestamp().Logger()
	}

	return zerolog.New(output).Level(lvl).With().Timestamp().Logger()
}

func connectDB(ctx context.Context, cfg *config.Config, logger zerolog.Logger) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing database URL: %w", err)
	}

	poolCfg.MaxConns = int32(cfg.DBMaxOpenConns)
	poolCfg.MinConns = int32(cfg.DBMaxIdleConns)
	poolCfg.MaxConnLifetime = cfg.DBConnMaxLifetime

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	logger.Info().Msg("connected to database")
	return pool, nil
}

func newRouter(pool *pgxpool.Pool, logger zerolog.Logger, cfg *config.Config) *chi.Mux {
	r := chi.NewRouter()
	r.Use(chimw.Recoverer)
	r.Use(requestLogger(logger))

	// CORS
	allowedOrigins := cfg.CORSOrigins
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"https://localhost", "http://localhost", "http://localhost:5173"}
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check (unauthenticated)
	r.Head("/healthz", healthHandler(pool, logger))
	r.Get("/healthz", healthHandler(pool, logger))

	// Auth
	tokenSvc := auth.NewTokenService(cfg.JWTSecret, cfg.JWTExpiryHours)
	googleOAuth := auth.NewGoogleOAuth(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL)
	authHandler := handler.NewAuthHandler(googleOAuth, tokenSvc, pool, cfg.AllowedEmails, logger, cfg.FrontendURL, cfg.DevMode, cfg.SecureCookies)

	r.Route("/auth", func(r chi.Router) {
		r.Get("/google/login", authHandler.GoogleLogin)
		r.Get("/google/callback", authHandler.GoogleCallback)
		r.Post("/logout", authHandler.Logout)
		if cfg.DevMode {
			r.Post("/dev", authHandler.DevLogin)
		}
	})

	// Domain stack
	store := eventstore.New(pool, logger)
	projector := projection.New(pool, logger)
	cmdHandler := domain.NewCommandHandler(store, projector)

	// Protected API routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.JWTAuth(tokenSvc, logger))

		taskHandler := handler.NewTaskHandler(cmdHandler, pool, logger)
		r.Route("/tasks", func(r chi.Router) {
			r.Post("/", taskHandler.Create)
			r.Get("/", taskHandler.List)
			r.Get("/{id}", taskHandler.Get)
			r.Patch("/{id}", taskHandler.Update)
			r.Delete("/{id}", taskHandler.Delete)
			r.Post("/{id}/restore", taskHandler.Restore)
			r.Post("/{id}/complete", taskHandler.Complete)
			r.Post("/{id}/uncomplete", taskHandler.Uncomplete)
			r.Post("/{id}/subtasks", taskHandler.CreateSubtask)
			r.Patch("/{id}/subtasks/{sid}", taskHandler.UpdateSubtaskTitle)
			r.Post("/{id}/subtasks/{sid}/complete", taskHandler.CompleteSubtask)
			r.Post("/{id}/subtasks/{sid}/uncomplete", taskHandler.UncompleteSubtask)
			r.Post("/{id}/labels", taskHandler.AddLabel)
			r.Delete("/{id}/labels/{lid}", taskHandler.RemoveLabel)
		})

		listHandler := handler.NewListHandler(cmdHandler, pool, logger)
		r.Route("/lists", func(r chi.Router) {
			r.Post("/", listHandler.Create)
			r.Get("/", listHandler.List)
			r.Delete("/{id}", listHandler.Delete)
		})

		labelHandler := handler.NewLabelHandler(cmdHandler, pool, logger)
		r.Route("/labels", func(r chi.Router) {
			r.Post("/", labelHandler.Create)
			r.Get("/", labelHandler.List)
			r.Delete("/{id}", labelHandler.Delete)
		})
	})

	return r
}

func healthHandler(pool *pgxpool.Pool, logger zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := pool.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			if encErr := json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy", "error": err.Error()}); encErr != nil {
				logger.Error().Err(encErr).Msg("failed to encode health response")
			}
			return
		}
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "healthy"}); err != nil {
			logger.Error().Err(err).Msg("failed to encode health response")
		}
	}
}

func requestLogger(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
			logger.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("remote", r.RemoteAddr).
				Msg("request started")

			next.ServeHTTP(ww, r)

			logger.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", ww.Status()).
				Msg("request completed")
		})
	}
}
