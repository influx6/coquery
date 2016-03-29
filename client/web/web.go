package web

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/influx6/coquery/data"
)

// HTTP provides a handle for the http request processor.
var HTTP webHTTP

type webHTTP struct{}

var client = http.Client{Timeout: 30 * time.Second}

// Do issues the requests and collects the response into a pack.
func (webHTTP) Do(addr string, body io.Reader) (data.ResponsePack, error) {
	var d data.ResponsePack

	// Make a post requests with a application/json body.
	res, err := client.Post(addr, "application/json", body)
	if err != nil {
		return d, err
	}

	defer res.Body.Close()

	// Attempt to decode information into appropriate structure.
	if err := json.NewDecoder(res.Body).Decode(&d); err != nil {
		return d, err
	}

	return d, nil
}
