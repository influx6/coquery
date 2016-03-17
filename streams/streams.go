package streams

import (
	"errors"
	"sync/atomic"
	"time"

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
func ReadResponse(e EventLog, context interface{}, mx time.Duration, rid string, in sumex.Streams) (*coquery.Response, coquery.ResponseError) {
	e.Log(context, "ReadResponse", "Started : RequestID[%s]", rid)

	var res *coquery.Response
	var rerr coquery.ResponseError

	// Collect the coquery.Response and error channels
	rs, re := ResponseStream(e, context, mx, rid, in)

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

// ErrRequestTimout is returned when the maximum wait time for a requests
// completion timeout.
var ErrRequestTimout = errors.New("Request Timed Out")

// ResponseStream returns a channel responds with a coquery.Response for a specific
// requests ID (RequestID())
func ResponseStream(e EventLog, context interface{}, maxWait time.Duration, rid string, in sumex.Streams) (<-chan *coquery.Response, <-chan coquery.ResponseError) {
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

			case <-time.After(maxWait):
				err := coquery.CoError{Rid: rid, Msg: "Timeout", IError: ErrRequestTimout}
				e.Log(context, "ResponseStream.GoRoutine", "Info : Received Error Response : ID[%s]", rid)
				outerr <- &err
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
	return data, err
}

// Config provies a configuration for the a new StreamOS.
type Config struct {
	Log     EventLog
	Wait    time.Duration
	Workers int
}

// StreamOS provides a registery for registering different query processors
// that can be tied into a query
type StreamOS struct {
	*Config
	sumex.Streams
	inport  sumex.Streams
	outport sumex.Streams
}

// New returns a new instance of StreamOS, setting the wait time and worker
// counts.
func New(c Config) *StreamOS {
	if c.Wait == 0 {
		c.Wait = time.Duration(2 * time.Minute)
	}

	os := StreamOS{
		Config:  &c,
		Streams: sumex.New(c.Workers, StreamOSHandler),
		inport:  sumex.Identity(c.Workers),
		outport: sumex.Identity(c.Workers),
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
func (s *StreamOS) Handle(context interface{}, rqs coquery.RecordRequests, rw coquery.ResponseWriter) {
	s.Log.Log(context, "Handle", "Started : Recieved New Requests : Total[%d]", len(rqs))

	for index, request := range rqs {

		// Continuesly send each request into the stream of processor and await
		// a response from the processor.
		s.inport.Inject(request)

		wait := s.Wait

		// If we have a request that wants to be smart about its wait period, then
		// that's fine and within its rights, allow it.
		// NOTE: We won't stop you from being stupid. :)
		if ts, ok := request.(coquery.RecordTimedRequest); ok {
			wait = ts.Wait()
		}

		s.Log.Log(context, "Handle", "Info : Request[%s] : Type[%s] : Wait Period [%s]", request.RequestID(), request.RequestName(), wait)
		// Read the response for this requests and if possible its error.
		res, err := ReadResponse(s.Config.Log, context, wait, request.RequestID(), s.outport)
		if err != nil {

			// If we failed, write the response and break this loop, we have no
			// business continuing.
			// TODO: Do we want to allow continous agumented queries?
			// I would not advice this though. We should make our queries idempotent,
			// they should only serve one request call for a client.
			rw.Write(context, nil, err)

			// Write out this error, so anyone listening and can see.
			// TODO: do we want to do this or wait until the end.
			s.InjectError(err)
			s.Log.Error(context, "Handle", err, "Completed : Request[%s] : Type[%s]", request.RequestID(), request.RequestName())
			return
		}

		// If we passed, send out the response to anyone who cares.
		// Are we at the last request, if so, write it to the ResponseWriter, else
		// continue until the last one.
		if index >= len(rqs)-1 {
			rw.Write(context, res, nil)
		}

		s.Inject(res)
		s.Log.Log(context, "Handle", "Completed : Request[%s] : Type[%s] : Status[%s]", request.RequestID(), request.RequestName(), "Ok")
	}

	s.Log.Log(context, "Handle", "Completed")
}

//==============================================================================
