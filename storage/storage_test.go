package storage_test

import (
	"fmt"
	"testing"

	"gopkg.in/mgo.v2/bson"

	"github.com/ardanlabs/kit/tests"
	"github.com/influx6/coquery/storage"
)

//==============================================================================

var context = "testing"

//==============================================================================

// TestStorage validates the storage API.
func TestStorage(t *testing.T) {

	t.Logf("Given the need to CRUD a coquery.storage")
	{
		t.Logf("\tWhen giving a coquery.Store API")
		{

			so := storage.New("store_id")

			if err := so.Add(map[string]interface{}{"store_id": "30", "name": "alex"}); err != nil {
				t.Fatalf("\t%s\tShould have successfully stored the new rcord: %s", tests.Failed, err)
			}
			t.Logf("\t%s\tShould have successfully stored the new rcord.", tests.Success)

			if err := so.Add(map[string]interface{}{"store_id": "30", "address": bson.M{"state": "lagos"}}); err != nil {
				t.Fatalf("\t%s\tShould have successfully updated an existing record: %s", tests.Failed, err)
			}
			t.Logf("\t%s\tShould have successfully updated an existing record.", tests.Success)

			if !so.Has("30") {
				t.Fatalf("\t%s\tShould have successfully found record with id '30'", tests.Failed)
			}
			t.Logf("\t%s\tShould have successfully found record with id '30'", tests.Success)

			if !so.HasRecord(map[string]interface{}{"store_id": "30"}) {
				t.Fatalf("\t%s\tShould have successfully found record with id '30'", tests.Failed)
			}
			t.Logf("\t%s\tShould have successfully found record with id '30'", tests.Success)

			rec, err := so.Get("30")
			if err != nil {
				t.Fatalf("\t%s\tShould have successfully retrieve record with id '30'", tests.Failed)
			}
			t.Logf("\t%s\tShould have successfully retrieve record with id '30'", tests.Success)

			fmt.Printf("%+s", rec)
		}
	}
}

//==============================================================================
