/**
#
# Pawel Grzesik, 2016
#
# To run as a test:
# > go run script
#
# To run on the production we need to compile it:
# > go build script
# > ./script
#
**/

/*
  Main Package
*/
package main

/*
  Here we are importing few modules that
  we need to use.
*/
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

/*
  ARRAY
  OURDOMAIN: domain list that we want to check
*/
var OURDOMAIN = [2]string{"foobar.org",
	"foobar.com"}

/*
  SLICE
  WHITELIST: we can whitelist e-mails by name
*/
var WHITELIST = [...]string{"alerter.test",
	"system.email",
	"jira",
	"alerter.live",
	"salesforce.com",
	"sf.com"}

var CheckUserList = [...]string{"pawel.grzesik"}

/*
  PROXYDIR: main directory for the proxyMail tool
  PROXYARCHIVE: directory for the queue (blocked mails)
  PROXYLOG: directory/path for the log file
*/
const PROXYDIR string = "/var/spool/mailProxy/"
const PROXYARCHIVE string = "/queue"
const PROXYLOG string = "/logs/proxy.log"

/*
  E-mail Notification Settings
*/
const NotificationFrom string = "mailProxy@"
const NotificationRecipients string = "pawel.grzesik@"
const NotificationSubject string = "mailProxy problem!"

/*
  Our main MailStruct struct type
  MailQueue: queue_num
  BackupFile: path to the blocked mail
  LogFile: log file
  MailData: raw mail source
*/
type MailStruct struct {
	MailQueue  string
	BackupFile string
	LogFile    string
	MailData   []byte
}

type APIStruct struct {
	Sender       string `json:"sender"`
	SenderHeader string `json:"senderheader"`
	Recipient    string `json:"recipient"`
	Queue        string `json:"queue`
	Blocked      bool   `json:"blocked"`
}

/*
  DEBUG MODE
  true or false
*/
const DEBUG bool = false

