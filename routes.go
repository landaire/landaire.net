package main

import "net/http"

type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

var routes = []Route{
	Route{
		Name:        "portfolio_index",
		Method:      "GET",
		Pattern:     "/",
		HandlerFunc: PortfolioIndex,
	},
}
