package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

var mu sync.Mutex
var hit int
var block int
var pass int
var START time.Time = time.Now()

func main() {
	http.HandleFunc("/hit", hitme)
	http.HandleFunc("/block", blocked)
	http.HandleFunc("/pass", passed)
	http.HandleFunc("/stats", stats)

	log.Fatal(http.ListenAndServe("localhost:8000", nil))
}

func hitme(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	hit++
	mu.Unlock()
}

func blocked(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	block++
	mu.Unlock()
}

func passed(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	pass++
	mu.Unlock()
}

func stats(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	fmt.Fprintf(w, "Running from: %v\n", START)
	fmt.Fprintf(w, "Running since: %.2fs\n", time.Since(START).Minutes())
	fmt.Fprintf(w, "\nHits: %d\n", hit)
	fmt.Fprintf(w, "Blocked: %d\n", block)
	fmt.Fprintf(w, "Passed: %d\n", pass)

	all := block + pass
	if hit == all {
		fmt.Fprintf(w, "\n[Count OK]: %d\n", all)
	} else {
		fmt.Fprintf(w, "\n[Count Not OK]: %d\n", all)
	}
	mu.Unlock()
}
