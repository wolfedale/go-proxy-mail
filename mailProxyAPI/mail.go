package main

/*
  Mail structure
*/
type Mail struct {
	Id           int    `json:"id"`
	Sender       string `json:"sender"`
	SenderHeader string `json:"senderheader"`
	Recipient    string `json:"recipient"`
	Date         string `json:"date"`
	Queue        string `json:"queue`
	Blocked      bool   `json:"blocked"`
}

/*
  Mails - is a slice of []Mail struct
  This is needed as we are going to create
  more then one e-mail and keep all of them
  in the same struct
*/
type Mails []Mail
