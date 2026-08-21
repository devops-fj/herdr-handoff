package main

import (
	"fmt"
	"os"

	"github.com/devops-fj/herdr-handoff/internal/app"
)

var version = "dev"

func main() {
	application := app.New()
	application.Version = version
	if err := application.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "herdr-handoff:", err)
		os.Exit(1)
	}
}
