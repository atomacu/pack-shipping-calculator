package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"pack-shipping-calculator/backend/internal/app"
)

var (
	runApp      = app.Run
	exitProcess = os.Exit
	logError    = log.Print
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := runApp(ctx); err != nil {
		logError(err)
		exitProcess(1)
	}
}
