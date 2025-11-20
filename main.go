package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/sierrasoftworks/vault-azfn/agent"
	"go.opentelemetry.io/otel"
)

var version = "0.0.0-dev"

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: vault-launcher <app> [args...]")
	}

	ctx := context.Background()
	tp := agent.NewTraceProvider(ctx, version)
	defer func() { _ = tp.Shutdown(ctx) }()

	otel.SetTracerProvider(tp)

	tmp, err := os.MkdirTemp(os.TempDir(), "vault")
	if err != nil {
		log.Fatal("Failed to create temporary directory: ", err)
	}
	defer os.RemoveAll(tmp)

	for i, arg := range os.Args[1:] {
		if strings.HasSuffix(arg, ".tpl") {
			f, err := os.Stat(arg)
			if err != nil {
				log.Println("Failed to render template file ", arg, ": ", err)
				continue
			}

			if f.IsDir() {
				log.Println("Failed to render template directory ", arg)
				continue
			}

			target := filepath.Join(tmp, arg[:len(arg)-len(".tpl")])
			if err = os.MkdirAll(filepath.Base(target), 0755); err != nil {
				log.Fatal("Failed to create template target directory: ", err)
				continue
			}

			agent.ApplyTemplate(arg, target)
			os.Args[i+1] = target
		}
	}

	if os.Getenv("VAULT_AGENT_SET_EXECUTABLE_PATTERN") != "" {
		agent.SetExecutablePattern(ctx, os.Getenv("VAULT_AGENT_SET_EXECUTABLE_PATTERN"))
	}

	binary := os.Args[1]
	args := os.Args[2:]

	if err = agent.RunApp(ctx, binary, args); err != nil {
		log.Fatal(err)
	}
}
