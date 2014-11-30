package main

import (
	"os"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	nlogrus "github.com/meatballhat/negroni-logrus"
)

var Log = nlogrus.NewMiddleware()

func main() {
	port := ":" + os.Getenv("PORT")
	if port == ":" {
		port = ":3000"
	}

	router := mux.NewRouter().StrictSlash(true)

	for _, route := range routes {
		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(route.HandlerFunc)
	}

	n := negroni.Classic()
	n.Use(Log)
	n.UseHandler(router)
	n.Run(port)
}
