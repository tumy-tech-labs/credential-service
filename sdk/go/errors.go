package sdk

import "errors"

var (
	// ErrUnauthorized indicates the credential or request was not authorized.
	ErrUnauthorized = errors.New("unauthorized")
	// ErrInvalidCredential indicates the provided credential was invalid or malformed.
	ErrInvalidCredential = errors.New("invalid_credential")
	// ErrServerError represents a server-side failure.
	ErrServerError = errors.New("server_error")
	// ErrNetwork represents network or transport failures.
	ErrNetwork = errors.New("network_error")
)

func mapStatusToError(status int) error {
	switch {
	case status == 400:
		return ErrInvalidCredential
	case status == 401 || status == 403:
		return ErrUnauthorized
	case status >= 500:
		return ErrServerError
	default:
		return ErrNetwork
	}
}
