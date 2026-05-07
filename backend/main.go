package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"loghub/config"
	"loghub/handler"
	"loghub/middleware"
	"loghub/service"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	configPath := flag.String("config", "./config.yaml", "Path to configuration file")
	seedData := flag.Bool("seed", false, "Seed demo log data on startup")
	flag.Parse()

	// Initialize structured logger
	logCfg := zap.Config{
		Level:       zap.NewAtomicLevelAt(zapcore.InfoLevel),
		Development: false,
		Encoding:    "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := logCfg.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Fatal("failed to load configuration",
			zap.String("path", *configPath),
			zap.Error(err),
		)
	}
	logger.Info("configuration loaded successfully",
		zap.Int("app_count", len(cfg.Apps)),
		zap.Int("http_port", cfg.Server.HTTPPort),
		zap.Int("wss_port", cfg.Server.WSSPort),
		zap.Int("max_retain_days", cfg.Log.MaxRetainDays),
	)

	// Initialize log store
	logStore, err := service.NewLogStore(cfg.Log.BaseDir, logger)
	if err != nil {
		logger.Fatal("failed to initialize log store", zap.Error(err))
	}
	logger.Info("log store initialized", zap.String("base_dir", cfg.Log.BaseDir))

	// Seed demo data if requested
	if *seedData {
		service.SeedDemoData(logStore, logger)
	}

	// Initialize WebSocket hub
	wsHub := handler.NewWSHub(logStore, logger)
	go wsHub.Run()
	logger.Info("websocket hub started")

	// Initialize cleanup service
	cleanupSvc := service.NewCleanupService(cfg.Log.BaseDir, logger)
	go cleanupSvc.Start()

	// Initialize API handler
	apiHandler := handler.NewAPIHandler(logStore, wsHub, logger)

	// Setup HTTP routes
	mux := http.NewServeMux()

	// Health check endpoint (no auth)
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":200,"message":"ok"}`))
	})

	// Public routes
	mux.HandleFunc("/api/login", apiHandler.HandleLogin)

	// Protected routes - wrap with auth middleware
	authMw := middleware.Auth(logger)

	mux.Handle("/api/dashboard", authMw(http.HandlerFunc(apiHandler.HandleGetDashboard)))
	mux.Handle("/api/apps", authMw(http.HandlerFunc(apiHandler.HandleGetApps)))
	mux.Handle("/api/logs/files", authMw(http.HandlerFunc(apiHandler.HandleGetLogFiles)))
	mux.Handle("/api/logs/query", authMw(http.HandlerFunc(apiHandler.HandleQueryLogs)))
	mux.Handle("/api/config", authMw(http.HandlerFunc(apiHandler.HandleGetConfig)))

	// WebSocket routes (producer uses WSS, viewer uses WS through HTTP)
	mux.HandleFunc("/ws/producer", wsHub.HandleProducer)
	mux.HandleFunc("/ws/viewer", wsHub.HandleViewer)

	// Apply global middleware
	var httpHandler http.Handler = mux
	httpHandler = middleware.CORS()(httpHandler)
	httpHandler = middleware.Logger(logger)(httpHandler)
	httpHandler = middleware.Recovery(logger)(httpHandler)

	// Start HTTP server
	httpAddr := fmt.Sprintf(":%d", cfg.Server.HTTPPort)
	httpServer := &http.Server{
		Addr:    httpAddr,
		Handler: httpHandler,
	}

	go func() {
		logger.Info("HTTP server starting", zap.String("addr", httpAddr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	// Start WSS server if certificates exist
	go func() {
		certFile := cfg.Server.CertFile
		keyFile := cfg.Server.KeyFile

		if _, err := os.Stat(certFile); os.IsNotExist(err) {
			logger.Warn("TLS certificate not found, WSS server not started. Run certgen to generate certificates.",
				zap.String("cert_file", certFile),
			)
			return
		}
		if _, err := os.Stat(keyFile); os.IsNotExist(err) {
			logger.Warn("TLS key not found, WSS server not started",
				zap.String("key_file", keyFile),
			)
			return
		}

		wssMux := http.NewServeMux()
		wssMux.HandleFunc("/ws/producer", wsHub.HandleProducer)

		var wssHandler http.Handler = wssMux
		wssHandler = middleware.CORS()(wssHandler)
		wssHandler = middleware.Logger(logger)(wssHandler)
		wssHandler = middleware.Recovery(logger)(wssHandler)

		wssAddr := fmt.Sprintf(":%d", cfg.Server.WSSPort)
		tlsCfg := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
		wssServer := &http.Server{
			Addr:      wssAddr,
			Handler:   wssHandler,
			TLSConfig: tlsCfg,
			// Disable HTTP/2 — WebSocket upgrades require HTTP/1.1
			TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
		}

		logger.Info("WSS server starting", zap.String("addr", wssAddr))
		if err := wssServer.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
			logger.Error("WSS server failed", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down servers...")
	cleanupSvc.Stop()
	logger.Info("server stopped gracefully")
}
