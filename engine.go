package coquery

import (
	"bytes"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/influx6/coquery/data"
	"github.com/influx6/coquery/parser"
	"github.com/influx6/coquery/storage"
	"github.com/influx6/faux/panics"
)

//==============================================================================

// EventLog defines event logger that allows us to record events for a specific
// action that occured.
type EventLog interface {
	Log(context interface{}, name string, message string, data ...interface{})
	Error(context interface{}, name string, err error, message string, data ...interface{})
}

//==============================================================================

// QueryProcessor provides an interface that processes a query into a requests
// lists returning an error if the query had or was an invalid requests.
type QueryProcessor interface {
	Generate(context interface{}, rid string, doc string, queries []string) (RecordRequests, ResponseError)
}

// ResponseWriter defines a interface for custom response writers for a
// RecordRequest returned from the internals of a Document query processor.
type ResponseWriter interface {
	Write(context interface{}, rs *Response, re ResponseError) error
}

// Document define an interface for a backend provider which handles and
// replies the needed requests from a
type Document interface {
	Handle(context interface{}, rq RecordRequests, rw ResponseWriter)
}

// Doc provides a interface that allows a single-level of responsibility
// for the object that provides both its Document system and
// its QueryProcessor.
type Doc interface {
	Document() Document
	Queries() QueryProcessor
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
	DocumentWith(context interface{}, path string, doc Doc) DocumentRouter
	Document(context interface{}, path string, qs QueryProcessor, d Document) DocumentRouter
	Serve(context interface{}, rid string, path string, queries []string, rw ResponseWriter)
}

// docSet defines a structure for storing a query processor and a Document
// Engine pair.
type docSet struct {
	query QueryProcessor
	doc   Document
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

// DocumentWith provides a function which uses a Doc to
// simplifies the argument lists and uses the central system to provide
// its QueryProcessor and Documents operating system.
func (d *DocRoute) DocumentWith(context interface{}, subPath string, doc Doc) DocumentRouter {
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
			d.documents[subPath] = &docSet{query: doc.Queries(), doc: doc.Document()}
		}
		atomic.AddInt64(&d.docAdd, -1)
	}

	d.Log(context, "Document", "Completed")
	return d
}

// Document provides the method to register a document processor for a specific
// subroute of a router. If a subrouter is already being used, the request is
// ignored.
func (d *DocRoute) Document(context interface{}, subPath string, qs QueryProcessor, dc Document) DocumentRouter {
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

// ErrDocumentRoutePanic is returned when a document internal processing panics.
var ErrDocumentRoutePanic = errors.New("Document Paniced")

// Serve takes the requests needed and serves up the requests lists to the
// response writer.
func (d *DocRoute) Serve(context interface{}, requestID string, subPath string, queries []string, rw ResponseWriter) {
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
			Rid:    requestID,
			Msg:    fmt.Sprintf("Invalid Path[%s] Request", subPath),
			IError: errors.New("404"),
		}

		d.Error(context, "Serve", err, "Completed")

		rw.Write(context, nil, err)
		return
	}

	reqs, err := set.query.Generate(context, requestID, subPath, queries)
	if err != nil {
		d.Error(context, "Serve", err, "Completed")
		rw.Write(context, nil, err)
		return
	}

	if len(reqs) == 0 {
		err := &CoError{
			Rid:    requestID,
			Msg:    "No Request Generated",
			IError: errors.New("404"),
		}

		d.Error(context, "Serve", err, "Completed")
		rw.Write(context, nil, err)
		return
	}

	// Let the handler work in a go-routine and report a panic if any.
	panics.Defer(func() {
		d.Log(context, "Serve.GoRoutine", "Started : Req %s", requestID)
		set.doc.Handle(context, reqs, rw)
		d.Log(context, "Serve.GoRoutine", "Completed")
	}, func(report *bytes.Buffer) {
		d.Error(context, "Serve", ErrDocumentRoutePanic, "Panic : \n%s", report.String())
	})

	d.Log(context, "Serve", "Completed")
}

//==============================================================================

// Engine defines a interface for a coquery service providers.
type Engine interface {
	Route(context interface{}, root string) DocumentRouter
	Serve(context interface{}, ctx *data.RequestContext, rw ResponseWriter)
}

// New returns a new Engine implementing structure for interfacing with
// other API.
func New(el EventLog, diff Diffs, store storage.Store) Engine {
	co := CoEngine{
		EventLog: el,
		store:    store,
		diff:     diff,
		routers:  make(map[string]DocumentRouter),
	}
	return &co
}

//==============================================================================

// CoEngine provides a concrete implementation of the coquery engine server.
// It provides a two level deep routing level which allows providing
// multiple response engines for different backends.
type CoEngine struct {
	EventLog
	diff     Diffs
	routeAdd int64
	store    storage.Store
	routers  map[string]DocumentRouter
}

// Serve processes the query using the coquery parser and runs the internal
// pieces accordingly sending the parts into the appropriate route else
// responding with an appropriate error.
func (co *CoEngine) Serve(context interface{}, rctx *data.RequestContext, rw ResponseWriter) {
	co.Log(context, "Serve", "Started : Request ID[%s] : Queries %s", rctx.RequestID, rctx.Queries)

	if len(rctx.Queries) == 0 {
		err := &CoError{
			Rid:    rctx.RequestID,
			Msg:    "No Request Generated",
			IError: errors.New("404"),
		}

		rw.Write(context, nil, err)
		return
	}

	// The final ResponseWriter for this request.
	var rws ResponseWriter

	// The internal writer for this request.
	var inRws ResponseWriter

	// If data.RequestContext.NoJSON is false, then we are allowed to wrap the
	// response writer with our JSONResponseWriter else use the provided response
	// writer.
	if !rctx.NoJSON {

		// Create the JSON response writer for this request.
		inRws = &JSONResponseWriter{
			ctx:   rctx,
			res:   rw,
			store: co.store,
			diff:  co.diff,
		}

	} else {
		inRws = rw
	}

	// Is this request a query batch type?
	// If so, then batch the create the BatchResponseWriter to adequately batch
	// the response before using its provided writer to write the final response.
	if len(rctx.Queries) > 1 {
		rws = &BatchResponseWriter{
			Res:   inRws,
			total: len(rctx.Queries),
		}
	} else {
		rws = inRws
	}

	for _, qry := range rctx.Queries {
		co.serve(context, qry, rctx, rws)
	}

	co.Log(context, "Serve", "Completed")
}

// serve processes the individual query strings that are to be processed by
// the coquery.API, using the appropriate API calls needed.
func (co *CoEngine) serve(context interface{}, query string, rctx *data.RequestContext, rw ResponseWriter) {
	co.Log(context, "serve", "Started : RequestID[%s] : Query[%s]", rctx.RequestID, rctx.Queries)

	queryList := parser.ParseQuery(context, query)

	if dl := len(queryList); dl < 3 {
		err := &CoError{
			Rid:    rctx.RequestID,
			Msg:    fmt.Sprintf("Invalid Query: %s", query),
			IError: fmt.Errorf("Invalid Query Length %d", dl),
		}

		co.Error(context, "serve", err, "Completed")
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
			Rid:    rctx.RequestID,
			Msg:    fmt.Sprintf("Invalid Query Path[%s]", root),
			IError: errors.New("504"),
		}

		co.Error(context, "serve", err, "Completed")
		rw.Write(context, nil, err)
		return
	}

	sub := queryList[1]
	qs := queryList[2:]

	set.Serve(context, rctx.RequestID, sub, qs, rw)
	co.Log(context, "serve", "Completed")
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
