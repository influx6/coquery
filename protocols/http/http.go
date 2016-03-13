// Package http provides a http protocol engine that builds on the coquery
// base level engine to allow tying this system into a http request-response
// server.
package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/influx6/coquery"
)

//==============================================================================

// EventLog defines event logger that allows us to record events for a specific
// action that occured.
type EventLog interface {
	Log(context interface{}, name string, message string, data ...interface{})
	Error(context interface{}, name string, err error, message string, data ...interface{})
}

//==============================================================================

// ResWriter provides a response writer for sending a coquery response
// to a http request.
type ResWriter struct {
	EventLog
	res http.ResponseWriter
	req *http.Request
}

// Write implements the coquery.ResponseWriter interface onto the
// ResWriter struct.
func (h *ResWriter) Write(context interface{}, rs *coquery.Response, re coquery.ResponseError) error {
	if re != nil {
		h.res.WriteHeader(http.StatusBadRequest)
		_, err := h.res.Write([]byte(re.Error()))
		return err
	}

	data, err := json.Marshal(rs.Data)
	if err != nil {
		h.res.WriteHeader(http.StatusBadRequest)
		h.res.Write([]byte(re.Error()))
		return err
	}

	_, err = h.res.Write(data)
	if err != nil {
		return err
	}

	return nil
}

//==============================================================================

// CoqueryHTTP provides a interface that combines the coquery.Engine and a http
// server handler.
type CoqueryHTTP interface {
	coquery.Engine
	http.Handler
}

// New returns a new CoqueryHTTP http server to respond to all coquery requests.
func New(e EventLog) CoqueryHTTP {
	coq := httpCoquery{
		EventLog: e,
		Engine:   coquery.New(e),
	}

	return &coq
}

//==============================================================================

// httpCoquery provides a http implementation of the coquery engine to work
// with a http server which allows us to serve coquery requests.
type httpCoquery struct {
	EventLog
	coquery.Engine
}

// ServeHTTP provides the http.Handler ServeHTTP method to serve http requests
// to a coquery.Engine.
func (h *httpCoquery) ServeHTTP(res http.ResponseWriter, req *http.Request) {

	// We want to provide flexibility in how a coquery request is received
	// from the user endpoint.
	// 1. When a POST and a ContentType application/x-www-form-urlencoded, that
	// has a query parameter
	// 2. When a GET and has a X-Coquery-Request header else checks if there is
	// a url parameter that has coquery="" which contains the query request.

	var query string
	var reqID string

	switch strings.ToLower(req.Method) {
	case "head":
		res.Header().Set("X-Coquery-Server", "HTTP-Coquery-1.0")
		res.Header().Set("Accepts", "application/x-www-form-urlencoded;")
		return

	case "post", "put":
		contentType := req.Header.Get("Content-Type")
		if !strings.Contains(contentType, "application/x-www.form-urlencoded") {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		if err := req.ParseForm(); err != nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		query = req.FormValue("coquery")

	case "get", "patch":
		xco := req.Header.Get("X-Coquery-Request")

		// If there exists no such then report as failure.
		if xco == "" {

			if err := req.ParseForm(); err != nil {
				res.WriteHeader(http.StatusBadRequest)
				return
			}

			xco = req.FormValue("coquery")
			if xco == "" {
				res.WriteHeader(http.StatusBadRequest)
				return
			}
		}

		query = xco
	}

	res.Header().Set("X-Coquery-Server", "HTTP-Coquery-1.0")
	res.Header().Set("X-Coquery-Request-ID", reqID)

	h.Serve("httpCoquery", reqID, query, &ResWriter{res: res, req: req})
}

//==============================================================================
