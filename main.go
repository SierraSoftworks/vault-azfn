package main

import (
	"log"
	"os"
	"strings"

	"github.com/sierrasoftworks/vault-azfn/agent"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: vault-launcher <app> [args...]")
	}

	insights := agent.GetInsights()
	defer insights.Channel().Close()

	for i, arg := range os.Args[1:] {
		if strings.HasSuffix(arg, ".tpl") {
			f, err := os.Stat(arg)
			if err != nil {
				log.Println("Failed to render template file ", arg, ": ", err)
				continue
			}

			if f.IsDir() {
				log.Println("Failed to render template directory ", arg)
			}

			agent.ApplyTemplate(arg, arg[:len(arg)-len(".tpl")])
			os.Args[i+1] = arg[:len(arg)-len(".tpl")]
		}
	}

	binary := os.Args[1]
	args := os.Args[2:]

	err := agent.RunApp(insights, binary, args)
	if err != nil {
		log.Fatal(err)
	}
}
