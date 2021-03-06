package main

import "github.com/julienschmidt/httprouter"

type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc httprouter.Handle
}

var routes = []Route{
	Route{
		Name:        "portfolio_index",
		Method:      "GET",
		Pattern:     "/",
		HandlerFunc: PortfolioIndex,
	},
	Route{
		Name:        "id3_fix",
		Method:      "GET",
		Pattern:     "/id3/fix",
		HandlerFunc: Id3FixSong,
	},
	Route{
		Name:        "xval_index",
		Method:      "GET",
		Pattern:     "/xval",
		HandlerFunc: XvalIndex,
	},
	//	Route{
	//		Name:        "xval_api",
	//		Method:      "GET",
	//		Pattern:     "/xval.json",
	//		HandlerFunc: XvalApi,
	//	},
}
