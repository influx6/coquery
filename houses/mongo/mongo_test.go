package mongo_test

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/ardanlabs/kit/log"
	"github.com/ardanlabs/kit/tests"
	"github.com/influx6/coquery"
	"github.com/influx6/coquery/db/mongo"
	hmongo "github.com/influx6/coquery/houses/mongo"
	"github.com/influx6/coquery/streams"
	"github.com/influx6/faux/sumex"
)

//==============================================================================

var context = "testing"

//==============================================================================

var logbuff bytes.Buffer
var logg = log.New(&logbuff, func() int { return log.DEV }, log.Ldefault)

//==============================================================================

// TestFindProc validates the operation provided by the FindProc operator.
func TestFindProc(t *testing.T) {
	logbuff.Reset()
	defer fmt.Printf(logbuff.String())

	t.Logf("Given the need to retrieve a record using FindProc operator")
	{

		t.Logf("\tWhen giving a mongo provider")
		{

			mo, merr := mongo.New(logg, mongo.Config{
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

			request := coquery.Find{
				Doc:   "marine_metric_history",
				RID:   "43D3UFZ6",
				Key:   "station_id",
				Value: "GMZ657",
			}

			finder := &hmongo.FindProc{
				Log:   logg,
				Mongo: mo,
				Query: mo,
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
	logbuff.Reset()
	defer fmt.Printf(logbuff.String())

	t.Logf("Given the need to retrieve a record using FindProc operator")
	{

		t.Logf("\tWhen giving a mongo provider")
		{

			mo, merr := mongo.New(logg, mongo.Config{
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

			request := &coquery.Find{
				Doc:   "marine_metric_history",
				RID:   "43D3UFZ6",
				Key:   "station_id",
				Value: "GMZ657",
			}

			finder := &hmongo.FindProc{
				Log:   logg,
				Mongo: mo,
				Query: mo,
			}

			findStream := sumex.New(3, finder)

			findStream.Inject(request)

			res, err := streams.ReadResponse(request.RID, findStream)
			if err != nil {
				t.Fatalf("\t%s\tShould have returned the error passed as second argument: %q", tests.Failed, err)
			}
			t.Logf("\t%s\tShould have returned the error passed as second argument,", tests.Success)

			if res.RID != request.RID {
				t.Fatalf("\t%s\tShould have recieved a reply for request ID %s", tests.Failed, request.RID)
			}
			t.Logf("\t%s\tShould have recieved a reply for request ID %s", tests.Success, request.RID)

		}
	}
}

//==============================================================================
