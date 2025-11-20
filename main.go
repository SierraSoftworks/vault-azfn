package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/sierrasoftworks/vault-azfn/agent"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
)

var version = "0.0.0-dev"

func main() {
	if len(os.Args) < 2 {
		logrus.Fatal("Usage: vault-launcher <app> [args...]")
	}

	ctx := context.Background()
	tp, lp := agent.NewTraceProvider(ctx, version)
	defer func() { _ = tp.Shutdown(ctx) }()
	defer func() { _ = lp.Shutdown(ctx) }()

	otel.SetTracerProvider(tp)
	global.SetLoggerProvider(lp)

	tmp, err := os.MkdirTemp(os.TempDir(), "vault")
	if err != nil {
		logrus.Fatal("Failed to create temporary directory: ", err)
	}
	defer os.RemoveAll(tmp)

	for i, arg := range os.Args[1:] {
		if strings.HasSuffix(arg, ".tpl") {
			f, err := os.Stat(arg)
			if err != nil {
				logrus.Error("Failed to render template file ", arg, ": ", err)
				continue
			}

			if f.IsDir() {
				logrus.Error("Failed to render template directory ", arg)
				continue
			}

			target := filepath.Join(tmp, arg[:len(arg)-len(".tpl")])
			if err = os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				logrus.Fatal("Failed to create template target directory: ", err)
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
		logrus.Fatal(err)
	}
}
