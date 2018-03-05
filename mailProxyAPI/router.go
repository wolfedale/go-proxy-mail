package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

func NewRouter() *mux.Router {
	/* When true, if the route path is "/path/", accessing "/path"
	   will redirect to the former and vice versa. In other words,
	   your application will always see the path as specified in the route.
	   When false, if the route path is "/path", accessing "/path/"
	   will not match this route and vice versa.
	*/
	router := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {
		// logs && logger
		var handler http.Handler
		handler = route.HandlerFunc
		handler = Logger(handler, route.Name)

		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}

	return router
}
