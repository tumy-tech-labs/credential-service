package issuer

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/bradtumy/credential-service/internal/httpserver"
)

// Router builds the issuer HTTP router with middleware.
func Router(svc *Service) http.Handler {
	r := mux.NewRouter()
	r.Use(httpserver.RequestContext)

	v1 := r.PathPrefix("/v1").Subrouter()
	v1.Handle("/credential", http.HandlerFunc(svc.IssueCredential)).Methods(http.MethodPost)

	return r
}
