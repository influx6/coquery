package streams

import (
	"sync/atomic"

	"github.com/influx6/coquery"
	"github.com/influx6/faux/sumex"
)

// ReadResponse uses the ResponseStream providing a decorating and allows its
// return values to be returned as a normal function call.
func ReadResponse(rid string, in sumex.Streams) (*coquery.Response, *coquery.ResponseError) {
	var res *coquery.Response
	var rerr *coquery.ResponseError

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
func ResponseStream(rid string, in sumex.Streams) (<-chan *coquery.Response, <-chan *coquery.ResponseError) {
	out := make(chan *coquery.Response)
	outerr := make(chan *coquery.ResponseError)

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

				res, rok := ru.(*coquery.Response)
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

				res, rok := eu.(*coquery.ResponseError)
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
