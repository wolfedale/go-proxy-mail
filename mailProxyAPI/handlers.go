package main

import (
	"encoding/json"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

/*
  API index, using for dashboard, we are using here
  template and giving access to mails struct
*/
func Index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	t, _ := template.ParseFiles("templates/index.html")
	t.Execute(w, mails)
}

/*
  API mailindex is simply listening all blocked e-mails
  and returning list in json format
*/
func MailIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(mails); err != nil {
		panic(err)
	}
}

/*
  mailshow will show us details about one specific e-mail
  if there is no e-mail we are returing 404
*/
func MailShow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var mailId int
	var err error
	if mailId, err = strconv.Atoi(vars["mailId"]); err != nil {
		panic(err)
	}
	mail := RepoFindMail(mailId)
	if mail.Id > 0 {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(mail); err != nil {
			panic(err)
		}
		return
	}

	// If we didn't find it, 404
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusNotFound)
	if err := json.NewEncoder(w).Encode(jsonErr{Code: http.StatusNotFound, Text: "Not Found"}); err != nil {
		panic(err)
	}

}

/*
  MailCreate is a POST request which can add e-mail to blocked list
  to test it we can call it like:
  curl -H "Content-Type: application/json" -d '{"name":"New Mail"}' http://localhost:8080/mails
*/
func MailCreate(w http.ResponseWriter, r *http.Request) {
	var mail Mail
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}
	if err := json.Unmarshal(body, &mail); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(422)
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	}

	t := RepoCreateMail(mail)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(t); err != nil {
		panic(err)
	}
}

/*
  MailDelete is a DELETE request which will delete e-mail from
  the blocked list. If there is nothing to delete it will return
  404 HTTP code.
  To test it:
  curl -i -X DELETE http://localhost:8080/mails/1
*/
func MailDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var mailId int
	var err error
	if mailId, err = strconv.Atoi(vars["mailId"]); err != nil {
		panic(err)
	}
	mail := RepoFindMail(mailId)
	if mail.Id > 0 {

		// Send e-mail
		Cmd(mail.Id)

		// Delete it from the API/Dashboard
		t := RepoDestroyMail(mail.Id)
		if t != nil {
			panic(err)
		}

		// Return correct status
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(jsonErr{Code: http.StatusOK, Text: "OK"}); err != nil {
			panic(err)
		}
	} else {
		// If we didn't find it, 404
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(jsonErr{Code: http.StatusNotFound, Text: "Not Found"}); err != nil {
			panic(err)
		}
	}
}
