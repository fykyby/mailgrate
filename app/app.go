package app

import (
	"app/worker"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"
)

func Start(e *echo.Echo, workerCount int) error {
	sigCtx, sigCancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer sigCancel()

	// start workers
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	var workerWg sync.WaitGroup
	for range workerCount {
		workerWg.Go(func() {
			slog.Info("starting worker")
			worker.StartWorker(workerCtx)
		})
	}

	slog.Info("http://localhost:" + os.Getenv("PORT"))

	// start server
	go func() {
		sc := echo.StartConfig{
			Address:         "0.0.0.0:" + os.Getenv("PORT"),
			GracefulTimeout: 10 * time.Second,
		}
		if err := sc.Start(sigCtx, e); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error(err.Error())
			workerCancel()
		}
	}()

	// wait for shutdown signal
	<-sigCtx.Done()
	slog.Info("shutdown signal received")

	// stop and wait for workers
	workerCancel()
	workerWg.Wait()

	return nil
}
