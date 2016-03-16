package streams

import (
	"sync/atomic"

	"github.com/influx6/coquery"
	"github.com/influx6/faux/sumex"
)

//==============================================================================

// EventLog defines event logger that allows us to record events for a specific
// action that occured.
type EventLog interface {
	Log(context interface{}, name string, message string, data ...interface{})
	Error(context interface{}, name string, err error, message string, data ...interface{})
}

//==============================================================================

// ReadResponse uses the ResponseStream providing a decorating and allows its
// return values to be returned as a normal function call.
func ReadResponse(e EventLog, context interface{}, rid string, in sumex.Streams) (*coquery.Response, coquery.ResponseError) {
	e.Log(context, "ReadResponse", "Started : RequestID[%s]", rid)

	var res *coquery.Response
	var rerr coquery.ResponseError

	// Collect the coquery.Response and error channels
	rs, re := ResponseStream(e, context, rid, in)

	select {
	case res = <-rs:
	case rerr = <-re:
	}

	if rerr != nil {
		e.Error(context, "ReadResponse", rerr, "Completed")
		return nil, rerr
	}

	e.Log(context, "ReadResponse", "Completed")
	return res, nil
}

//==============================================================================

// ResponseStream returns a channel responds with a coquery.Response for a specific
// requests ID (RequestID())
func ResponseStream(e EventLog, context interface{}, rid string, in sumex.Streams) (<-chan *coquery.Response, <-chan coquery.ResponseError) {
	e.Log(context, "ResponseStream", "Started : RequestID[%s]", rid)

	out := make(chan *coquery.Response)
	outerr := make(chan coquery.ResponseError)

	// Create a receiver and pass the needed information into the out stream
	// if the RequestID() matches.
	rc, rcs := sumex.Receive(in)
	re, res := sumex.ReceiveError(in)

	go func() {
		e.Log(context, "ResponseStream.GoRoutine", "Start")
		defer rcs.Shutdown()
		defer res.Shutdown()

		var dead int64

		for {
			if atomic.LoadInt64(&dead) > 1 {
				e.Log(context, "ResponseStream.GoRoutine", "Completed")
				return
			}

			select {
			case ru, ok := <-rc:
				if !ok {
					rc = nil
					atomic.AddInt64(&dead, 1)
					continue
				}

				res, rok := ru.(*coquery.Response)
				if !rok {
					continue
				}

				if res.RequestID() != rid {
					continue
				}

				e.Log(context, "ResponseStream.GoRoutine", "Info : Received Response : ID[%s]", res.RequestID())
				out <- res
				e.Log(context, "ResponseStream.GoRoutine", "Completed")
				return

			case eu, ok := <-re:
				if !ok {
					re = nil
					atomic.AddInt64(&dead, 1)
					continue
				}

				res, rok := eu.(coquery.ResponseError)
				if !rok {
					continue
				}

				if res.RequestID() != rid {
					continue
				}

				e.Log(context, "ResponseStream.GoRoutine", "Info : Received Error Response : ID[%s]", res.RequestID())
				outerr <- res
				e.Log(context, "ResponseStream.GoRoutine", "Completed")
				return

			}
		}
	}()

	e.Log(context, "ResponseStream", "Completed")
	return out, outerr
}

//==============================================================================

// StreamOSHandler defines a global StreamOSHandler to be used by the
// internal streamos streamer.
var StreamOSHandler osHandler

// osHandler implements sumex.Proc.
type osHandler struct{}

// Do performs the action for the OSHandler processor.
func (os osHandler) Do(data interface{}, err error) (interface{}, error) {

	return nil, nil
}

// StreamOS provides a registery for registering different query processors
// that can be tied into a query
type StreamOS struct {
	sumex.Streams
	EventLog
	inport  sumex.Streams
	outport sumex.Streams
}

// New returns a new instance of StreamOS.
func New(e EventLog, workers int) *StreamOS {
	os := StreamOS{
		EventLog: e,
		Streams:  sumex.New(workers, StreamOSHandler),
		inport:   sumex.Identity(workers),
		outport:  sumex.Identity(workers),
	}

	return &os
}

// Stream overwrites the internal Stream(sumex.Stream) method connecting
// the provided stream into a internal steamer that allows us to process
// and return the appropriate results for a request.
func (s *StreamOS) Stream(hs sumex.Streams) sumex.Streams {
	s.inport.Stream(hs).Stream(s.outport)
	return hs
}

// Handle provides the implementation of the Document API that allows
// using the sumex api.
func (s *StreamOS) Handle(context interface{}, rq coquery.RecordRequest, rw coquery.ResponseWriter) {

}

//==============================================================================
