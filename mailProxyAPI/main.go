/*

 Pawel Grzesik, 2017
 pawel.grzesik

 It's a really easy and fast implementation of API
 for mailProxy and DLP purposes.

 We can:
 - GET / to check the dashboard
 - GET /mails to list all blocked emails
 - GET /mails/{mailId} to get some details about one email
 - POST /mails to add email to blocked list
 - DELETE /mails/{mailId} to delete e-mail from blocked list
 - GET /mails/delete/{mailId} to delete an e-mail using dashboard

Dashboard/API is listening on port 8080 - in default
*/

package main

import (
	"log"
	"net/http"
)

/*
  constant for the path where we are going to keep
  our blocked e-mails
*/
const QUEUEDIR string = "/var/spool/mailProxy/queue/"

/*
  here we are starting our API
*/
func main() {
	// creating new router using mux
	router := NewRouter()

	// giving access to CSS, JS and QUEUE dir
	cssHandler := http.FileServer(http.Dir("./templates/css/"))
	http.Handle("/css/", http.StripPrefix("/css/", cssHandler))

	jsHandler := http.FileServer(http.Dir("./templates/js/"))
	http.Handle("/js/", http.StripPrefix("/js/", jsHandler))

	qHandler := http.FileServer(http.Dir(QUEUEDIR))
	http.Handle("/queue/", http.StripPrefix("/queue/", qHandler))

	http.Handle("/", router)

	//Fatal is equivalent to Print() followed by a call to os.Exit(1).
	//log.Fatal(http.ListenAndServe(":8080", router))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
