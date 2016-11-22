/**
#
# To run as a test:
# > go run script
#
# To run on the production we need to compile it:
# > go build script
# > ./script
#
**/

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/mail"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

var queue string

const OURDOMAIN string = "mydomain"
const PROXYDIR string = "/tmp/proxy"
const PROXYARCHIVE string = "/queue"
const PROXYLOG string = "/logs/proxy.log"

func main() {
	/*
	  Needed for random queue string
	*/
	rand.Seed(time.Now().UnixNano())

	/*
	  Generate random string with 10 characters.
	*/
	queue = RandStringBytesRmndr(10)

	/*
	  Setup correct PATHs for the app
	  PROXYDIR is where the app is going to work
	  PROXYARCHIVE is where we are keeping blocked e-mails
	  PROXYLOG is location for our log file
	*/
	archiveFile := path.Join(PROXYDIR, PROXYARCHIVE, queue)
	logFile := path.Join(PROXYDIR, PROXYLOG)

	/*
	  Open and deal with the log file
	*/
	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)

	/*
	   Call stats API
	*/
	statsAPI("hit")

	/*
	  Check who is the sender from the mail Headers
	*/
	sender, err := MailSender()
	if err != nil {
		log.Println(queue+" no sender ", err)
		os.Exit(0)
	}

	/*
	  Check recipients and create string from them
	*/
	recipients, err := MailRecipients()
	log.Println(recipients)
	if err != nil {
		log.Println(queue+" no recipients ", err)
		os.Exit(0)
	}

	/*
	  Read mail source (raw) from the STDIN
	*/
	maildata, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
		os.Exit(0)
	}

	/*
	  Check who is the sender from the mail Body
	  If we cannot verify it, e-mail will be send
	  and save for the later investigation
	*/
	senderHeader, err := MailSenderHeader(maildata)
	if err != nil {
		log.Println(queue+"senderHeader() err: ", err)
		doMail := sendMail(sender, recipients, maildata)
		if doMail != nil {
			log.Println(queue+" sendMail() ", doMail)
		}
		aMail := saveMail(archiveFile, maildata)
		if aMail != nil {
			log.Println(queue+" sendMail() error: ", aMail)
		}
	}

	/*
	  Checking if sender is from OURDOMAIN.
	  Block if it is.
	  Pass if it's not.
	*/
	if CheckDomain(senderHeader) == true {
		statsAPI("block")
		log.Println(queue + " BLOCKED: " + sender + " => " + recipients)
		n_from := "proxy@mydomain"
		n_to := "alert@"
		n_body := []byte(queue + " BLOCKED: " + sender + " => " + recipients)

		// Send Mail
		sMail := sendMail(n_from, n_to, n_body)
		if sMail != nil {
			log.Println(queue+" sendMail() error: ", sMail)
		}

		// Archive Mail
		log.Println(queue + " saved to: " + archiveFile)
		aMail := saveMail(archiveFile, maildata)
		if aMail != nil {
			log.Println(queue+" sendMail() error: ", aMail)
		}

		// Exit with 0
		os.Exit(0)
	} else {
		statsAPI("pass")
		log.Println(queue + " PASSED: " + sender + " => " + recipients)
		doMail := sendMail(sender, recipients, maildata)
		if doMail != nil {
			log.Println(queue+" sendMail() ", doMail)
		}
	}
}

/*
  return MAIL_FROM (checking mail headers)
*/
func MailSender() (string, error) {
	sender := os.Args[1]
	return sender, nil
}

/*
  return MAIL_RECIPIENTS as a one string
*/
func MailRecipients() (string, error) {
	var templist []string
	for _, rec := range os.Args[2:] {
		templist = append(templist, rec)
	}
	recipients := strings.Join(templist, " ")
	return recipients, nil
}

/*
  return MAIL_FROM (checking mail body)
*/
func MailSenderHeader(maildata []byte) (string, error) {
	mailString := stdinToString(maildata)
	header := mailHeader(readMail(mailString))
	sender := header.Get("From")
	return sender, nil
}

/*
  Generate random string for a mail queue
*/
func RandStringBytesRmndr(n int) string {
	const letterBytes = "123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

/*
  Checking if OURDOMAIN is the same as MAIL_FROM
*/
func CheckDomain(sender string) bool {
	var rbool bool
	host := strings.Split(sender, "@")[1]

	if host == OURDOMAIN {
		log.Println(queue + " checking " + host + " == " + OURDOMAIN)
		rbool = true
	} else {
		log.Println(queue + " checking " + host + " != " + OURDOMAIN)
		rbool = false
	}
	return rbool
}

/*
  Function will send an e-mail, we need to call it with three
  arguments:
  from - mail from
  recipients - mail recipients
  maildata - mail source
*/
func sendMail(from, recipients string, maildata []byte) error {
	sendmail := exec.Command("/usr/sbin/sendmail", "-G", "-i", "-f", from, "--", recipients)
	pipe, err := sendmail.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}

	err = sendmail.Start()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprintf(pipe, "%s", maildata)
	pipe.Close()

	return err
}

/*
  Save e-mail for later investigation
*/
func saveMail(archiveFile string, maildata []byte) error {
	err := ioutil.WriteFile(archiveFile, maildata, 0644)
	return err
}

/*
  Convert mail raw to string
*/
func stdinToString(maildata []byte) *strings.Reader {
	r := strings.NewReader(string(maildata))
	return r
}

/*
  Return header from the Mail.
*/
func mailHeader(m *mail.Message) mail.Header {
	header := m.Header
	return header
}

/*
  Read mail string and return *mail.Message
*/
func readMail(r *strings.Reader) *mail.Message {
	m, err := mail.ReadMessage(r)
	if err != nil {
		log.Fatal(err)
	}
	return m
}

/*
  Stats API needed for Sensu/Monitoring
*/
func statsAPI(value string) {
	url := "http://localhost:8000/" + value
	resp, err := http.Get(url)
	if err != nil {
		log.Println("Cannot connect to the API: " + value)
	} else {
		defer resp.Body.Close()
	}
}
