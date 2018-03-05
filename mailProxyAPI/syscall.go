package main

import (
	"fmt"
	"io/ioutil"
	"os/exec"
)

/*
  here we are taking mail.id
  looking for it in our struct
  and then accessing Sender and Recipient
  and calling sendMail to send an e-mail
*/
func Cmd(mailId int) {
	var from string
	var recipients string
	var queue string

	for _, t := range mails {
		if t.Id == mailId {
			from = t.Sender
			recipients = t.Recipient
			queue = QUEUEDIR + t.Queue
		}
	}
	dat, err := ioutil.ReadFile(queue)
	check(err)
	mail_err := sendMail(from, recipients, dat)
	if mail_err != nil {
		panic(err)
	}
}

/*
  Sending/Passing an e-mail
*/
func sendMail(from, recipients string, maildata []byte) error {
	sendmail := exec.Command("/usr/sbin/sendmail", "-G", "-i", "-f", from, "--", recipients)
	pipe, _ := sendmail.StdinPipe()
	sendmail.Start()
	fmt.Fprintf(pipe, "%s", maildata)
	pipe.Close()
	return nil
}

/*
  Check for panic
*/
func check(e error) {
	if e != nil {
		panic(e)
	}
}
