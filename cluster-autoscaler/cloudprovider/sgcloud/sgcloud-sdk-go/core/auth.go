package core

import "net/http"

type Authenticator interface {
	Signature(*http.Request, interface{}) error
}
