package main

import (
	"bufio"
	"net/http"
)


func main() {
	http.HandleFunc("/main", MainHandler)
	http.ListenAndServe(":8080", nil)

}
