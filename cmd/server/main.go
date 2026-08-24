package main

import (
	"concrete-curing-maturity-monitor/backend/internal/config"
	"concrete-curing-maturity-monitor/backend/internal/constants"
	"concrete-curing-maturity-monitor/backend/internal/handler"
	"concrete-curing-maturity-monitor/backend/internal/middleware"
	"concrete-curing-maturity-monitor/backend/internal/repository"
	"concrete-curing-maturity-monitor/backend/internal/router"
	"concrete-curing-maturity-monitor/backend/internal/service"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	settings, err := config.Load()
	if err != nil {
		logger.Error("configuration failed", "error", err)
		os.Exit(1)
	}
	database, err := config.OpenDatabase(settings)
	if err != nil {
		logger.Error("database initialization failed", "error", err)
		os.Exit(1)
	}
	sqlDatabase, err := database.DB()
	if err != nil {
		logger.Error("database pool initialization failed", "error", err)
		os.Exit(1)
	}
	defer sqlDatabase.Close()

	audit := middleware.NewAuditRecorder(database)
	authenticator := middleware.NewAuthenticator(database, settings.JWTSecret, settings.JWTExpiry)
	mixRepository := repository.NewMixDesignRepository(database)
	sectionRepository := repository.NewPourSectionRepository(database)
	seriesRepository := repository.NewTemperatureSeriesRepository(database)
	forecastRepository := repository.NewStrengthForecastRepository(database)

	mixHandler := handler.NewMixDesignHandler(service.NewMixDesignService(mixRepository, audit))
	sectionHandler := handler.NewPourSectionHandler(service.NewPourSectionService(sectionRepository, mixRepository, audit))
	seriesHandler := handler.NewTemperatureSeriesHandler(service.NewTemperatureSeriesService(seriesRepository, sectionRepository, audit, settings.MaxMissingRatio))
	forecastHandler := handler.NewStrengthForecastHandler(service.NewStrengthForecastService(forecastRepository, sectionRepository, seriesRepository, mixRepository, audit))
	authHandler := handler.NewAuthHandler(authenticator)

	engine := gin.New()
	engine.Use(middleware.RequestID(), middleware.Recovery(logger), middleware.ErrorHandler(logger), middleware.AuditContext(audit))
	engine.GET("/healthz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := sqlDatabase.PingContext(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "database": "unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "concrete-curing-maturity-monitor"})
	})

	loginLimiter := middleware.NewRateLimiter(settings.LoginLimitPerMinute, time.Minute)
	apiV1 := engine.Group("/api/v1")
	apiV1.POST("/auth/login", loginLimiter.Middleware("login"), authHandler.Login)
	protected := apiV1.Group("")
	protected.Use(authenticator.Middleware())
	router.RegisterPourSectionRoutes(protected, sectionHandler)
	router.RegisterMixDesignRoutes(protected, mixHandler)
	router.RegisterTemperatureSeriesRoutes(protected, seriesHandler, middleware.NewRateLimiter(settings.ImportLimitPerMinute, time.Minute))
	router.RegisterStrengthForecastRoutes(protected, forecastHandler, middleware.NewRateLimiter(settings.ForecastLimitPerMinute, time.Minute))
	protected.GET("/audit-logs", middleware.RequirePermission(constants.PermissionAuditRead), audit.ListHandler)

	server := &http.Server{
		Addr:              ":" + settings.Port,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	errChannel := make(chan error, 1)
	go func() {
		logger.Info("server listening", "port", settings.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChannel <- err
		}
	}()

	signalContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case <-signalContext.Done():
		logger.Info("shutdown requested")
	case err := <-errChannel:
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
	shutdownContext, cancel := context.WithTimeout(context.Background(), settings.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("server stopped")
}
