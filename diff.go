package coquery

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/pborman/uuid"
)

// Diffs defines an interface for storing store diffs for the coquery system
// that lets us know which records had changed during a request.
type Diffs interface {
	Clear()
	Keys() []string
	Has(id string) bool
	Put([]string) string
	Get(id string) []string
	Diffs() map[string]struct{}
	PullFrom(id string) []string
	Analyze([]string) map[string]bool
	AnalyzeWith(string, []string) map[string]bool
}

//==============================================================================

// Diff provides a diff object which stores a diff and the timestamp it was
// added.
type Diff struct {
	Key     string
	Diff    []string
	Time    time.Time
	expired bool
}

// DiffStore provides a inmemory diff storage system for coquery, it stores and
// clears out all its old diff lists after a giving period/duration, only keeping
// diffs information within the valid period of lifetime.
// Implements the Diffs interface.
type DiffStore struct {
	EventLog
	lifeTime time.Time
	maxAge   time.Duration
	everyms  time.Duration
	dl       sync.RWMutex
	diffs    []*Diff
	keys     map[string]int
}

// NewDiffs returns a new instance of a DiffStore which does not expire it
// records.
func NewDiffs(el EventLog) *DiffStore {
	diff := DiffStore{
		EventLog: el,
		keys:     make(map[string]int),
	}

	return &diff
}

// NewExpiringDiffs returns a new instance of a DiffStore which expires its
// records after a specific lifetime duration.
// You can set the maximum age for records and the intervals you want to
// check for expiration of records.
func NewExpiringDiffs(el EventLog, maxAge time.Duration) *DiffStore {
	diff := DiffStore{
		EventLog: el,
		maxAge:   maxAge,
		lifeTime: time.Now().Add(maxAge),
		keys:     make(map[string]int),
	}

	return &diff
}

// Analyze analyzes the giving record keys with its internal records and returns
// a truth map to indicate which record has changed or not.
func (diff *DiffStore) Analyze(keys []string) map[string]bool {
	diff.Log("DiffStore", "Analyze", "Started : Records %s", keys)
	diff.clean()

	truths := make(map[string]bool)
	changes := diff.Diffs()
	diff.Log("DiffStore", "Analyze", "Info : Diffs %s", changes)

	for _, key := range keys {
		if _, ok := changes[key]; ok {
			truths[key] = true
			continue
		}
		truths[key] = false
	}

	diff.Log("DiffStore", "Analyze", "Completed")
	return truths
}

// AnalyzeWith uses the giving last diff Id to create a lists of changes and
// returns a truth table map (of key:change_status), to indicate which record
// key indeed was registed to have changed.
func (diff *DiffStore) AnalyzeWith(lastID string, keys []string) map[string]bool {
	diff.Log("DiffStore", "AnalyzeWith", "Started : Last Diff Key[%s] : Records %s", lastID, keys)
	diff.clean()

	truths := make(map[string]bool)

	// Set all keys as false.
	for _, k := range keys {
		truths[k] = false
	}

	changes := diff.PullFrom(lastID)
	if len(changes) < 1 {
		return truths
	}

	diff.Log("DiffStore", "AnalyzeWith", "Info : Diffs %s", changes)

	// Run throug the changes and set true for the changed keys.
	for _, id := range changes {
		if _, ok := truths[id]; ok {
			truths[id] = true
		}
	}

	diff.Log("DiffStore", "AnalyzeWith", "Completed")
	return truths
}

// PullFrom pulls all the changes that has occured until the last record diff
// returning all the records a single list. It removes any duplicates.
// If no such key exists, it returns an empty string.
func (diff *DiffStore) PullFrom(id string) []string {
	diff.Log("DiffStore", "PullFrom", "Started : Last Record ID[%s]", id)
	diff.clean()

	diff.dl.RLock()
	defer diff.dl.RUnlock()

	index, ok := diff.keys[id]
	if !ok {
		diff.Error("DiffStore", "PullFrom", ErrRecordNotFound, "Completed")
		return nil
	}

	dlen := len(diff.diffs)

	// If this is the last diff then return nil
	if index+1 >= dlen {
		return nil
	}

	found := make(map[string]struct{})

	// Collect all diffs from this point and merge the information.
	for i := index + 1; i < dlen; i++ {
		rec := diff.diffs[i]
		for _, id := range rec.Diff {
			found[id] = struct{}{}
		}
	}

	var records []string
	for key := range found {
		records = append(records, key)
	}

	diff.Log("DiffStore", "PullFrom", "Completed")
	return records
}

