package web

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/influx6/coquery"
)

// HTTP provides a handle for the http request processor.
var HTTP webHTTP

type webHTTP struct{}

var client = http.Client{Timeout: 30 * time.Second}

// Do issues the requests and collects the response into a pack.
func (webHTTP) Do(addr string, body io.Reader) (coquery.ResponsePack, error) {
	var data coquery.ResponsePack

	// Make a post requests with a application/json body.
	res, err := client.Post(addr, "application/json", body)
	if err != nil {
		return data, err
	}

	defer res.Body.Close()

	// Attempt to decode information into appropriate structure.
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return data, err
	}

	return data, nil
}
