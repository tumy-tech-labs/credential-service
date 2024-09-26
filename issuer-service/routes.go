package main

import (
	"net/http"
)

// InitializeRoutes sets up the HTTP routes
func InitializeRoutes() {
	http.HandleFunc("/credential", issueCredential)
}
