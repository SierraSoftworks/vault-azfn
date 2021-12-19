package agent

import (
	"io/ioutil"
	"log"
	"os"
	"text/template"
)

var templateFunctions = template.FuncMap{
	"env": func(key string) string {
		return os.Getenv(key)
	},
}

func ApplyTemplate(src, dst string) {
	content, err := ioutil.ReadFile(src)
	if err != nil {
		log.Fatal(err)
	}

	tmpl := template.Must(template.New("template").
		Funcs(templateFunctions).
		Parse(string(content)))

	f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	err = f.Truncate(0)
	if err != nil {
		log.Fatal(err)
	}

	err = tmpl.Execute(f, nil)
	if err != nil {
		log.Fatal(err)
	}
}
