package coquery_test

import (
	"fmt"
	"testing"

	"github.com/ardanlabs/kit/tests"
	"github.com/influx6/coquery"
)

//==============================================================================

// BenchmarkDiff benchmarks the addition and deletion of records using
// the coquery.Storage.
func BenchmarkDiff(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	so := coquery.NewDiffs(events)

	var keys []string

	// Store N items.
	for i := 0; i < b.N; i++ {
		keys = append(keys, so.Put([]string{fmt.Sprintf("%d", i)}))
	}

	// Get all N items
	for _, key := range keys {
		so.Get(key)
	}

}

// TestDiff validates the coquery.Diff behaviour.
func TestDiff(t *testing.T) {

	t.Logf("Given the need to use the coquery.Diff interface")
	{
		t.Logf("\tWhen giving a coquery.Diff")
		{

			diff := coquery.NewDiffs(events)

			defer diff.Clear()

			id := diff.Put([]string{"1", "2", "3"})

			if !diff.Has(id) {
				t.Fatalf("\t%s\tShould find provided key in diff store", tests.Failed)
			}
			t.Logf("\t%s\tShould find provided key in diff store", tests.Success)

			if keys := diff.Get(id); len(keys) < 3 {
				t.Fatalf("\t%s\tShould find 3 keys in diff store for key[%s]", tests.Failed, id)
			}
			t.Logf("\t%s\tShould find 3 keys in diff store for key[%s]", tests.Success, id)

			diff.Put([]string{"1", "12", "31"})

			if pl := diff.PullFrom(id); len(pl) < 3 {
				t.Logf("\t\tPullFrom: %s\n", pl)
				t.Fatalf("\t%s\tShould find 3 record keys in diff list from Diff Key[%s]", tests.Failed, id)
			}
			t.Logf("\t%s\tShould find 3 record keys in diff list from Diff Key[%s]", tests.Success, id)

			changes := diff.AnalyzeWith(id, []string{"1"})
			if !changes["1"] {
				t.Logf("\t\tChanges: %+v\n", changes)
				t.Fatalf("\t%s\tShould expect changes for record 1 from ID[%s]", tests.Failed, id)
			}
			t.Logf("\t%s\tShould expect changes for record 1 from ID[%s]", tests.Success, id)

			changes = diff.Analyze([]string{"12", "31"})
			if !changes["12"] || !changes["31"] {
				t.Logf("\t\tChanges: %+v\n", changes)
				t.Fatalf("\t%s\tShould expect changes for record 12 and 31 from ID[%s]", tests.Failed, id)
			}
			t.Logf("\t%s\tShould expect changes for record 12 and 31 from ID[%s]", tests.Success, id)

			if df := diff.Diffs(); len(df) < 5 {
				t.Logf("\t\tDiff: %+v\n", df)
				t.Fatalf("\t%s\tShould expect 5 diff changes from store", tests.Failed)
			} else {
				t.Logf("\t%s\tShould expect 5 diff changes from store", tests.Success)
			}

		}
	}
}
