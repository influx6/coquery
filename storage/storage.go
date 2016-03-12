package storage

import (
	"errors"
	"reflect"
	"sync"

	"gopkg.in/mgo.v2/bson"
)

//==============================================================================

// Record define a lists of Record items within the understore.
type Record map[string]interface{}

// Store defines a interface that provides necessary storage methods
type Store interface {
	ClearTainted()
	ClearDeleted()

	Has(string) bool
	HasRecord(Record) bool

	Add(Record) error
	Remove(Record) error

	Get(string) (Record, error)

	TaintedRecords() []string
	DeletedRecords() []string
}

//==============================================================================

// under provides an adequate means of storing full large scale json/json graph
// documents which allows us to cache.
type under struct {
	key     string
	rl      sync.RWMutex
	Records map[string]Record
	tainted map[string]bool
	deleted map[string]bool
}

// New returns a new instance of the under store.
func New(RecordKey string) Store {
	un := under{
		key:     RecordKey,
		Records: make(map[string]Record),
		tainted: make(map[string]bool),
		deleted: make(map[string]bool),
	}

	return &un
}

//==============================================================================

// ErrNoKeyInRecord is returned when the Record lacks the wanted key.
var ErrNoKeyInRecord = errors.New("Record Lacks Wanted key")

// ClearDeleted resets the deleted Record lists, emptying all.
func (u *under) ClearDeleted() {
	u.rl.Lock()
	defer u.rl.Unlock()
	u.deleted = make(map[string]bool)
}

// ClearTainted resets the tainted Record lists, emptying all.
func (u *under) ClearTainted() {
	u.rl.Lock()
	defer u.rl.Unlock()
	u.tainted = make(map[string]bool)
}

//==============================================================================

// TaintedRecords returns the tainted Records in this map. That is Records that
// have forgone some change.
func (u *under) TaintedRecords() []string {
	u.rl.RLock()
	defer u.rl.RUnlock()

	var Records []string

	for key := range u.tainted {
		Records = append(Records, key)
	}

	return Records
}

// DeletedRecords returns the deleted Records in this map. That is Records that
// have been removed.
func (u *under) DeletedRecords() []string {
	u.rl.RLock()
	defer u.rl.RUnlock()

	var Records []string

	for key := range u.deleted {
		Records = append(Records, key)
	}

	return Records
}

//==============================================================================

// Has returns true/false whether the Record into the storage maps.
func (u *under) Has(rec string) bool {
	u.rl.RLock()
	defer u.rl.RUnlock()

	_, ok := u.Records[rec]
	return ok
}

// HasRecord returns true/false whether the Record into the storage maps.
func (u *under) HasRecord(rec Record) bool {
	if _, ok := rec[u.key]; !ok {
		return false
	}

	key := rec[u.key].(string)

	u.rl.RLock()
	defer u.rl.RUnlock()

	_, ok := u.Records[key]
	return ok
}

//==============================================================================

// Remove removes the Record into the storage maps.
func (u *under) Remove(rec Record) error {
	if _, ok := rec[u.key]; !ok {
		return ErrNoKeyInRecord
	}

	key := rec[u.key].(string)

	u.rl.Lock()
	defer u.rl.Unlock()

	delete(u.Records, key)
	delete(u.tainted, key)
	u.deleted[key] = true
	return nil
}

//==============================================================================

// ErrNotFound is returned when a Record is not found.
var ErrNotFound = errors.New("Record Not Found")

// Get returns the internal Record stroed in the map.
func (u *under) Get(id string) (Record, error) {
	if u.Has(id) {
		return nil, ErrNotFound
	}

	u.rl.RLock()
	inrec := u.Records[id]
	u.rl.RUnlock()

	return inrec, nil
}

//==============================================================================

// Add adds the Record into the storage maps.
func (u *under) Add(rec Record) error {

	// If this does not have the specified Record key then return error.
	if _, ok := rec[u.key]; !ok {
		return ErrNoKeyInRecord
	}

	key := rec[u.key].(string)

	u.rl.RLock()
	inrec, ok := u.Records[key]
	u.rl.RUnlock()

	// If the Record has no previous instance then add it.
	if !ok {
		u.rl.Lock()
		defer u.rl.Unlock()

		u.Records[key] = rec
		return nil
	}

	// If the Record has a previous instance, then we need to merge it.
	MergeMaps(inrec, rec)

	u.rl.Lock()
	defer u.rl.Unlock()

	u.Records[key] = inrec
	u.tainted[key] = true
	return nil
}

//==============================================================================

// MergeMaps merges the the first map with the contents of the second map if
// the second map types match those of the first or if the first lacks an item
// from the second map. If both keys exists in both maps and their types are
// different then that key is excluded from merging.
func MergeMaps(to, from map[string]interface{}) {
	for key, value := range from {

		switch value.(type) {
		case bson.M:
			item := to[key]
			valMap := value.(map[string]interface{})
			itemMap := item.(map[string]interface{})
			MergeMaps(itemMap, valMap)
			continue

		case map[string]interface{}:
			item := to[key]
			valMap := value.(map[string]interface{})
			itemMap := item.(map[string]interface{})
			MergeMaps(itemMap, valMap)
			continue

		default:
			if _, ok := to[key]; !ok {
				to[key] = value
				continue
			}

			ttype := reflect.TypeOf(value)
			ftype := reflect.TypeOf(to[key])

			// Do this type match, if they don't exclude.
			if !ttype.AssignableTo(ftype) && !ttype.ConvertibleTo(ftype) {
				continue
			}

			if !ttype.AssignableTo(ftype) && ttype.ConvertibleTo(ftype) {
				vk := reflect.ValueOf(value)
				to[key] = vk.Convert(ftype)
			}

			to[key] = value
		}
	}
}

//==============================================================================
