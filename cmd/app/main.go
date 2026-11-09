package main

import (
	"context"
	"fmt"
	"os"

	"github.com/tennuem/tbot/internal/historycmd"
	"github.com/tennuem/tbot/internal/server"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if len(os.Args) > 1 && os.Args[1] == "history" {
		if err := historycmd.Run(ctx, os.Args[2:], os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "history: %s\n", err)
			os.Exit(1)
		}
		return
	}
	svr := server.NewServer()
	if err := svr.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to run server: %s\n", err)
		os.Exit(1)
	}
}
