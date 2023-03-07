package main

import (
	"context"
	"fmt"
	"os"

	"github.com/tennuem/tbot/internal/server"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svr := server.NewServer()
	if err := svr.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to run server: %s", err)
		os.Exit(1)
	}
}