func main() {
	/*
	  Needed for random queue string every time when we execute the code
	*/
	rand.Seed(time.Now().UnixNano())

	/*
	  Generate random string with 10 characters.
	*/
	queue, err := RandStringBytesRmndr(10)
	if err != nil {
		log.Println("Problem with RandStringBytesRmndr: ", err)
		os.Exit(0)
	}

	/*
	  Setup correct PATHs for the app
	*/
	archiveFile := path.Join(PROXYDIR, PROXYARCHIVE, queue)
	logFile := path.Join(PROXYDIR, PROXYLOG)

	/*
	   Initialization
	   This allocates memory for all the fields, sets each of them
	   to their zero value and returns a pointer.
	*/
	s := &MailStruct{
		MailQueue:  queue,
		BackupFile: archiveFile,
		LogFile:    logFile,
	}

	/*
	  Open and deal with the log file
	*/
	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Println(s.MailQueue+" Problem with opening os.OpenFile(): ", err)
		s.saveMail()
		s.sendNotification("Problem with opening os.OpenFile()")
		os.Exit(0)
	}
	defer f.Close()

	log.SetOutput(f)

	/*
	   Read mail source (raw) from the STDIN
	*/
	struct_err := s.readData()
	if struct_err != nil {
		log.Println(s.MailQueue+" Problem with maildata: ", struct_err)
		s.sendNotification("Problem with maildata")
	}

	/*
	   From now on we need to backup mail raw source in case of any problems.
	*/

	/*
	  Check who is the sender from the mail Headers
	*/
	sender, err := MailSender()
	log.Println(s.MailQueue + " Sender from args postfix: " + sender)
	if err != nil {
		log.Println(s.MailQueue+" no sender ", err)
		s.saveMail()
		s.sendNotification("no sender")
		os.Exit(0)
	}

	/*
	  Check recipients and create string from them
	*/
	recipients, err := MailRecipients()
	log.Println(s.MailQueue + " Recipients from args postfix: " + recipients)
	if err != nil {
		log.Println(s.MailQueue+" no recipients ", err)
		s.saveMail()
		s.sendNotification("no recipients")
		os.Exit(0)
	}

	/*
	  Check who is the sender from the mail Body
	  If we cannot verify it, e-mail will be send
	  and save for the later investigation
	*/

	// Convert raw mail to string
	mailString, err := stdinToString(s.MailData)
	if err != nil {
		log.Println(s.MailQueue+" Cannot convert raw mail to string ", err)
		s.saveMail()
		s.sendNotification("Cannot convert raw mail to string")
		os.Exit(0)
	}

	// Read mail and return *mail.Message
	mM, err := readMail(mailString)
	if err != nil {
		log.Println(s.MailQueue+" Cannot convert mail to *mail.Message ", err)
		s.saveMail()
		s.sendNotification("Cannot convert mail to *mail.Message")
		sendMail(sender, recipients, s.MailData)
		os.Exit(0)
	}

	// Check header
	header, err := mailHeader(mM)
	if err != nil {
		log.Println(s.MailQueue+" Cannot parse mailHeader ", err)
		s.saveMail()
		s.sendNotification("Cannot parse mailHeader")
		sendMail(sender, recipients, s.MailData)
		os.Exit(0)
	}

	// Check "From" from header
	senderHeader, err := MailSenderHeader(header)
	log.Println(s.MailQueue + " Sender from *mail.Message: " + senderHeader)
	if err != nil {
		log.Println(s.MailQueue+" Cannot check From header: ", err)
		s.saveMail()
		s.sendNotification("Cannot check From header")
		sendMail(sender, recipients, s.MailData)
		os.Exit(0)
	}

	// Check if Sender From has a correct format
	senderFromFormat, err := senderFormat(senderHeader)
	if err != nil {
		log.Println(s.MailQueue+" Cannot check From format: ", err)
		s.saveMail()
		s.sendNotification("Cannot check From format")
		sendMail(sender, recipients, s.MailData)
		os.Exit(0)
	}

	if senderFromFormat == false {
		log.Println(s.MailQueue+" Wrong From format: ", err)
		s.saveMail()
		s.sendNotification("Wrong From format")
		sendMail(sender, recipients, s.MailData)
		os.Exit(0)
	}

	/*
	  Checking if sender is in our WHITELIST
	  Pass an e-mail if it is.
	*/
	// return user@domain.com (from ARG)
	userFromArg, err := CheckUserNameFromList(sender)
	if err != nil {
		log.Println(s.MailQueue+" Error when checking userFromArg: ", err)
		s.saveMail()
		s.sendNotification("Error when checking userFromArg")
		sendMail(sender, recipients, s.MailData)
		os.Exit(0)
	}

	// return user@domain.com (from Headers)
	userFromHeaders, err := CheckUserNameFromList(senderHeader)
	if err != nil {
		log.Println(s.MailQueue+" Error when checking userFromHeaders: ", err)
		s.saveMail()
		s.sendNotification("Error when checking userFromHeaders")
		sendMail(sender, recipients, s.MailData)
		os.Exit(0)
	}

	// checking if user is on the list
	userResult, err := CheckUserFromList(sender)
	if err != nil {
		log.Println(s.MailQueue+" Error when checking userResult: ", err)
		s.saveMail()
		s.sendNotification("Error when checking userResult")
		sendMail(sender, recipients, s.MailData)
		os.Exit(0)
	}

	// checking if user is on the list
	userResultHeader, err := CheckUserFromList(senderHeader)
	if err != nil {
		log.Println(s.MailQueue+" Error when checking userResultHeader: ", err)
		s.saveMail()
		s.sendNotification("Error when checking userResultHeader")
		sendMail(sender, recipients, s.MailData)
		os.Exit(0)
	}

	/*
	   Checking if sender is from OURDOMAIN.
	   Block if it is.
	   Pass if it's not.
	*/
	domainResult, err := CheckDomain(senderHeader)
	if err != nil {
		log.Println(s.MailQueue+" Error when checking domainResult: ", err)
		s.saveMail()
		s.sendNotification("Error when checking domainResult")
		sendMail(sender, recipients, s.MailData)
		os.Exit(0)
	}

	// checking whitelist
	whitelistDomainFromArg, err := whitelistDomainCheck(sender)
	if err != nil {
		log.Println(s.MailQueue+" Error when checking whitelistDomainFromArgt: ", err)
		s.saveMail()
		s.sendNotification("Error when checking whitelistDomainFromArg")
		sendMail(sender, recipients, s.MailData)
		os.Exit(0)
	}

	// checking whitelist
	whitelistDomainFromHeaders, err := whitelistDomainCheck(senderHeader)
	if err != nil {
		log.Println(s.MailQueue+" Error when checking whitelistDomainFromHeaders: ", err)
		s.saveMail()
		s.sendNotification("Error when checking whitelistDomainFromHeaderst")
		sendMail(sender, recipients, s.MailData)
		os.Exit(0)
	}

	// check whitelist domains
	if whitelistDomainFromArg == true || whitelistDomainFromHeaders == true {
		log.Println(s.MailQueue + " PASSED (WHITELISTED domain): " + sender + " => " + recipients)
		doMail := sendMail(sender, recipients, s.MailData)
		if doMail != nil {
			log.Println(s.MailQueue+" sendMail() ", doMail)
		}
		os.Exit(0)
	}

	// if domainResult == true {
	// and userResult || userResultHeader (those are addresses from the SLICE LIST)
	if domainResult == true && ((userResult == true) || (userResultHeader == true)) {
		if userFromArg != userFromHeaders {
			// Log it
			log.Println(s.MailQueue + " BLOCKED: " + sender + " => " + recipients)

			// Send notification
			s.sendNotification("BLOCKED: " + sender + "=>" + recipients)

			// Archive Mail
			log.Println(s.MailQueue + " saved to: " + archiveFile)
			s.saveMail()

			call := &APIStruct{
				Sender:       sender,
				SenderHeader: senderHeader,
				Recipient:    recipients,
				Queue:        queue,
				Blocked:      true,
			}

			// Check DEBUG mode
			if DEBUG == true {
				doMail := sendMail(sender, recipients, s.MailData)
				if doMail != nil {
					log.Println(s.MailQueue+" sendMail(debug=true) ", doMail)
				}
				log.Println(s.MailQueue + " mail has been sent (DEBUG=true) ")
				call.Blocked = false
				err = call.api()
				if err != nil {
					os.Exit(0)
				}
				os.Exit(0)
			} else {
				err = call.api()
				if err != nil {
					os.Exit(0)
				}
				os.Exit(0)
			}
		}
		// Log it
		log.Println(s.MailQueue + " PASSED: " + sender + " => " + recipients)
		doMail := sendMail(sender, recipients, s.MailData)
		if doMail != nil {
			log.Println(s.MailQueue+" sendMail() ", doMail)
		}
		os.Exit(0)
	}
	// Log it
	log.Println(s.MailQueue + " PASSED: " + sender + " => " + recipients)
	doMail := sendMail(sender, recipients, s.MailData)
	if doMail != nil {
		log.Println(s.MailQueue+" sendMail() ", doMail)
	}
	os.Exit(0)
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
func MailSenderHeader(header mail.Header) (string, error) {
	sender := header.Get("From")
	return sender, nil
}

