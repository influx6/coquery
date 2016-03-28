package client

import (
	"bytes"
	"encoding/json"
	"io"
	"sync/atomic"
	"time"

	"github.com/influx6/coquery"
	"github.com/pborman/uuid"
)

//==============================================================================

// ServeTransport defines an interface for requests transport, which allows us
// build custom transports based on different low-level systems(HTTP,Websocket).
type ServeTransport interface {
	Do(endpoint string, body io.Reader) (coquery.ResponsePack, error)
}

// Server provides a central request manager for different query requests and
// subscriptions.
type Server interface {
	Register(query string) Requestor
	serve(query string, client Requestor) error
}

//==============================================================================

// Servo defines a concrete implementation of the Server interface.
// It handles scheduling query requests and providing the appropriate
// Response parameter for each requster.
type Servo struct {
	pending      int64
	addr         string
	uuid         string
	pendingTime  time.Time
	wait         time.Duration
	transport    ServeTransport
	pendingQuery map[string]int
	providers    map[string]Requestor
	lastPack     coquery.ResponsePack
}

// NewServo creates a new Servo instance. It takes a coquery server address
// and the maximum time to wait to allow requests batching and the underline
// transport to be used to make such requests with.
func NewServo(addr string, wait time.Duration, transport ServeTransport) *Servo {
	if wait == 0 {
		wait = 500 * time.Millisecond
	}

	svo := Servo{
		addr:      addr,
		wait:      wait,
		uuid:      uuid.New(),
		transport: transport,
		providers: make(map[string]Requestor),
	}

	return &svo
}

// Register adds a query provider into the service lists else returns the
// provider if the query already exists. This allows a central point of
// responsibility for how queries are processed and managed.
func (s *Servo) Register(query string) Requestor {
	var provider Requestor
	var ok bool

	atomic.StoreInt64(&s.pending, 1)
	{
		provider, ok = s.providers[query]
		if !ok {
			provider = NewBaseRequester(query, s)
			s.providers[query] = provider
		}
	}
	atomic.StoreInt64(&s.pending, 0)

	return provider
}

// serve process the requests queries which will be batched and sent within a
// specified timing these allows us to batch and send as much request over
// specific period of times without wasting bandwidth.
func (s *Servo) serve(query string, client Requestor) error {

	// If we have already sent the data that has been queued, then
	// reset all details accordinly and prepare to to batch new requests.
	if s.pendingQuery == nil {
		s.pendingTime = time.Now().Add(s.wait)
		s.pendingQuery = make(map[string]int)
	}

	// Add the pending query with the right index.
	index := len(s.pendingQuery)
	s.pendingQuery[query] = index

	if !time.Now().After(s.pendingTime) {
		return nil
	}

	// Collect all the queries with their specified index and allocated into
	// a prelength list, build json request body and send off to transport
	// for delivery to endpoint.
	queries := make([]string, len(s.pendingQuery))

	for qry, index := range s.pendingQuery {
		queries[index] = qry
	}

	var prevDiff string

	atomic.StoreInt64(&s.pending, 1)
	{
		prevDiff = s.lastPack.DeltaID
	}
	atomic.StoreInt64(&s.pending, 0)

	var data coquery.RequestContext
	data.RequestID = s.uuid
	data.Queries = queries
	data.Diffs = true

	// if s.lastPack != nil {
	data.DiffTag = prevDiff
	// data.DiffWatch = s.lastPack.Deltas
	// }

	var buf bytes.Buffer

	// Attemp to encode the request data as json else return error.
	if err := json.NewEncoder(&buf).Encode(&data); err != nil {
		return err
	}

	reply, err := s.transport.Do(s.addr, &buf)
	if err != nil {
		s.pendingQuery = nil

		// Notify all concerned providers of error.
		atomic.StoreInt64(&s.pending, 1)
		{

			for qry := range s.pendingQuery {
				s.providers[qry].Receive(err, reply)
			}

		}
		atomic.StoreInt64(&s.pending, 0)

		return err
	}

	var pending = s.pendingQuery

	s.pendingQuery = nil
	s.lastPack = reply

	// Notify all concerned providers of response.
	atomic.StoreInt64(&s.pending, 1)
	{

		for qry := range s.pendingQuery {
			s.providers[qry].Receive(nil, reply)
		}

		// Check if last delta tag is same as the new recieved reply, if it is not
		// then proceed update check cycle.
		if reply.DeltaID != prevDiff && len(reply.Deltas) > 0 {

			// Check the providers who were not queue if they need to be updated and
			// schedule updates accordingly.

			for key, provider := range s.providers {
				if _, ok := pending[key]; ok {
					continue
				}

				if provider.ShouldUpdate(reply.Deltas) {
					s.serve(query, provider)
				}
			}

		}

	}
	atomic.StoreInt64(&s.pending, 0)

	return nil
}
