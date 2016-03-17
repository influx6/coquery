// Package cohttp provides a http protocol engine that builds on the coquery
// base level engine to allow tying this system into a http request-response
// server.
package cohttp

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/influx6/coquery"
	"github.com/pborman/uuid"
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
	var status string

	if rs != nil {
		status = "OK!"
	} else {
		status = "Error!"
	}

	h.Log(context, "Write", "HTTP : ResponseWriter : Status : %s", status)

	if re != nil {
		h.Error(context, "Write", re, "Completed")
		h.res.WriteHeader(http.StatusBadRequest)
		_, err := h.res.Write([]byte(re.Error()))
		return err
	}

	h.Log(context, "Write", "Info : Marshalling Data To JSON : %s", fmt.Sprintf("%+v", rs.Data))

	data, err := json.Marshal(rs.Data)
	if err != nil {
		h.Error(context, "Write", err, "Completed")
		h.res.WriteHeader(http.StatusBadRequest)
		h.res.Write([]byte(err.Error()))
		return err
	}

	h.Log(context, "Write", "Info : Write JSON : %s", data)
	_, err = h.res.Write(data)
	if err != nil {
		h.Error(context, "Write", err, "Completed")
		return err
	}

	h.Log(context, "Write", "Completed")
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

// ListenAndServe runs a http server with the httpCoquery instance wired
// to serve its incoming requests.
func (h *httpCoquery) ListenAndServe(context interface{}, addr string) {
	h.Log(context, "ListenAndServe", "Started : Addr[%s]", addr)

	// Lunch the http server in a goroutine.
	go func() {
		h.Log(context, "ListenAndServe", "Listening on: %s", addr)
		http.ListenAndServe(addr, h)
	}()

	// Listen for an interrupt signal from the OS.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan

	h.Log(context, "ListenAndServe", "Completed")
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
		req.ParseForm()
		reqID = req.FormValue("rid")

		if reqID == "" {
			reqID = uuid.New()
		}

		res.Header().Set("X-CoQuery-Version", "CoQuery.v1.0")
		res.Header().Set("X-CoQuery-Request-ID", reqID)
		res.Header().Set("Methods", "HEAD, GET, POST, PUT, PATCH")
		res.Header().Set("Accepts", "application/x-www-form-urlencoded;")
		return

	case "post", "put":
		contentType := req.Header.Get("Content-Type")

		isQuery := strings.Contains(contentType, "application/x-coquery")
		isForm := strings.Contains(contentType, "application/x-www-form-urlencoded")

		if !isForm && !isQuery {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		if isForm {
			req.ParseForm()

			query = req.FormValue("coquery")
			reqID = req.FormValue("rid")
		}

		if isQuery {
			defer req.Body.Close()

			bo, err := ioutil.ReadAll(req.Body)
			if err != nil {
				res.WriteHeader(http.StatusBadRequest)
				res.Write([]byte(err.Error()))
				return
			}

			query = string(bo)
		}

	case "get", "patch":
		xco := req.Header.Get("X-Coquery-Request")

		// If there exists no such then report as failure.
		if xco == "" {

			req.ParseForm()

			reqID = req.FormValue("rid")
			xco = req.FormValue("coquery")
			if xco == "" {
				res.WriteHeader(http.StatusBadRequest)
				return
			}
		}

		query = xco
	}

	if reqID == "" {
		reqID = uuid.New()
	}

	res.Header().Set("X-CoQuery-Version", "CoQuery.v1.0")
	res.Header().Set("X-CoQuery-Request-ID", reqID)

	h.Serve("httpCoquery", reqID, query, &ResWriter{res: res, req: req})
}

//==============================================================================
