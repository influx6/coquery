package coquery_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/ardanlabs/kit/tests"
	"github.com/influx6/coquery"
)

//==============================================================================

var context = "testing"

//==============================================================================

// logg provides a concrete implementation of a logger.
type logg struct{}

// Log logs all standard log reports.
func (l *logg) Log(context interface{}, name string, message string, data ...interface{}) {
	if testing.Verbose() {
		fmt.Printf("Log : %s : %s : %s\n", context, name, fmt.Sprintf(message, data...))
	}
}

// Error logs all error reports.
func (l *logg) Error(context interface{}, name string, err error, message string, data ...interface{}) {
	if testing.Verbose() {
		fmt.Printf("Error : %s : %s : %s\n", context, name, fmt.Sprintf(message, data...))
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
		RID:  id,
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

		log := &logg{}
		eos := coquery.New(log)

		eos.Route(context, "doc").
			Document(context, "greetings", &coquery.BasicQueries{EventLog: log}, &inMemory{})

		q1 := "doc.greetings.find(id,1)"
		t.Logf("\tWhen giving a query request: %q", q1)
		{

			writer := &spyWriter{
				Out: make(chan *coquery.Response),
				Err: make(chan coquery.ResponseError),
			}

			qid := "432UFY"

			eos.Serve(context, qid, q1, writer)

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

			if first.Get("greeting") != "Hello World!" {
				t.Fatalf("\t%s\tShould have successfull matched greetings as 'Hello Word!': %s", tests.Failed, first.Get("greeting"))
			}
			t.Logf("\t%s\tShould have successfull matched greetings as 'Hello Word!'", tests.Success)

		}
	}
}
