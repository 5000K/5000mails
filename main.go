package main

import (
	"context"
	"crypto/ed25519"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/5000K/5000mails/api"
	"github.com/5000K/5000mails/cli"
	"github.com/5000K/5000mails/config"
	"github.com/5000K/5000mails/db"
	"github.com/5000K/5000mails/renderer"
	"github.com/5000K/5000mails/service"
	"github.com/5000K/5000mails/smtp"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	cfg, err := config.Get()
	if err != nil {
		logger.Error("loading config", slog.Any("error", err))
		os.Exit(1)
	}

	database, err := db.Connect(cfg.DB.Type, cfg.DB.DSN)
	if err != nil {
		logger.Error("connecting to database", slog.Any("error", err))
		os.Exit(1)
	}
	if err := db.AutoMigrate(database); err != nil {
		logger.Error("migrating database", slog.Any("error", err))
		os.Exit(1)
	}

	repo := db.NewMailingListRepository(database, logger)

	sender, err := smtp.NewSender(cfg.Smtp, logger)
	if err != nil {
		logger.Error("creating smtp sender", slog.Any("error", err))
		os.Exit(1)
	}

	tmplBytes, err := config.FetchResource(cfg.Paths.Template)
	if err != nil {
		logger.Error("loading template", slog.String("path", cfg.Paths.Template), slog.Any("error", err))
		os.Exit(1)
	}
	rndr, err := renderer.NewGoldmarkRenderer(tmplBytes, logger)
	if err != nil {
		logger.Error("creating renderer", slog.Any("error", err))
		os.Exit(1)
	}

	confirmRaw, err := config.FetchResource(cfg.Paths.ConfirmMail)
	if err != nil {
		logger.Error("loading confirm mail template", slog.String("path", cfg.Paths.ConfirmMail), slog.Any("error", err))
		os.Exit(1)
	}

	subscriptionSvc := service.NewSubscriptionService(repo, repo, repo, rndr, sender, string(confirmRaw), cfg.BaseURL)
	listSvc := service.NewListService(repo, repo)
	mailSvc := service.NewMailService(repo, repo, repo, rndr, sender, cfg.BaseURL)
	schedulingSvc := service.NewSchedulingService(repo, mailSvc, 30*time.Second, logger)
	schedulingSvc.Start()

	publicHandler := api.NewPublicHandler(subscriptionSvc, mailSvc, api.RedirectPages{
		SubscribeSuccess:   cfg.Redirects.SubscribeSuccess,
		SubscribeError:     cfg.Redirects.SubscribeError,
		ConfirmSuccess:     cfg.Redirects.ConfirmSuccess,
		ConfirmError:       cfg.Redirects.ConfirmError,
		UnsubscribeSuccess: cfg.Redirects.UnsubscribeSuccess,
		UnsubscribeError:   cfg.Redirects.UnsubscribeError,
	}, logger)

	var publicKey ed25519.PublicKey
	if cfg.Auth.PublicKeyPath != "" {
		publicKey, err = cli.ReadPublicKey(cfg.Auth.PublicKeyPath)
		if err != nil {
			logger.Error("loading auth public key", slog.String("path", cfg.Auth.PublicKeyPath), slog.Any("error", err))
			os.Exit(1)
		}
		logger.Info("private API authentication enabled")
	} else {
		logger.Warn("private API authentication disabled - no public key configured")
	}

	privateHandler := api.NewPrivateHandler(listSvc, mailSvc, mailSvc, schedulingSvc, publicKey, logger)

	publicServer := &http.Server{Addr: cfg.PublicAddr, Handler: publicHandler.Routes()}
	privateServer := &http.Server{Addr: cfg.PrivateAddr, Handler: privateHandler.Routes()}

	go func() {
		logger.Info("public API listening", slog.String("addr", cfg.PublicAddr))
		if err := publicServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("public server error", slog.Any("error", err))
		}
	}()

	go func() {
		logger.Info("private API listening", slog.String("addr", cfg.PrivateAddr))
		if err := privateServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("private server error", slog.Any("error", err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.Info("shutting down", slog.String("signal", sig.String()))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := publicServer.Shutdown(ctx); err != nil {
		logger.Error("public server shutdown error", slog.Any("error", err))
	}
	if err := privateServer.Shutdown(ctx); err != nil {
		logger.Error("private server shutdown error", slog.Any("error", err))
	}

	logger.Info("servers stopped")
	schedulingSvc.Stop()
}
