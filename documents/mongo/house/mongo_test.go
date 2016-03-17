package house_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ardanlabs/kit/tests"
	"github.com/influx6/coquery"
	mongo "github.com/influx6/coquery/documents/mongo/db"
	hmongo "github.com/influx6/coquery/documents/mongo/house"
	"github.com/influx6/coquery/streams"
	"github.com/influx6/faux/sumex"
)

//==============================================================================

var context = "testing"

//==============================================================================

// logg provides a concrete implementation of a logger.
type logg struct{}

// Log logs all standard log reports.
func (l *logg) Log(context interface{}, name string, message string, data ...interface{}) {
	if testing.Verbose() {
		fmt.Printf("Log : %s : %s : %s", context, name, fmt.Sprintf(message, data...))
	}
}

// Error logs all error reports.
func (l *logg) Error(context interface{}, name string, err error, message string, data ...interface{}) {
	if testing.Verbose() {
		fmt.Printf("Error : %s : %s : %s", context, name, fmt.Sprintf(message, data...))
	}
}

//==============================================================================

// TestFindProc validates the operation provided by the FindProc operator.
func TestFindProc(t *testing.T) {
	t.Logf("Given the need to retrieve a record using FindProc operator")
	{

		t.Logf("\tWhen giving a mongo provider")
		{

			var lg coquery.EventLog
			lg = &logg{}

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

			request := coquery.Find{
				Doc:   "marine_metric_history",
				RID:   "43D3UFZ6",
				Key:   "station_id",
				Value: "GMZ657",
			}

			finder := &hmongo.FindProc{
				EventLog: lg,
				Mongo:    mo,
				Query:    mo,
			}

			if finder.Name() != "find" {
				t.Fatalf("\t%s\tShould have the giving name %q for this operator: %q", tests.Failed, "find", finder.Name())
			}
			t.Logf("\t%s\tShould have the giving name %q for this operator: %q", tests.Success, "find", finder.Name())

			err1 := errors.New("Invalid Operation")

			if _, err := finder.Do(&request, err1); err != err1 {
				t.Fatalf("\t%s\tShould have returned the error passed as second argument: %q", tests.Failed, err1)
			}
			t.Logf("\t%s\tShould have returned the error passed as second argument,", tests.Success)

			_, err := finder.Do(&request, nil)
			if err != nil {
				t.Fatalf("\t%s\tShould have retrived record without error: %q", tests.Failed, err)
			}
			t.Logf("\t%s\tShould have retrived record without error.", tests.Success)

		}
	}
}

//==============================================================================

// TestFindProcStream validates the operation provided by the FindProc operator
// when using the streaming interface.
func TestFindProcStream(t *testing.T) {
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

			request := &coquery.Find{
				Doc:   "marine_metric_history",
				RID:   "43D3UFZ6",
				Key:   "station_id",
				Value: "GMZ657",
			}

			finder := &hmongo.FindProc{
				EventLog: lg,
				Mongo:    mo,
				Query:    mo,
			}

			findStream := sumex.New(3, finder)

			findStream.Inject(request)

			res, err := streams.ReadResponse(lg, context, 1*time.Minute, request.RID, findStream)
			if err != nil {
				t.Fatalf("\t%s\tShould have returned the error passed as second argument: %q", tests.Failed, err)
			}
			t.Logf("\t%s\tShould have returned the error passed as second argument,", tests.Success)

			if res.RequestID() != request.RequestID() {
				t.Fatalf("\t%s\tShould have recieved a reply for request ID %s", tests.Failed, request.RID)
			}
			t.Logf("\t%s\tShould have recieved a reply for request ID %s", tests.Success, request.RID)

		}
	}
}

//==============================================================================
