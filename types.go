package coquery

import (
	"sync/atomic"

	"github.com/influx6/faux/sumex"
)

//==============================================================================

// Parameter defines the basic data type for all data received from the
// providers.
type Parameter map[string]interface{}

// Parameters defines a lists of Parameter types.
type Parameters []Parameter

// Response provides a response struct for replies to coquery requests.
type Response struct {
	Req  RecordRequest `json:"-" bson:"-"`
	RID  string        `json:"rid" bson:"rid"`
	Data Parameters    `json:"reply" bson:"reply"`
}

// ReadResponse uses the ResponseStream providing a decorating and allows its
// return values to be returned as a normal function call.
func ReadResponse(rid string, in sumex.Streams) (*Response, *ResponseError) {
	var res *Response
	var rerr *ResponseError

	// Collect the response and error channels
	rs, re := ResponseStream(rid, in)

	select {
	case res = <-rs:
	case rerr = <-re:
	}

	if rerr != nil {
		return nil, rerr
	}

	return res, nil
}

// ResponseStream returns a channel responds with a response for a specific
// requests ID (RID)
func ResponseStream(rid string, in sumex.Streams) (<-chan *Response, <-chan *ResponseError) {
	out := make(chan *Response)
	outerr := make(chan *ResponseError)

	// Create a receiver and pass the needed information into the out stream
	// if the RID matches.
	rc, rcs := sumex.Receive(in)
	re, res := sumex.ReceiveError(in)

	go func() {
		defer rcs.Shutdown()
		defer res.Shutdown()

		var dead int64

		for {
			if atomic.LoadInt64(&dead) > 1 {
				return
			}

			select {
			case ru, ok := <-rc:
				if !ok {
					rc = nil
					atomic.AddInt64(&dead, 1)
					continue
				}

				res, rok := ru.(*Response)
				if !rok {
					continue
				}

				if res.RID != rid {
					continue
				}

				out <- res
				return

			case eu, ok := <-re:
				if !ok {
					re = nil
					atomic.AddInt64(&dead, 1)
					continue
				}

				res, rok := eu.(*ResponseError)
				if !rok {
					continue
				}

				if res.RID != rid {
					continue
				}

				outerr <- res
				return

			}
		}
	}()

	return out, outerr
}

//==============================================================================

// ResponseError provides a custom error message for requests types.
type ResponseError struct {
	RID     string `json:"rid" bson:"rid"`
	Message string `json:"message" bson:"message"`
	IError  error  `json:"error" bson:"error"`
}

// Error returns the error message for this response error.
func (r ResponseError) Error() string {
	if r.IError != nil {
		return r.RID + " : " + r.Message + " : " + r.IError.Error()
	}

	return r.RID + " : " + r.Message
}

//==============================================================================

// RecordRequest defines a base type for the supported request types
type RecordRequest interface {
	Name() string
}

// RecordReplyChannel defines a channel type that provides the response for a
// coquery RecordRequest.
type RecordReplyChannel chan Response

//==============================================================================

// Find defines a record retrieve request based on the KV query.
type Find struct {
	Doc   string      `json:"doc" bson:"doc"`
	RID   string      `json:"rid" bson:"rid"`
	Key   string      `json:"key" bson:"key"`
	Value interface{} `json:"value" bson:"value"`
	// Next  []string // the next query to be executed after this query
}

// Name returns the name for the giving request type.
func (f Find) Name() string {
	return "find"
}

//==============================================================================

// Collect retrieves specific keyed items from the coquery stores.
type Collect struct {
	RID  string   `json:"rid" bson:"rid"`
	Keys []string `json:"keys" bson:"keys"`
	// Next  []string // the next query to be executed after this query
}

// Name returns the name for the giving request type.
func (f Collect) Name() string {
	return "collect"
}

//==============================================================================

// Mutate provides json data to be saved/augmented into a new version of the
// current document.
type Mutate struct {
	RID  string `json:"rid" bson:"rid"`
	Data []byte `json:"mutate" bson:"mutate"`
	// Next  []string
}

// Name returns the name for the giving request type.
func (f Mutate) Name() string {
	return "mutate"
}

//==============================================================================
