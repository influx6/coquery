package coquery_test

import (
	"fmt"
	"testing"

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

			// diff := coquery.NewDiffs(events)
		}
	}
}
