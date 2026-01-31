package app

import (
	"app/config"
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

	"github.com/labstack/echo/v5"
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
	for range config.Config.WorkerCount {
		workerWg.Go(func() {
			worker.StartWorker(workerCtx)
		})
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