// Diffs returns a map of all changed record keys.
func (diff *DiffStore) Diffs() map[string]struct{} {
	diff.Log("DiffStore", "Keys", "Started")
	diff.clean()

	diff.dl.RLock()
	defer diff.dl.RUnlock()

	changes := make(map[string]struct{})

	for _, rec := range diff.diffs {
		if rec.expired {
			continue
		}

		for _, key := range rec.Diff {
			changes[key] = struct{}{}
		}
	}

	diff.Log("DiffStore", "Keys", "Completed")
	return changes
}

// Keys returns a lists of records keys within the store.
func (diff *DiffStore) Keys() []string {
	diff.Log("DiffStore", "Keys", "Started")
	diff.clean()

	diff.dl.RLock()
	defer diff.dl.RUnlock()

	var keys []string

	for _, rec := range diff.diffs {
		if rec.expired {
			continue
		}
		keys = append(keys, rec.Key)
	}

	diff.Log("DiffStore", "Keys", "Completed")
	return keys
}

// ErrRecordNotFound is returned when a giving record key has no associted record.
var ErrRecordNotFound = errors.New("Record Not Found")

// Get retrieves a key diff record if it keys.
func (diff *DiffStore) Get(record string) []string {
	diff.Log("DiffStore", "Get", "Started : Retrieve Record : Key[%s]", record)
	diff.clean()

	diff.dl.RLock()
	defer diff.dl.RUnlock()

	rec, ok := diff.keys[record]
	if !ok {
		diff.Error("DiffStore", "Get", ErrRecordNotFound, "Completed")
		return nil
	}

	dl := diff.diffs[rec]

	diff.Log("DiffStore", "Get", "Completed")
	return dl.Diff
}

// Put stores a list of diffs and returns the associated key for this diff.
func (diff *DiffStore) Put(record []string) string {
	diff.Log("DiffStore", "Put", "Started : Adding New Record : %s", fmt.Sprintf("%+v", record))
	diff.clean()

	key := uuid.New()

	if len(record) < 1 {
		diff.Error("DiffStore", "Put", fmt.Errorf("Empty Record"), "Completed")
		return key
	}

	diff.dl.Lock()
	defer diff.dl.Unlock()

	df := &Diff{Key: key, Diff: record, Time: time.Now()}
	diff.keys[key] = len(diff.diffs)
	diff.diffs = append(diff.diffs, df)

	diff.Log("DiffStore", "Put", "Completed")
	return key
}

// Clear clears out all record stores within the diff map.
func (diff *DiffStore) Clear() {
	diff.Log("DiffStore", "Clear", "Started")
	diff.dl.Lock()
	defer diff.dl.Unlock()
	diff.diffs = nil
	diff.keys = make(map[string]int)
	diff.Log("DiffStore", "Clear", "Completed")
}

// Has returns true/false if the record key exists within the store.
func (diff *DiffStore) Has(key string) bool {
	diff.Log("DiffStore", "Has", "Started : Checking Key[%s]", key)
	diff.dl.RLock()
	defer diff.dl.RUnlock()

	_, ok := diff.keys[key]
	if !ok {
		diff.Error("DiffStore", "Has", ErrRecordNotFound, "Completed")
		return false
	}

	diff.Log("DiffStore", "Has", "Completed")
	return true
}

// clean removes all expired records from the lists.
func (diff *DiffStore) clean() {
	diff.Log("DiffStore", "clean", "Started")

	if diff.maxAge == 0 {
		diff.Log("DiffStore", "clean", "Info : No Checks")
		diff.Log("DiffStore", "clean", "Completed")
		return
	}

	age := time.Now()

	diff.dl.Lock()
	defer diff.dl.Unlock()

	for key, ind := range diff.keys {
		rec := diff.diffs[ind]
		if age.Sub(rec.Time) < diff.maxAge {
			continue
		}

		rec.expired = true

		delete(diff.keys, key)
		diff.diffs = append(diff.diffs[:ind], diff.diffs[ind+1:]...)
	}

	diff.Log("DiffStore", "clean", "Completed")
}

//==============================================================================
