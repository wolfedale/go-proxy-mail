package main

import (
	"net/http"
)

type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

type Routes []Route

// all our routes
var routes = Routes{
	Route{
		"Index",
		"GET",
		"/",
		Index,
	},
	Route{
		"MailIndex",
		"GET",
		"/mails",
		MailIndex,
	},
	Route{
		"MailShow",
		"GET",
		"/mails/{mailId}",
		MailShow,
	},
	Route{
		"MailCreate",
		"POST",
		"/mails",
		MailCreate,
	},
	Route{
		"MailDelete",
		"DELETE",
		"/mails/{mailId}",
		MailDelete,
	},
	Route{
		"MailDelete",
		"GET",
		"/mails/delete/{mailId}",
		MailDelete,
	},
}
