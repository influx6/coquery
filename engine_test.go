package coquery_test

import (
	"errors"
	"os"
	"testing"

	"github.com/ardanlabs/kit/log"
	"github.com/ardanlabs/kit/tests"
	"github.com/influx6/coquery"
	"github.com/influx6/coquery/storage"
)

//==============================================================================

var context = "testing"

//==============================================================================

func init() {
	log.Init(os.Stdout, func() int { return log.DEV }, log.Ldefault)
}

//==============================================================================

var events eventlog

// logg provides a concrete implementation of a logger.
type eventlog struct{}

// Log logs all standard log reports.
func (l eventlog) Log(context interface{}, name string, message string, data ...interface{}) {
	if testing.Verbose() {
		log.Dev(context, name, message, data...)
	}
}

// Error logs all error reports.
func (l eventlog) Error(context interface{}, name string, err error, message string, data ...interface{}) {
	if testing.Verbose() {
		log.Error(context, name, err, message, data...)
	}
}

//==============================================================================

type inMemory struct{}

// Handle replies the giving requests received from the backend.
func (im *inMemory) Handle(context interface{}, reqs coquery.RecordRequests, res coquery.ResponseWriter) {
	id := reqs[0].RequestID()

	if len(reqs) > 1 {
		res.Write(context, nil, &coquery.CoError{
			Rid:    id,
			Msg:    "Expected Only 1 request",
			IError: errors.New("Too Much Requests"),
		})
		return
	}

	res.Write(context, &coquery.Response{
		Req:  reqs[0],
		Data: []coquery.Parameter{{"id": 1, "greeting": "Hello World!"}},
	}, nil)
}

//==============================================================================

type spyWriter struct {
	Out chan *coquery.Response
	Err chan coquery.ResponseError
}

func (s *spyWriter) Write(context interface{}, rs *coquery.Response, re coquery.ResponseError) error {
	go func() {
		if rs == nil {
			s.Err <- re
			return
		}

		s.Out <- rs
	}()

	return nil
}

//==============================================================================

// TestCoEngine validates the operation provided by the coquery.Engine.
func TestCoEngine(t *testing.T) {
	t.Logf("Given the need to pass requests to a coquery.Engine")
	{

		eos := coquery.New(events, coquery.NewDiffs(events), storage.New("id"))

		eos.Route(context, "doc").
			Document(context, "greetings", &coquery.BasicQueries{EventLog: events}, &inMemory{})

		q1 := "doc.greetings.find(id,1)"
		t.Logf("\tWhen giving a query with one request: %q", q1)
		{

			writer := &spyWriter{
				Out: make(chan *coquery.Response),
				Err: make(chan coquery.ResponseError),
			}

			qid := "432UFY"

			eos.Serve(context, &coquery.RequestContext{
				RequestID: qid,
				Query:     []string{q1},
			}, writer)

			var res *coquery.Response
			var err coquery.ResponseError

			select {
			case res = <-writer.Out:
			case err = <-writer.Err:
			}

			if err != nil {
				t.Logf("Error: %+s", err)
				t.Fatalf("\t%s\tShould have successfull received a response: %s", tests.Failed, err.Error())
			}
			t.Logf("\t%s\tShould have successfull received a response.", tests.Success)

			if res.RequestID() != qid {
				t.Fatalf("\t%s\tShould have received response with request Id[%s]: %s", tests.Failed, qid, res.RequestID())
			}
			t.Logf("\t%s\tShould have received response with request Id[%s]", tests.Success, qid)

			first := res.Data[0]
			result := (first.Get("results").(coquery.Parameters))[0]

			if result.Get("greeting") != "Hello World!" {
				t.Logf("\t\t%+s\n", result)
				t.Fatalf("\t%s\tShould have successfull matched greetings as 'Hello Word!': %s", tests.Failed, result.Get("greeting"))
			}
			t.Logf("\t%s\tShould have successfull matched greetings as 'Hello Word!'", tests.Success)

		}

		q2 := "doc.greetings.find(id,1).collects(greeting)"
		t.Logf("\tWhen giving a query with more than one request: %q", q2)
		{

			writer := &spyWriter{
				Out: make(chan *coquery.Response),
				Err: make(chan coquery.ResponseError),
			}

			qid := "632UFY"

			eos.Serve(context, &coquery.RequestContext{
				RequestID: qid,
				Query:     []string{q2},
			}, writer)

			// var res *coquery.Response
			var err coquery.ResponseError

			select {
			case <-writer.Out:
			case err = <-writer.Err:
			}

			if err == nil {
				t.Fatalf("\t%s\tShould have failed when there was more than one reques.", tests.Failed)
			}
			t.Logf("\t%s\tShould have failed when there was more than one request: %q", tests.Success, err.Error())
		}
	}
}
