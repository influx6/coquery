package storage

import (
	"errors"
	"reflect"
	"sync"

	"gopkg.in/mgo.v2/bson"
)

// records define a lists of record items within the understore.
type record map[string]interface{}

// Store defines a interface that provides necessary storage methods
type Store interface {
	Add(record) error
	Remove(record) error
	Has(record) bool
	ClearTainted()
	ClearDeleted()
	TaintedRecords() []string
	DeletedRecords() []string
}

// under provides an adequate means of storing full large scale json/json graph
// documents which allows us to cache.
type under struct {
	key     string
	rl      sync.RWMutex
	records map[string]record
	tainted map[string]bool
	deleted map[string]bool
}

// New returns a new instance of the under store.
func New(recordKey string) Store {
	un := under{
		key:     recordKey,
		records: make(map[string]record),
		tainted: make(map[string]bool),
		deleted: make(map[string]bool),
	}

	return &un
}

// ErrNoKeyInRecord is returned when the record lacks the wanted key.
var ErrNoKeyInRecord = errors.New("Record Lacks Wanted key")

// ClearDeleted resets the deleted record lists, emptying all.
func (u *under) ClearDeleted() {
	u.rl.Lock()
	defer u.rl.Unlock()
	u.deleted = make(map[string]bool)
}

// ClearTainted resets the tainted record lists, emptying all.
func (u *under) ClearTainted() {
	u.rl.Lock()
	defer u.rl.Unlock()
	u.tainted = make(map[string]bool)
}

// TaintedRecords returns the tainted records in this map. That is records that
// have forgone some change.
func (u *under) TaintedRecords() []string {
	u.rl.RLock()
	defer u.rl.RUnlock()

	var records []string

	for key := range u.tainted {
		records = append(records, key)
	}

	return records
}

// DeletedRecords returns the deleted records in this map. That is records that
// have been removed.
func (u *under) DeletedRecords() []string {
	u.rl.RLock()
	defer u.rl.RUnlock()

	var records []string

	for key := range u.deleted {
		records = append(records, key)
	}

	return records
}

// Has returns true/false whether the record into the storage maps.
func (u *under) Has(rec record) bool {
	if _, ok := rec[u.key]; !ok {
		return false
	}

	key := rec[u.key].(string)

	u.rl.RLock()
	defer u.rl.RUnlock()

	_, ok := u.records[key]
	return ok
}

// Remove removes the record into the storage maps.
func (u *under) Remove(rec record) error {
	if _, ok := rec[u.key]; !ok {
		return ErrNoKeyInRecord
	}

	key := rec[u.key].(string)

	u.rl.Lock()
	defer u.rl.Unlock()

	delete(u.records, key)
	delete(u.tainted, key)
	u.deleted[key] = true
	return nil
}

// Add adds the record into the storage maps.
func (u *under) Add(rec record) error {

	// If this does not have the specified record key then return error.
	if _, ok := rec[u.key]; !ok {
		return ErrNoKeyInRecord
	}

	key := rec[u.key].(string)

	u.rl.RLock()
	inrec, ok := u.records[key]
	u.rl.RUnlock()

	// If the record has no previous instance then add it.
	if !ok {
		u.rl.Lock()
		defer u.rl.Unlock()

		u.records[key] = rec
		return nil
	}

	// If the record has a previous instance, then we need to merge it.
	MergeMaps(inrec, rec)

	u.rl.Lock()
	defer u.rl.Unlock()

	u.records[key] = inrec
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
