package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/influx6/coquery/data"
	"github.com/influx6/faux/utils"
)

//==============================================================================

// Events defines event logger that allows us to record events for a specific
// action that occured.
type Events interface {
	Log(context interface{}, name string, message string, data ...interface{})
	Error(context interface{}, name string, err error, message string, data ...interface{})
}

//==============================================================================

// ServeTransport defines an interface for requests transport, which allows us
// build custom transports based on different low-level systems(HTTP,Websocket).
type ServeTransport interface {
	Do(endpoint string, body io.Reader) (data.ResponsePack, error)
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
	Events
	pending      int64
	watching     int64
	addr         string
	uuid         string
	pendingTime  time.Time
	wait         time.Duration
	transport    ServeTransport
	pendingQuery map[string]int
	providers    map[string]Requestor
	lastPack     data.ResponsePack
}

// NewServo creates a new Servo instance. It takes a coquery server address
// and the maximum time to wait to allow requests batching and the underline
// transport to be used to make such requests with.
func NewServo(events Events, addr string, wait time.Duration, transport ServeTransport) *Servo {
	if wait == 0 {
		wait = 500 * time.Millisecond
	}

	svo := Servo{
		Events:    events,
		addr:      addr,
		wait:      wait,
		uuid:      utils.UUID(),
		transport: transport,
		providers: make(map[string]Requestor),
	}

	return &svo
}

// Register adds a query provider into the service lists else returns the
// provider if the query already exists. This allows a central point of
// responsibility for how queries are processed and managed.
func (s *Servo) Register(query string) Requestor {
	s.Events.Log("Servo", "Register", "Started : Registering Query[%s]", query)
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

	s.Events.Log("Servo", "Register", "Completed")
	return provider
}

// serve process the requests queries which will be batched and sent within a
// specified timing these allows us to batch and send as much request over
// specific period of times without wasting bandwidth.
func (s *Servo) serve(query string, client Requestor) error {
	s.Events.Log("Servo", "serve", "Started")

	if s.batch(query, client) {
		s.Events.Log("Servo", "serve", "Completed")
		return s.sendNow()
	}

	if atomic.LoadInt64(&s.watching) > 0 {
		s.Events.Log("Servo", "serve", "Completed")
		return nil
	}

	atomic.StoreInt64(&s.watching, 1)

	// Since we need to still make a request at the end of the set time,
	// we must schedule a go-routine to lunch the sendNow function when the
	// buffer delay time as passed else if it has already being resolved then ignore.
	go func() {
		// fmt.Printf("Initing sendNow() \n")
		<-time.After(s.wait + 2)
		if atomic.LoadInt64(&s.watching) == 0 {
			return
		}

		// fmt.Printf("Calling sendNow() \n")
		if err := s.sendNow(); err != nil {
			s.Events.Error("Servo", "serve", err, "Completed")
		}
	}()

	s.Events.Log("Servo", "serve", "Completed")
	return nil
}

// sendNow initializes and forwards the internal requests to the transport
// regardless of batching rules and limits.
func (s *Servo) sendNow() error {
	s.Events.Log("Servo", "sendNow", "Started")

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

	var mdata data.RequestContext
	mdata.RequestID = s.uuid
	mdata.Queries = queries
	mdata.Diffs = true

	// if s.lastPack != nil {
	mdata.DiffTag = prevDiff
	// data.DiffWatch = s.lastPack.Deltas
	// }

	var buf bytes.Buffer
	var reply data.ResponsePack

	// Attemp to encode the request data as json else return error.
	if err := json.NewEncoder(&buf).Encode(&mdata); err != nil {
		s.pendingQuery = nil

		// Notify all concerned providers of error.
		atomic.StoreInt64(&s.pending, 1)
		{

			for qry := range s.pendingQuery {
				s.providers[qry].Receive(err, reply)
			}

		}
		atomic.StoreInt64(&s.pending, 0)

		s.Events.Error("Servo", "sendNow", err, "Completed")
		return err
	}

	var err error

	// Deliver body to the transport layer.
	reply, err = s.transport.Do(s.addr, &buf)
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

		s.Events.Error("Servo", "sendNow", err, "Completed")
		return err
	}

	var pending = s.pendingQuery

	s.pendingQuery = nil
	s.lastPack = reply

	if len(reply.Results) < len(queries) {
		err := errors.New("Inadequate Response Length")
		s.Events.Error("Servo", "sendNow", err, "Completed")
		return err
	}

	atomic.StoreInt64(&s.pending, 1)
	{

		for ind, qry := range queries {
			if !reply.Batched {
				s.providers[qry].Receive(nil, reply)
				continue
			}

			localReply := reply
			localReply.Results = nil

			rez := reply.Results[ind]

			if failed, ok := rez["QueryFailed"].(bool); ok && failed {
				failedErr := fmt.Errorf("Message{%s} - Error{%s}", rez["Message"], rez["Error"])
				s.Events.Error("Servo", "sendNow", failedErr, "Info : Query [%s] : Failed", qry)
				s.providers[qry].Receive(failedErr, localReply)
				continue
			}

			mrdos := rez["data"]

			if mrdos == nil {
				s.providers[qry].Receive(nil, localReply)
				continue
			}

			mrd := mrdos.([]interface{})

			// var failedErr error

			for _, prec := range mrd {
				pmrec := prec.(map[string]interface{})

				localReply.Results = append(localReply.Results, data.Parameter(pmrec))
			}

			// if failedErr != nil {
			// 	s.providers[qry].Receive(failedErr, localReply)
			// 	continue
			// }

			s.providers[qry].Receive(nil, localReply)
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
					s.batch(key, provider)
				}
			}

		}

	}
	atomic.StoreInt64(&s.pending, 0)

	atomic.StoreInt64(&s.watching, 0)
	s.Events.Log("Servo", "sendNow", "Completed")
	return nil
}

// batch adds the given request into the batch lists. It returns true/false
// if the requests should be immediately served to the transport provider.
func (s *Servo) batch(query string, client Requestor) bool {
	s.Events.Log("Servo", "batch", "Started : Batching Query : %s", query)

	// If we have already sent the data that has been queued, then
	// reset all details accordinly and prepare to to batch new requests.
	if s.pendingQuery == nil {
		s.pendingTime = time.Now().Add(s.wait)
		s.pendingQuery = make(map[string]int)
	}

	// Add the pending query with the right index.
	index := len(s.pendingQuery)
	s.pendingQuery[query] = index

	if len(s.providers) > 1 && !time.Now().After(s.pendingTime) {
		s.Events.Log("Servo", "batch", "Completed")
		return false
	}

	s.Events.Log("Servo", "batch", "Completed")
	return true
}
