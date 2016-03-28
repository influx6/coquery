package web

import (
	"net/http"
	"time"

	"github.com/influx6/coquery/client/data"
)

// HTTP provides a handle for the http request processor.
var HTTP httpc

type httpc struct{}

var client = http.Client{Timeout: 30 * time.Second}

// Do issues the requests and collects the response into a pack.
func (httpc) Do(rid string, queries []string, lastResponse data.Pack) (data.Pack, error) {
	var dp data.Pack

  json := map[string]interface{
    "request_id": rid,
    "queries": queries,
    "diff_tag": lastResponse.DeltaID,
    "diff_watch": lastResponse.Deltas,
    "diffs": lastResponse.
  }

	return dp, nil
}
