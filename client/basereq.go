package client

import (
	"sync/atomic"

	"github.com/influx6/coquery/client/data"
)

//==============================================================================

// PerRequestHandler defines a handler type for receving a per data response.
type PerRequestHandler func(error, data.Response)

// RequestHandler defines a handler type for receving a per data response.
type RequestHandler func(error, []data.Response)

// Requestor provides a interface for requesting data from the giving endpoint.
type Requestor interface {
	Do() error
	UUID() string
	Listen(rx RequestHandler)
	ListenFor(key interface{}, rx PerRequestHandler)
	Receive(err error, d data.Pack)
	ShouldUpdate(deltas []string) bool
}

//==============================================================================

// BaseRequestor defines a base requester which implements the Requestor
// interfacing, which is returned on a registered query by a Server.
type BaseRequestor struct {
	records   map[interface{}]bool //contains keys of all records received through it.
	handles   []RequestHandler     // lists of callbacks registered to this provider.
	uuid      string               // unique uuid for this provider.
	query     string               // unique query for this provider.
	key       string               // unique key for records.
	server    Server               // internal server to which this provider belongs.
	pending   int64
	keyUpdate int64
}

// Do sends of the query to be serviced by the server, processing all necessary
// and passing information off to interested handlers.
func (b *BaseRequestor) Do() error {
	return b.server.serve(b.query, b)
}

// ListenFor allows listening for a specific record recieved from the server
// using the provided key for that record. Because BaseRequestor stores all
// record keys retrieved from the server using the key attribute received
// from the server in the data.Pack.
func (b *BaseRequestor) ListenFor(key interface{}, rx PerRequestHandler) {
	atomic.StoreInt64(&b.pending, 1)
	{
		b.handles = append(b.handles, func(err error, records []data.Response) {
			if err != nil {
				rx(err, nil)
				return
			}

			for _, record := range records {
				if record[b.key] != key {
					continue
				}

				rx(nil, record)
				return
			}
		})
	}
	atomic.StoreInt64(&b.pending, 0)
}

// Listen provides a general purpose listen method where all registered will
// be notified of updates from the server.
func (b *BaseRequestor) Listen(rx RequestHandler) {
	atomic.StoreInt64(&b.pending, 1)
	{
		b.handles = append(b.handles, rx)
	}
	atomic.StoreInt64(&b.pending, 0)
}

// ShouldUpdate takes the recieved delta record keys from the server and compares
// them with its internal records return TRUE if any keys match, which
// automatically schedules it for a update from the server end.
func (b *BaseRequestor) ShouldUpdate(deltas []string) bool {
	var found bool

	atomic.StoreInt64(&b.keyUpdate, 1)
	{
		for _, key := range deltas {
			if b.records[key] {
				found = true
				break
			}
		}
	}
	atomic.StoreInt64(&b.keyUpdate, 0)

	return found
}

// Receive provides the central method for providing record updates to the
// registered callbacks for this
func (b *BaseRequestor) Receive(err error, data data.Pack) {
	// If error occured, to ensure our callbacks are not left starving, notify
	// everyone of error and let them react accordingly.
	if err != nil {
		atomic.StoreInt64(&b.pending, 1)
		{
			for _, client := range b.handles {
				go client(err, nil)
			}
		}
		atomic.StoreInt64(&b.pending, 0)
		return
	}

	b.key = data.RecordKey

	atomic.StoreInt64(&b.keyUpdate, 1)
	{

		// Collect all record keys and store them for so we can review the delta
		// lists incase we need to make requests for updates
		for _, record := range data.Results {
			key := record[data.RecordKey]
			b.records[key] = true
		}

	}
	atomic.StoreInt64(&b.keyUpdate, 0)

	atomic.StoreInt64(&b.pending, 1)
	{
		// Notify all clients of new arrivals.
		for _, client := range b.handles {
			go client(nil, data.Results)
		}

	}
	atomic.StoreInt64(&b.pending, 0)
}

// UUID returns uuid for this provider.
func (b *BaseRequestor) UUID() string {
	return b.uuid
}
