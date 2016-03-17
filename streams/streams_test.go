package streams_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/ardanlabs/kit/tests"
	"github.com/influx6/coquery"
	mongo "github.com/influx6/coquery/documents/mongo/db"
	hmongo "github.com/influx6/coquery/documents/mongo/house"
	"github.com/influx6/coquery/storage"
	"github.com/influx6/coquery/streams"
	"github.com/influx6/faux/sumex"
)

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
		fmt.Printf("Error : %s : %s : %s : %q\n", context, name, fmt.Sprintf(message, data...), err.Error())
	}
}

//==============================================================================

type spyWriter struct {
	Out chan *coquery.Response
	Err chan coquery.ResponseError
}

// Write writes the appropriate response to the appropriate receiver.
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

// TestStreamOSWithEngine validates the operations of the new streamos system
// using the coquery.Engine.
func TestStreamOSWithEngine(t *testing.T) {
	t.Logf("Given the need to retrieve a record using FindProc operator")
	{

		t.Logf("\tWhen giving a mongo provider")
		{

			lg := &logg{}

			mo, merr := mongo.New(lg, mongo.Config{
				Host:     "127.0.0.1:27017",
				AuthDB:   "outcast",
				DB:       "outcast",
				User:     "box",
				Password: "box",
			})

			if merr != nil {
				t.Fatalf("\t%s\tShould have successfully connected to mongodb instance: %q", tests.Failed, merr)
			}
			t.Logf("\t%s\tShould have successfully connected to mongodb instance.", tests.Success)

			defer mo.Shutdown(context)

			store := storage.New("station_id")

			streamos := streams.New(streams.Config{
				Log:     lg,
				Wait:    4 * time.Minute,
				Workers: 4,
			})

			streamos.Stream(sumex.New(3, lg, &hmongo.FindProc{
				EventLog: lg,
				Mongo:    mo,
				Query:    mo,
				Store:    store,
			}))

			engine := coquery.New(lg)
			engine.Route(context, "docs").
				Document(context, "marine_metric_history", &coquery.BasicQueries{EventLog: lg}, streamos)

			//
			// request := &coquery.Find{
			// 	Doc:   "marine_metric_history",
			// 	RID:   "43D3UFZ6",
			// 	Key:   "station_id",
			// 	Value: "GMZ657",
			// }

			writer := &spyWriter{
				Out: make(chan *coquery.Response),
				Err: make(chan coquery.ResponseError),
			}

			qid := "432UFY"

			go engine.Serve(context, qid, "docs.marine_metric_history.find(station_id,GMZ657)", writer)

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

			if len(res.Data) < 1 {
				t.Fatalf("\t%s\tShould have received a valid data with request Id[%s]: %s", tests.Failed, qid, res.RequestID())
			}
			t.Logf("\t%s\tShould have received a valid data with request Id[%s]: %s", tests.Success, qid, res.RequestID())
		}
	}
}

//==============================================================================
