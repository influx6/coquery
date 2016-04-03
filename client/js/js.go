package js

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"time"

	"github.com/influx6/coquery/data"

	"honnef.co/go/js/xhr"
)

// ClientTimeout sets the default timeout for a requests based on the
// xhr XMLRequestHTTP API.
var ClientTimeout = 60 * time.Second

// HTTP provides the transport layer build on the XMLRequestHTTP provided by
// the browser to allow requests to be made to the backend from the client side.
var HTTP jsHTTP

type jsHTTP struct{}

// ErrFailedRequest is returned when a request response status fails below or
// above 2xx.
var ErrFailedRequest = errors.New("Request Failed")

// Do issues the requests and collects the response into a pack.
func (jsHTTP) Do(addr string, body io.Reader) (data.ResponsePack, error) {
	var d data.ResponsePack

	jsonBuff, err := ioutil.ReadAll(body)
	if err != nil {
		return d, err
	}

	req := xhr.NewRequest("POST", addr)
	// req.Timeout = int(ClientTimeout.Seconds())
	req.ResponseType = xhr.Text

	if err := req.Send(jsonBuff); err != nil {
		return d, err
	}

	// if req.ReadyState
	if req.Status < 200 || req.Status >= 300 {
		return d, ErrFailedRequest
	}

	// fmt.Printf("Text:%+s\n", req.ResponseText)

	var buf bytes.Buffer
	buf.Write([]byte(req.ResponseText))

	// Attempt to decode information into appropriate structure.
	if err := json.NewDecoder(&buf).Decode(&d); err != nil {
		return d, err
	}

	return d, nil
}
