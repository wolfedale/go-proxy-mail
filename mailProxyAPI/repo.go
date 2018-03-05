package main

import "fmt"
import "time"

var currentId int

var mails Mails

/*
  Find e-mail in our struct
*/
func RepoFindMail(id int) Mail {
	for _, t := range mails {
		if t.Id == id {
			return t
		}
	}
	// return empty Mail if not found
	return Mail{}
}

/*
  Add e-mail to blocked list
*/
func RepoCreateMail(t Mail) Mail {
	currentId += 1
	t.Id = currentId

	// generate time.now and add it to the struct
	// every time when we are calling API
	timenow := time.Now()
	t.Date = timenow.String()

	mails = append(mails, t)
	return t
}

/*
  Delete e-mail from blocked list
*/
func RepoDestroyMail(id int) error {
	for i, t := range mails {
		if t.Id == id {
			mails = append(mails[:i], mails[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("Could not find Mail with id of %d to delete", id)
}
