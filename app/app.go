package app

import (
	"app/config"
	"app/db"
	"app/models"
	"app/worker"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/uptrace/bun/driver/pgdriver"
)

func Start(e *echo.Echo) error {
	sigCtx, sigCancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer sigCancel()

	// start workers
	workerCtx, stopWorkerCtx := context.WithCancel(context.Background())
	var workerWg sync.WaitGroup

	ln := pgdriver.NewListener(db.Bun)
	err := ln.Listen(workerCtx, "jobs:updated")
	if err != nil {
		slog.Error("failed to listen for jobs:updated", "error", err)
		stopWorkerCtx()
		return err
	}
	defer ln.Close()

	for range config.Config.WorkerCount {
		workerWg.Go(func() {
			worker.StartWorker(workerCtx, ln.Channel())
		})
	}

	err = pgdriver.Notify(workerCtx, db.Bun, "jobs:updated", "")
	if err != nil {
		slog.Error("failed to notify jobs:updated", "error", err)
		stopWorkerCtx()
		workerWg.Wait()
		return err
	}

	slog.Info("http://localhost:" + strconv.Itoa(config.Config.Port))

	// start server
	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- e.Start("0.0.0.0:" + strconv.Itoa(config.Config.Port))
	}()

	// wait for shutdown signal
	select {
	case <-sigCtx.Done():
		slog.Info("shutdown signal received")
	case err := <-serverErrCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error: " + err.Error())
		}
		slog.Info("server exited, initiating shutdown")
	}

	// stop accepting new jobs
	stopWorkerCtx()

	// wait for workers to finish running jobs
	slog.Info("waiting for workers to finish jobs...")
	workerWg.Wait()

	return nil
}

func RunBackgroundCleanUp(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				slog.Info("running background cleanup")
				_ = models.DeleteExpiredUsers(ctx)
			}
		}
	}()
}