/*
  Generate random string for a mail queue
*/
func RandStringBytesRmndr(n int) (string, error) {
	const letterBytes = "123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b), nil
}

/*
  Checking if sender has a corrent format
*/
func senderFormat(sender string) (bool, error) {
	senderOk := true
	numAt := strings.Count(sender, "@")
	if (numAt != 1) && (numAt != 2) {
		senderOk = false
	}
	return senderOk, nil
}

/*
  Checking if OURDOMAIN is the same as MAIL_FROM
*/
func CheckDomain(sender string) (bool, error) {
	rbool := false
	host := strings.Split(sender, "@")[1]
	hostfinal := strings.Split(host, ">")[0]
	for _, domain := range OURDOMAIN {
		// log.Println("Checking domain: " + domain)
		if hostfinal == domain {
			// log.Println(domain + " == " + hostfinal)
			rbool = true
			break
		}
	}
	return rbool, nil
}

/*
  Checking if e-mail name is in our WHITELIST slice
*/
func whitelistDomainCheck(sender string) (bool, error) {
	rbool := false
	domain := strings.Split(sender, "@")[1]
	if len(strings.Split(domain, "<")) == 2 {
		domain = strings.Split(domain, "<")[1]
	}
	for _, w := range WHITELIST {
		if domain == w {
			rbool = true
			break
		}
	}
	return rbool, nil
}

