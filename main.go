package main

import (
	"os"

	"github.com/codegangsta/negroni"
	"github.com/julienschmidt/httprouter"
	nlogrus "github.com/meatballhat/negroni-logrus"
)

var (
	Log    = nlogrus.NewMiddleware()
	logger = Log.Logger
)

func main() {
	port := ":" + os.Getenv("PORT")
	if port == ":" {
		port = ":3000"
	}

	router := httprouter.New()

	for _, route := range routes {
		router.Handle(route.Method, route.Pattern, route.HandlerFunc)
	}

	n := negroni.Classic()
	n.Use(Log)
	n.UseHandler(router)
	n.Run(port)
}
