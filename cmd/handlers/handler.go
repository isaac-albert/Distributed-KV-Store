package handlers

import (
	"fmt"
	"net/http"

	"www.github.com/isaac-albert/Distributed-KV-Store/internal/parser"
)

func MethodRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		PostHandler(w, r)
		return
	case http.MethodGet:
		GetHandler(w, r)
		return
	default:
		fmt.Fprintf(w, "invalid method")
		return
	}
}

func PostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		fmt.Fprintf(w, "empty body \n")
		return
	}
	err := parser.Parse(r.Body)
	if err != nil {
		http.Error(w, "error parsing the body", http.StatusBadRequest)
		return
	}
	fmt.Fprintf(w, "successfull key value post\n")
}

func GetHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "get method was successful \n")
}
