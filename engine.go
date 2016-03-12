package coquery

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/influx6/coquery/parser"
)

//==============================================================================

// EventLog defines event logger that allows us to record events for a specific
// action that occured.
type EventLog interface {
	Log(context interface{}, name string, message string, data ...interface{})
	Error(context interface{}, name string, err error, message string, data ...interface{})
}

//==============================================================================

// RecordRequest defines a base type for the supported request types
type RecordRequest interface {
	RequestName() string
}

// RecordRequests defines a lists of record requests genered from a query
// proccessor.
type RecordRequests []RecordRequests

//==============================================================================

// QueryProcessor provides an interface that processes a query into a requests
// lists returning an error if the query had or was an invalid requests.
type QueryProcessor interface {
	Generate(context interface{}, queries []string) (RecordRequests, ResponseError)
}

// ResponseWriter defines a interface for custom response writers for a
// RecordRequest returned from the internals of a Document query processor.
type ResponseWriter interface {
	Write(context interface{}, rs *Response, re ResponseError) error
}

// Documents define an interface for a backend provider which handles and
// replies the needed requests from a
type Documents interface {
	Handle(context interface{}, rq RecordRequests, rw ResponseWriter)
}

//==============================================================================

// CoError provides a custom error message for requests types.
type CoError struct {
	Rid    string `json:"rid" bson:"rid"`
	Msg    string `json:"message" bson:"message"`
	IError error  `json:"error" bson:"error"`
}

// Message returns the internal message for this error
func (r *CoError) Message() string {
	return r.Msg
}

// RequestID returns the response error requestID
func (r *CoError) RequestID() string {
	return r.Rid
}

// Error returns the error message for this response error.
func (r *CoError) Error() string {
	if r.IError != nil {
		return r.Rid + " : " + r.Msg + " : " + r.IError.Error()
	}

	return r.Rid + " : " + r.Msg
}

//==============================================================================

// DocumentRouter defines a interface that defines a means for registering
// document providers for request processing.
type DocumentRouter interface {
	Document(context interface{}, path string, qs QueryProcessor, d Documents) DocumentRouter
	Serve(context interface{}, path string, queries []string, rw ResponseWriter)
}

// docSet defines a structure for storing a query processor and a Document
// Engine pair.
type docSet struct {
	query QueryProcessor
	doc   Documents
}

// DocRoute defines a coquery engine system for routing and management of
// coquery requests.
type DocRoute struct {
	EventLog
	docAdd    int64
	documents map[string]*docSet
}

// NewDocRoute returns a new instance of a DocRoute.
func NewDocRoute(elog EventLog) *DocRoute {
	dr := DocRoute{
		EventLog:  elog,
		documents: make(map[string]*docSet),
	}

	return &dr
}

// Document provides the method to register a document processor for a specific
// subroute of a router. If a subrouter is already being used, the request is
// ignored.
func (d *DocRoute) Document(context interface{}, subPath string, qs QueryProcessor, dc Documents) DocumentRouter {
	d.Log(context, "Document", "Started : Register Document : %s", subPath)
	var ok bool

	atomic.AddInt64(&d.docAdd, 1)
	{
		_, ok = d.documents[subPath]
	}
	atomic.AddInt64(&d.docAdd, -1)

	if !ok {
		atomic.AddInt64(&d.docAdd, 1)
		{
			d.documents[subPath] = &docSet{query: qs, doc: dc}
		}
		atomic.AddInt64(&d.docAdd, -1)
	}

	d.Log(context, "Document", "Completed")
	return d
}

// Serve takes the requests needed and serves up the requests lists to the
// response writer.
func (d *DocRoute) Serve(context interface{}, subPath string, queries []string, rw ResponseWriter) {
	d.Log(context, "Serve", "Started : Path[%s] : Query: %s", subPath, queries)

	var ok bool
	var set *docSet

	atomic.AddInt64(&d.docAdd, 1)
	{
		set, ok = d.documents[subPath]
	}
	atomic.AddInt64(&d.docAdd, -1)

	if !ok {
		err := &CoError{
			Rid:    "DocumentRouter",
			Msg:    fmt.Sprintf("Invalid Path[%s] Request", subPath),
			IError: errors.New("404"),
		}

		d.Error(context, "Serve", err, "Completed")

		rw.Write(context, nil, err)
		return
	}

	reqs, err := set.query.Generate(context, queries)
	if err != nil {
		d.Error(context, "Serve", err, "Completed")
		rw.Write(context, nil, err)
		return
	}

	set.doc.Handle(context, reqs, rw)
	d.Log(context, "Serve", "Completed")
}

//==============================================================================

// Engine defines a interface for a coquery service providers.
type Engine interface {
	Route(context interface{}, root string) DocumentRouter
	Serve(context interface{}, query string, rw ResponseWriter)
}

// CoEngine provides a concrete implementation of the coquery engine server.
// It provides a two level deep routing level which allows providing
// multiple response engines for different backends.
type CoEngine struct {
	EventLog
	routers  map[string]DocumentRouter
	routeAdd int64
}

// Serve processes the query using the coquery parser and runs the internal
// pieces accordingly sending the parts into the appropriate route else
// responding with an appropriate error.
func (co *CoEngine) Serve(context interface{}, query string, rw ResponseWriter) {
	co.Log(context, "Serve", "Started : Query[%s]", query)

	queryList := parser.ParseQuery(context, query)

	if dl := len(queryList); dl < 3 {
		err := &CoError{
			Rid:    "CoEngine",
			Msg:    fmt.Sprintf("Invalid Query: %s", query),
			IError: fmt.Errorf("Invalid Query Length %d", dl),
		}

		co.Error(context, "Serve", err, "Completed")
		rw.Write(context, nil, err)
		return
	}

	var ok bool
	var set DocumentRouter

	root := queryList[0]

	atomic.AddInt64(&co.routeAdd, 1)
	{
		set, ok = co.routers[root]
	}
	atomic.AddInt64(&co.routeAdd, -1)

	if !ok {
		err := &CoError{
			Rid:    "CoEngine",
			Msg:    fmt.Sprintf("Invalid Query Path[%s]", root),
			IError: errors.New("504"),
		}

		co.Error(context, "Serve", err, "Completed")
		rw.Write(context, nil, err)
		return
	}

	sub := queryList[1]
	qs := queryList[2:]

	set.Serve(context, sub, qs, rw)
	co.Log(context, "Serve", "Completed")
}

// Route sets up a document router for handling subdocuments for this
// specific route.
func (co *CoEngine) Route(context interface{}, root string) DocumentRouter {
	co.Log(context, "Route", "Started : Add Route : Route[%s]", root)

	var ok bool
	var doc DocumentRouter

	atomic.AddInt64(&co.routeAdd, 1)
	{
		doc, ok = co.routers[root]
	}
	atomic.AddInt64(&co.routeAdd, -1)

	if ok {
		co.Log(context, "Route", "Completed")
		return doc
	}

	doc = NewDocRoute(co.EventLog)

	atomic.AddInt64(&co.routeAdd, 1)
	{
		co.routers[root] = doc
	}
	atomic.AddInt64(&co.routeAdd, -1)

	co.Log(context, "Route", "Completed")
	return doc
}

//==============================================================================
