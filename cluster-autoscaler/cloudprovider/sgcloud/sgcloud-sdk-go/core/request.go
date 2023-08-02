package core

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
)

// Request is http request, but has some custom functions.
type Request http.Request

// NewRequest returns a request client
func NewRequest(method, url string, body interface{}) (*Request, error) {
	method = strings.ToUpper(method)

	buff := &bytes.Buffer{}
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		buff = bytes.NewBuffer(b)
	}
	rawRequest, err := http.NewRequest(method, url, buff)

	if file, ok := body.(*os.File); ok {
		fileInfo, err := file.Stat()

		if err != nil {
			return nil, err
		}

		rawRequest.ContentLength = fileInfo.Size()
	}

	req := (*Request)(rawRequest)
	return req, err
}

func (req *Request) raw() *http.Request {
	return (*http.Request)(req)
}
