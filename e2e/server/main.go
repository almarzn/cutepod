package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	msg := os.Getenv("MESSAGE")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, msg)
	})
	http.ListenAndServe(":8080", nil)
}
