package main

import (
	"os"

	"github.com/tlaceby/hot-reload/src/handlers"
)

func main() {
	args := os.Args[1:]

	var commandHandlers = map[string]func([]string){
		"init":  handlers.InitHandler,
		"help":  handlers.HelpHandler,
		"watch": handlers.WatchHandler,
	}

	if len(args) == 0 {
		handlers.HelpHandler(args)
		os.Exit(1)
	}

	for command, handler := range commandHandlers {
		if args[0] == command {
			handler(args)
		}
	}
}
