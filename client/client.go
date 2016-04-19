package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/influx6/coquery/data"
	"github.com/influx6/faux/utils"
)

//==============================================================================

// ServeTransport defines an interface for requests transport, which allows us
// build custom transports based on different low-level systems(HTTP,Websocket).
type ServeTransport interface {
	Do(endpoint string, body io.Reader) (data.ResponsePack, error)
}

//==============================================================================

// Events defines event logger that allows us to record events for a specific
// action that occured.
type Events interface {
	Log(context interface{}, name string, message string, data ...interface{})
	Error(context interface{}, name string, err error, message string, data ...interface{})
}

//==============================================================================

// Handler defines a handler type for receving a per data response.
type Handler func(error, data.ResponseMeta, data.Parameters)

// Handlers defines a lists of Handler functions attributed to a query.
type Handlers struct {
	Qry string
	hl  []Handler
}

// Emit applies its arguments to its giving handlers.
func (h Handlers) Emit(err error, m data.ResponseMeta, d data.Parameters) {
	for _, hl := range h.hl {
		hl(err, m, d)
	}
}

//==============================================================================

// UpdateTrigger provides a struct that allows registering for updates for
// specific records related to a specific query, when this records keys
// come through as deltas reports, this trigger function is called for the client
// to acct accordinly.
type UpdateTrigger struct {
	qry     string
	hl      sync.RWMutex
	keys    map[interface{}]bool
	touched map[interface{}]bool
	trigger func()
}

// Update checks the giving deltas against its internal keys and triggers the
// internal callback if any of its keys match.
func (h *UpdateTrigger) Update(deltas []string) {
	tlen := len(h.touched)
	klen := len(h.keys)

	if tlen == klen {
		return
	}

	defer func() {
		h.touched = make(map[interface{}]bool)
	}()

	h.hl.RLock()
	defer h.hl.RUnlock()

	for _, key := range deltas {
		if h.keys[key] {
			h.trigger()
			return
		}
	}
}

// UpdateKeys updates the keys within the update trigger that matches.
func (h *UpdateTrigger) UpdateKeys(meta data.ResponseMeta, da data.ResponsePack) {
	h.hl.Lock()
	defer h.hl.Unlock()

	// Collect all record keys and store them for so we can review the delta
	// lists incase we need to make requests for updates
	for _, record := range da.Results {
		key := record[meta.RecordKey]
		h.keys[key] = true
		h.keys[key] = true
	}
}

//==============================================================================

// Server provides a central request manager for different query requests and
// subscriptions.
type Server interface {
	Request(query string, hl Handler) error
	Updates(query string, hl func())
	Serve() error
}

//==============================================================================

// Servo defines a concrete implementation of the Server interface.
// It handles scheduling query requests and providing the appropriate
// Response parameter for each requster.
type Servo struct {
	Events
	addr        string
	uuid        string
	pendingTime time.Time
	wait        time.Duration
	transport   ServeTransport
	rl          sync.RWMutex
	requests    []Handlers
	ul          sync.RWMutex
	updates     []*UpdateTrigger
	lastPack    data.ResponsePack
	locked      int64
}

// NewServo creates a new Servo instance. It takes a coquery server address
// and the maximum time to wait to allow requests batching and the underline
// transport to be used to make such requests with.
func NewServo(events Events, addr string, wait time.Duration, transport ServeTransport) *Servo {
	if wait == 0 {
		wait = 500 * time.Millisecond
	}

	svo := Servo{
		locked:      1,
		addr:        addr,
		wait:        wait,
		Events:      events,
		pendingTime: time.Now().Add(wait),
		transport:   transport,
		uuid:        utils.UUID(),
		requests:    make([]Handlers, 0),
		updates:     make([]*UpdateTrigger, 0),
	}

	return &svo
}

// Updates stacks a query requests to the api and calls the given
// handler with the response for that query when returned.
func (s *Servo) Updates(query string, hl func()) error {
	s.Events.Log("Servo", "Request", "Updates : Query[%s]", query)

	s.ul.RLock()
	defer s.ul.RUnlock()

	s.updates = append(s.updates, &UpdateTrigger{
		qry:     query,
		trigger: hl,
		keys:    make(map[interface{}]bool),
	})

	s.Events.Log("Servo", "Updates", "Completed")
	return nil
}

