// Package cohttp provides a http protocol engine that builds on the coquery
// base level engine to allow tying this system into a http request-response
// server.
package cohttp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/influx6/coquery"
	"github.com/influx6/coquery/storage"
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
	h.Log(context, "cohttp.ResWriter.Write", "Started")

	if re != nil {
		h.Error(context, "cohttp.ResWriter.Write", re, "Completed")
		h.res.WriteHeader(http.StatusBadRequest)
		_, err := h.res.Write([]byte(re.Error()))
		if err != nil {
			h.Error(context, "cohttp.ResWriter.Write", err, "Info : Response Write Error")
			return err
		}

		return nil
	}

	h.Log(context, "cohttp.ResWriter.Write", "Info : JSON.Marshal : %s", fmt.Sprintf("%+v", rs.Data))

	var data []byte
	var err error

	if len(rs.Data) > 1 {
		data, err = json.Marshal(rs.Data)
	} else {
		data, err = json.Marshal(rs.Data[0])
	}

	if err != nil {
		h.Error(context, "cohttp.ResWriter.Write", err, "Completed")
		h.res.WriteHeader(http.StatusBadRequest)
		h.res.Write([]byte(err.Error()))
		return err
	}

	h.Log(context, "cohttp.ResWriter.Write", "Info : Response JSON : %s", data)

	h.res.Header().Set("Content-Type", "application/json")
	h.res.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	_, err = h.res.Write(data)
	if err != nil {
		h.Error(context, "cohttp.ResWriter.Write", err, "Completed")
		return err
	}

	h.Log(context, "cohttp.ResWriter.Write", "Completed")
	return nil
}

//==============================================================================

// CoqueryHTTP provides a interface that combines the coquery.Engine and a http
// server handler.
type CoqueryHTTP interface {
	coquery.Engine
	http.Handler
	ListenAndServe(context interface{}, addr string)
	EnableCORS()
}

// New returns a new CoqueryHTTP http server to respond to all coquery requests.
func New(e EventLog, diff coquery.Diffs, store storage.Store) CoqueryHTTP {
	coq := httpCoquery{
		EventLog: e,
		Engine:   coquery.New(e, diff, store),
	}

	return &coq
}

//==============================================================================

// httpCoquery provides a http implementation of the coquery engine to work
// with a http server which allows us to serve coquery requests.
type httpCoquery struct {
	EventLog
	coquery.Engine
	useCORS bool
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

// EnableCORS flips the flag to add CORS headers to all response to true.
func (h *httpCoquery) EnableCORS() {
	h.useCORS = true
}

// ServeHTTP provides the http.Handler ServeHTTP method to serve http requests
// to a coquery.Engine.
func (h *httpCoquery) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	h.Log("HTTPCoquery", "ServeHTTP", "Started : Request %s : Method %s", req.URL.String(), req.Method)

	// We want to provide flexibility in how a coquery request is received
	// from the user endpoint.
	// 1. When a POST and a ContentType application/x-www-form-urlencoded, that
	// has a query parameter
	// 2. When a GET and has a X-Coquery-Request header else checks if there is
	// a url parameter that has coquery="" which contains the query request.

	res.Header().Set("X-Coquery-Version", "1.0")
	res.Header().Set("Methods", "HEAD, GET, POST, PUT, PATCH")

	if h.useCORS {
		res.Header().Set("Access-Control-Allow-Origin", "*")
		res.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		res.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		res.Header().Set("Access-Control-Max-Age", "86400")
		res.Header().Set("Content-Type", "application/json")
	}

	method := strings.ToLower(req.Method)

	// If this is a head method then stop here.
	if method == "head" {
		return
	}

	req.ParseForm()
	// req.ParseMultipartForm(maxMemory int64)

	var rctx coquery.RequestContext

	contentType := req.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {

		if err := json.NewDecoder(req.Body).Decode(&rctx); err != nil {
			req.Body.Close()
			res.WriteHeader(http.StatusBadRequest)
			res.Write([]byte(err.Error()))
			h.Error("HTTPCoquery", "ServeHTTP", err, "Completed")
			return
		}

		req.Body.Close()

		res.Header().Set("X-Coquery-Request-ID", rctx.RequestID)

		h.Serve("httpCoquery", &rctx, &ResWriter{
			EventLog: h.EventLog,
			res:      res,
			req:      req,
		})

		h.Log("HTTPCoquery", "ServeHTTP", "Completed")
		return
	}

	// if strings.Contains(contentType, "application/x-www-form-urlencoded") {
	rid := req.Form.Get("requestid")

	if strings.TrimSpace(rid) != "" {
		rctx.RequestID = rid
	} else {
		rctx.RequestID = uuid.New()
	}

	qrs := req.Form.Get("coquery")

	if strings.TrimSpace(qrs) == "" {
		res.WriteHeader(http.StatusBadRequest)
		h.Error("HTTPCoquery", "ServeHTTP", fmt.Errorf("Bad Request: %d", http.StatusBadGateway), "Completed")
		return
	}

	rctx.Queries = []string{qrs}

	// }

	res.Header().Set("X-Coquery-Request-ID", rctx.RequestID)

	// // Did we catch our target content-types? If not fail the request.
	// if !ok {
	// 	res.WriteHeader(http.StatusBadRequest)
	// 	h.Error("HTTPCoquery", "ServeHTTP", fmt.Errorf("Bad Request: %d", http.StatusBadGateway), "Completed")
	// 	return
	// }

	h.Serve("httpCoquery", &rctx, &ResWriter{
		EventLog: h.EventLog,
		res:      res,
		req:      req,
	})

	h.Log("HTTPCoquery", "ServeHTTP", "Completed")
}

//==============================================================================
