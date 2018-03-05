package main

/*
  simple structure for errors
*/

type jsonErr struct {
	Code int    `json:"code"`
	Text string `json:"text"`
}
