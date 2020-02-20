package main

import "github.com/tennuem/tbot/internal/server"

func main() {
	svr := server.NewServer()
	svr.Run()
}
