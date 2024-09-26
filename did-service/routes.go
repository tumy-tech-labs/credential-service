package main

import (
	"net/http"
)

func InitializeRoutes() {
	http.HandleFunc("/dids", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			createDID(w, r)
		} else if r.Method == http.MethodGet {
			getDIDs(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/dids/resolver", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			getDID(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}
