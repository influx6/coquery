package storage_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/ardanlabs/kit/tests"
	"github.com/influx6/coquery/storage"
)

//==============================================================================

var context = "testing"

//==============================================================================

// BenchmarkStorageStore benchmarks the addition and deletion of records using
// the coquery.Storage.
func BenchmarkStorageDelete(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	so := storage.New("store_id")

	// Store N items.
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("%d", i)
		so.Add(map[string]interface{}{"store_id": key, "name": "alex"})
	}
}

// BenchmarkStorageStoreAndDelete benchmarks the addition and deletion of records using
// the coquery.Storage.
func BenchmarkStorageStoreAndDelete(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	so := storage.New("store_id")

	// Store N items.
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("%d", i)
		so.Add(map[string]interface{}{"store_id": key, "name": "alex"})
	}

	// Delete N items
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("%d", i)
		so.Delete(key)
	}
}

// BenchmarkStorage benchmarks the addition and deletion of records using
// the coquery.Storage.
func BenchmarkStorage(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	so := storage.New("store_id")

	// Store N items.
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("%d", i)
		so.Add(map[string]interface{}{"store_id": key, "name": "alex"})
	}

	// Delete N items
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("%d", i)
		so.Delete(key)
	}
}

// BenchmarkStorageWithRef benchmarks the addition and deletion of records using
// the coquery.Storage, and adding reference for the address.street key.
func BenchmarkStorageWithRef(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	so := storage.New("store_id")

	// Store N items.
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("%d", i)
		so.Add(map[string]interface{}{"store_id": key, "name": "alex"})
	}

	// Mod N items with new data and add refs
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("%d", i)
		so.AddRef(map[string]interface{}{"store_id": key, "address": map[string]interface{}{"state": fmt.Sprintf("lagos-%d", i), "country": "NG"}}, "address.state")
	}

	// Mod N items with new data and add refs
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("lagos-%d", i)
		so.GetByRef("address.state", key)
	}

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("%d", i)
		so.Delete(key)
	}
}

// BenchmarkStorage benchmarks the addition and deletion of records using
// the coquery.Storage with expiration turned on.
func BenchmarkExpirableStorage(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	so := storage.NewExpirable("store_id", 5*time.Second)

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("%d", i)
		so.Add(map[string]interface{}{"store_id": key, "name": "alex"})
	}

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("%d", i)
		so.Delete(key)
	}
}

// TestExpirationStorage validates the storage API.
func TestExpirationStorage(t *testing.T) {
	t.Logf("Given the need to CRUD a coquery.storage with expiration")
	{
		t.Logf("\tWhen giving a coquery.Store API")
		{

			so := storage.NewExpirable("store_id", 100*time.Millisecond)

			if err := so.Add(map[string]interface{}{"store_id": "30", "name": "alex"}); err != nil {
				t.Fatalf("\t%s\tShould have successfully stored the new rcord: %s", tests.Failed, err)
			}
			t.Logf("\t%s\tShould have successfully stored the new rcord.", tests.Success)

			_, err := so.Get("30")
			if err != nil {
				t.Fatalf("\t%s\tShould have successfully retrieve record with id '30'", tests.Failed)
			}
			t.Logf("\t%s\tShould have successfully retrieve record with id '30'", tests.Success)

			<-time.After(400 * time.Millisecond)

			_, err = so.Get("30")
			if err == nil {
				t.Fatalf("\t%s\tShould have failed to retrieve record with id '30'", tests.Failed)
			}
			t.Logf("\t%s\tShould have failed to retrieve record with id '30'", tests.Success)

		}
	}
}

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

			if err := so.ModRefBy(map[string]interface{}{"store_id": "30", "address": map[string]interface{}{"state": "lagos", "country": "NG"}}, "address.state", true); err != nil {
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

			rc, err := so.Get("30")
			if err != nil {
				t.Fatalf("\t%s\tShould have successfully retrieve record with id '30'", tests.Failed)
			}
			t.Logf("\t%s\tShould have successfully retrieve record with id '30'", tests.Success)

			_, err = so.GetByRef("address.state", "lagos")
			if err != nil {
				t.Fatalf("\t%s\tShould have successfully retrieve record by ref", tests.Failed)
			}
			t.Logf("\t%s\tShould have successfully retrieve record by ref", tests.Success)

			if _, ok := storage.PullKeys(rc, "address.state"); !ok {
				t.Fatalf("\t%s\tShould have successfully retrieve deep key[%s] record", tests.Failed, "address.state")
			}
			t.Logf("\t%s\tShould have successfully retrieve deep key[%s] record", tests.Success, "address.state")

			if err := so.RemoveByValue(map[string]interface{}{"store_id": "30", "address": map[string]interface{}{"state": "lagos"}}); err != nil {
				t.Fatalf("\t%s\tShould have successfully remove key with value on existing record: %s", tests.Failed, err)
			}
			t.Logf("\t%s\tShould have successfully remove key with value on existing record", tests.Success)

			if err := so.RemoveByKey(map[string]interface{}{"store_id": "30", "address": nil}); err != nil {
				t.Fatalf("\t%s\tShould have successfully remove key on existing record: %s", tests.Failed, err)
			}
			t.Logf("\t%s\tShould have successfully remove key on existing record.", tests.Success)

			so.Delete("30")
			// fmt.Printf("%+s\n", so)

			_, err = so.Get("30")
			if err == nil {
				t.Fatalf("\t%s\tShould have failed to retrieve record with id '30'", tests.Failed)
			}
			t.Logf("\t%s\tShould have failed to retrieve record with id '30'", tests.Success)

		}
	}
}

//==============================================================================