// Request stacks a query requests to the api and calls the given
// handler with the response for that query when returned.
func (s *Servo) Request(query string, hl Handler) error {
	s.Events.Log("Servo", "Request", "Started : Query[%s]", query)

	if atomic.LoadInt64(&s.locked) < 1 {
		atomic.StoreInt64(&s.locked, 1)
	}

	s.rl.RLock()
	defer s.rl.RUnlock()

	new := true

	for _, hls := range s.requests {
		if hls.Qry == query {
			hls.hl = append(hls.hl, hl)
			new = false
			break
		}
	}

	if new {
		s.requests = append(s.requests, Handlers{
			Qry: query,
			hl:  []Handler{hl},
		})
	}

	go func() {
		<-time.After(s.wait)
		if atomic.LoadInt64(&s.locked) > 0 {
			s.Serve()
		}
	}()

	s.Events.Log("Servo", "Request", "Completed")
	return s.Serve()
}

// Serve process the requests queries which will be batched and sent within a
// specified timing these allows us to batch and send as much request over
// specific period of times without wasting bandwidth.
func (s *Servo) Serve() error {
	s.Events.Log("Servo", "serve", "Started")

	if time.Now().Before(s.pendingTime) {
		s.Events.Log("Servo", "serve", "Completed")
		return nil
	}

	atomic.StoreInt64(&s.locked, 0)

	// Collect the pending requests and reset the pendingTime.
	s.rl.Lock()
	pendings := s.requests
	s.requests = nil
	s.rl.Unlock()

	defer func() {
		s.pendingTime = time.Now().Add(s.wait)
	}()

	diff := s.lastPack.DeltaID

	var mdata data.RequestContext
	mdata.RequestID = s.uuid
	mdata.Diffs = true
	mdata.DiffTag = diff

	for _, hl := range pendings {
		mdata.Queries = append(mdata.Queries, hl.Qry)
	}

	var buf bytes.Buffer
	var meta data.ResponseMeta
	var reply data.ResponsePack

	// Attemp to encode the request data as json else return error.
	if err := json.NewEncoder(&buf).Encode(&mdata); err != nil {
		for _, hl := range pendings {
			hl.Emit(err, meta, reply.Results)
		}

		s.Events.Error("Servo", "serve", err, "Completed")
		return err
	}

	var err error

	// Deliver body to the transport layer.
	reply, err = s.transport.Do(s.addr, &buf)
	if err != nil {
		for _, hl := range pendings {
			hl.Emit(err, meta, reply.Results)
		}
		s.Events.Error("Servo", "serve", err, "Completed")
		return err
	}

	meta.DeltaID = reply.DeltaID
	meta.RecordKey = reply.RecordKey
	meta.RequestID = reply.RequestID

	s.lastPack = reply

	if len(reply.Results) < len(pendings) {
		err := errors.New("Inadequate Response Length")
		s.Events.Error("Servo", "sendNow", err, "Completed")
		return err
	}

	for ind, qry := range pendings {
		pending := pendings[ind]

		if !reply.Batched {
			pending.Emit(nil, meta, reply.Results)

			for _, upd := range s.updates {
				if upd.qry != pending.Qry {
					continue
				}
				upd.UpdateKeys(meta, reply)
			}

			continue
		}

		localReply := reply
		localReply.Results = nil

		rez := reply.Results[ind]

		if failed, ok := rez["QueryFailed"].(bool); ok && failed {
			failedErr := fmt.Errorf("Message{%s} - Error{%s}", rez["Message"], rez["Error"])
			s.Events.Error("Servo", "sendNow", failedErr, "Info : Query [%s] : Failed", qry)
			pending.Emit(failedErr, meta, localReply.Results)
			continue
		}

		mrdos := rez["data"]

		if mrdos == nil {
			pending.Emit(nil, meta, localReply.Results)
			continue
		}

		mrd := mrdos.([]interface{})

		for _, prec := range mrd {
			pmrec := prec.(map[string]interface{})
			localReply.Results = append(localReply.Results, data.Parameter(pmrec))
		}

		pending.Emit(nil, meta, localReply.Results)

		for _, upd := range s.updates {
			if upd.qry != pending.Qry {
				continue
			}
			upd.UpdateKeys(meta, localReply)
		}
	}

	// Check if last delta tag is same as the new recieved reply, if it is not
	// then proceed update check cycle.
	if reply.DeltaID != diff && len(reply.Deltas) > 0 {
		// Check the providers who were not queue if they need to be updated and
		// schedule updates accordingly.
		s.ul.RLock()

		for _, upd := range s.updates {
			upd.Update(reply.Deltas)
		}

		s.ul.RUnlock()
	}

	s.Events.Log("Servo", "serve", "Completed")
	return nil
}
