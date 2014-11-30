package main

import (
	"io/ioutil"
	"log"
	"net/http"

	"github.com/flosch/pongo2"
	_ "github.com/flosch/pongo2-addons"
)

func PortfolioIndex(w http.ResponseWriter, r *http.Request) {
	log.Printf("Hit portfolio")

	content, err := ioutil.ReadFile("./views/portfolio.md")

	if err != nil {
		return
		// Do something here
	}

	// Render the template
	template := pongo2.Must(pongo2.FromFile("./views/portfolio.tpl"))
	template.ExecuteWriter(pongo2.Context{"body_content": string(content)}, w)
}