func CheckUserFromList(sender string) (bool, error) {
	rbool := false
	user := strings.Split(sender, "@")[0]
	if len(strings.Split(user, "<")) == 2 {
		user = strings.Split(user, "<")[1]
	}
	user = strings.ToLower(user)
	for _, w := range CheckUserList {
		// log.Println("Checking user from CheckUserList: " + user + "==" + w)
		if user == w {
			rbool = true
			break
		}
	}
	return rbool, nil
}

func CheckUserNameFromList(sender string) (string, error) {
	user := strings.Split(sender, "@")[0]
	domain := strings.Split(sender, "@")[1]

	if len(strings.Split(user, "<")) == 2 {
		user = strings.Split(user, "<")[1]
		tmp := strings.Split(sender, "@")[1]
		domain = strings.Split(tmp, ">")[0]
	}
	return user + "@" + domain, nil
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
		log.Println("Problem with sendmail() method: ", err)
	}

	err = sendmail.Start()
	if err != nil {
		log.Println("Problem with sendmail() method: ", err)
	}

	fmt.Fprintf(pipe, "%s", maildata)
	pipe.Close()

	return err
}

/*
  Convert mail raw to string
*/
func stdinToString(maildata []byte) (*strings.Reader, error) {
	r := strings.NewReader(string(maildata))
	return r, nil
}

/*
  Return header from the Mail.
*/
func mailHeader(m *mail.Message) (mail.Header, error) {
	header := m.Header
	return header, nil
}

/*
  Read mail string and return *mail.Message
*/
func readMail(r *strings.Reader) (*mail.Message, error) {
	m, err := mail.ReadMessage(r)
	return m, err
}

/*
  Read mail and add it to the MailStruct
*/
func (s *MailStruct) readData() error {
	data, err := ioutil.ReadAll(os.Stdin)
	s.MailData = data
	return err
}

/*
  Save mail. Reasons:
    - blocked
	- error
*/
func (s *MailStruct) saveMail() error {
	err := ioutil.WriteFile(s.BackupFile, s.MailData, 0644)
	return err
}

/*
  Send e-mail notification
*/
func (s *MailStruct) sendNotification(body string) error {

	from := NotificationFrom
	recipients := NotificationRecipients
	subject := NotificationSubject
	MailBody := s.MailQueue + " " + body

	sendmail := exec.Command("/usr/sbin/sendmail", "-G", "-i", "-f", from, "--", recipients)
	pipe, err := sendmail.StdinPipe()
	if err != nil {
		log.Println("Problem with sendmail() method: ", err)
	}
	err = sendmail.Start()
	if err != nil {
		log.Println("Problem with sendmail() method: ", err)
	}

	fmt.Fprintf(pipe, "Subject: %s\n\n", subject)
	fmt.Fprintf(pipe, "%s", MailBody)
	pipe.Close()
	return err
}

func (call *APIStruct) api() error {
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(call)
	res, err := http.Post("http://localhost:8080/mails", "application/json; charset=utf-8", b)
	if err != nil {
		log.Println("Cannot connect to the API")
		os.Exit(0)
	}
	defer res.Body.Close()
	io.Copy(os.Stdout, res.Body)
	return nil
}
