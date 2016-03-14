package storage

import (
	"errors"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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
	AddRef(Record, string) error
	ModRef(Record, string) error
	ModRefBy(Record, string, bool) error

	Remove(Record) error
	RemoveByKey(Record) error
	RemoveByValue(Record) error
	Delete(string) error

	Get(string) (Record, error)

	TaintedRecords() []string
	DeletedRecords() []string
}

//==============================================================================

// Truthtable defines a map of string values with a bool value.
type Truthtable map[string]bool

// UnSet sets the giving key as false in the truthtable
func (t Truthtable) UnSet(k string) {
	t[k] = false
}

// Set adds a key into the map setting its value to true.
func (t Truthtable) Set(k string) {
	t[k] = true
}

// Has returns true/false if the giving key exists.
func (t Truthtable) Has(k string) bool {
	return t[k]
}

// RecordRefList defines a lists of record refs associated with a lists of
// record keys for fast indexes.
type RecordRefList map[interface{}]Truthtable

// Get returns the k truthable for the giving key else returns nil if it does
// not exists.
func (r RecordRefList) Get(k interface{}) Truthtable {
	v, ok := r[k]
	if !ok {
		return nil
	}

	return v
}

// Set adds the giving key and a giving value into its provided truth table.
func (r RecordRefList) Set(k interface{}, v string) {
	_, ok := r[k]
	if !ok {
		tm := make(Truthtable)
		tm.Set(v)
		r[k] = tm
		return
	}

	tm := r[k]
	tm.Set(v)
}

// HasTruth returns true/false if the key has a Truthtable and has the giving
// value set.
func (r RecordRefList) HasTruth(k interface{}, v string) bool {
	rv, ok := r[k]
	if !ok {
		return false
	}

	return rv.Has(v)
}

// Has returns true/false if the giving Truthtable exists.
func (r RecordRefList) Has(k interface{}) bool {
	_, ok := r[k]
	return ok
}

// RefList defines a map of RecordRefList for storing Truthtables.
type RefList map[string]RecordRefList

// Get returns the giving RecordRefList associated with this key.
func (r RefList) Get(k string) RecordRefList {
	rf, ok := r[k]
	if !ok {
		return nil
	}

	return rf
}

// Add adds the giving key into the reflists and returns its RecordRefList.
func (r RefList) Add(k string) RecordRefList {
	rf, ok := r[k]
	if !ok {
		rf = make(RecordRefList)
		r[k] = rf
	}

	return rf
}

// Has returns true/false if the giving RecordRefList exists for the key.
func (r RefList) Has(k string) bool {
	_, ok := r[k]
	return ok
}

//==============================================================================

// under provides an adequate means of storing full large scale json/json graph
// documents which allows us to cache.
type under struct {
	key        string
	rl         sync.RWMutex
	records    map[string]Record
	tainted    map[string]bool
	deleted    map[string]bool
	scans      map[string]int64
	rfl        sync.RWMutex
	recordRefs RefList
	afl        sync.RWMutex
	active     map[string]int64
}

// New returns a new instance of the under store.
func New(recordKey string) Store {
	un := under{
		key:        recordKey,
		records:    make(map[string]Record),
		tainted:    make(map[string]bool),
		deleted:    make(map[string]bool),
		scans:      make(map[string]int64),
		active:     make(map[string]int64),
		recordRefs: make(RefList),
	}

	return &un
}

// NewExpirable returns a new store but which has a expiration timer
// check on all records in the store. If a record has not being assed
// for a while then that record is deleted from within the stores.
func NewExpirable(recordKey string, maxAge time.Duration) Store {
	un := under{
		key:        recordKey,
		records:    make(map[string]Record),
		tainted:    make(map[string]bool),
		deleted:    make(map[string]bool),
		scans:      make(map[string]int64),
		active:     make(map[string]int64),
		recordRefs: make(RefList),
	}

	// Lunch our expiration checker in a go-routine
	go func() {
		for {
			<-time.After(maxAge)

			un.afl.RLock()
			defer un.afl.RUnlock()

			for key, state := range un.active {
				if du := atomic.LoadInt64(&state); du-1 <= 0 {
					un.Delete(key)
					continue
				}

				atomic.StoreInt64(&state, 1)
				un.active[key] = state
			}

		}

	}()

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

//==============================================================================

// Has returns true/false whether the Record into the storage maps.
func (u *under) Has(rec string) bool {
	u.rl.RLock()
	defer u.rl.RUnlock()

	_, ok := u.records[rec]
	return ok
}

// ValidRecord returns true/false if the record has the needed key within it.
// of the needed type.
func (u *under) ValidRecord(rec Record) bool {
	if m, ok := rec[u.key]; ok {
		if _, osk := m.(string); !osk {
			return false
		}
	}

	return true
}

// HasRecord returns true/false whether the Record into the storage maps.
func (u *under) HasRecord(rec Record) bool {
	key, ok := rec[u.key].(string)
	if !ok {
		return false
	}

	u.rl.RLock()
	defer u.rl.RUnlock()

	_, ok = u.records[key]
	return ok
}

//==============================================================================

// RemoveByValue removes the Record into the storage maps.
func (u *under) RemoveByValue(rec Record) error {
	if !u.ValidRecord(rec) {
		return ErrNoKeyInRecord
	}

	key := rec[u.key].(string)

	u.rl.RLock()
	inrec := u.records[key]
	u.rl.RUnlock()

	RemoveValuesDiff(inrec, rec)
	return nil
}

// RemoveByKey removes the Record into the storage maps.
func (u *under) RemoveByKey(rec Record) error {
	if !u.ValidRecord(rec) {
		return ErrNoKeyInRecord
	}

	key := rec[u.key].(string)

	u.rl.RLock()
	inrec := u.records[key]
	u.rl.RUnlock()

	RemoveMapDiff(inrec, rec)
	return nil
}

// ErrInvalidRecKey is returned when a key has no associated record in store.
var ErrInvalidRecKey = errors.New("Invalid Record Key")

// Delete removes the Record into the storage maps using its key.
func (u *under) Delete(key string) error {
	_, ok := u.records[key]
	if !ok {
		return ErrInvalidRecKey
	}

	u.rl.Lock()
	defer u.rl.Unlock()

	delete(u.records, key)
	delete(u.tainted, key)

	// Remove this record from all refs.
	for _, ref := range u.recordRefs {
		for _, rfg := range ref {
			rfg.UnSet(key)
		}
	}

	u.deleted[key] = true
	return nil
}

// Remove removes the Record into the storage maps.
func (u *under) Remove(rec Record) error {
	if !u.ValidRecord(rec) {
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

//==============================================================================

// ErrNotFound is returned when a Record is not found.
var ErrNotFound = errors.New("Record Not Found")

// Get returns the internal Record stroed in the map.
func (u *under) Get(id string) (Record, error) {
	if !u.Has(id) {
		return nil, ErrNotFound
	}

	u.rl.RLock()
	inrec := u.records[id]
	u.rl.RUnlock()

	u.afl.RLock()
	d := u.active[id]
	atomic.AddInt64(&d, 1)
	u.afl.RUnlock()

	u.afl.Lock()
	u.active[id] = d
	u.afl.Unlock()

	return inrec, nil
}

//==============================================================================

// Add adds the Record into the storage maps.
func (u *under) Add(rec Record) error {

	// If this does not have the specified Record key then return error.
	if !u.ValidRecord(rec) {
		return ErrNoKeyInRecord
	}

	key := rec[u.key].(string)

	u.rl.RLock()
	inrec, ok := u.records[key]
	u.rl.RUnlock()

	// If the Record has no previous instance then add it.
	if !ok {
		u.rl.Lock()
		defer u.rl.Unlock()

		u.records[key] = rec

		u.afl.Lock()
		u.active[key] = 2
		u.afl.Unlock()
		return nil
	}

	// If the Record has a previous instance, then we need to merge it.
	MergeMaps(inrec, rec)

	u.rl.Lock()
	defer u.rl.Unlock()

	u.records[key] = inrec
	u.tainted[key] = true

	u.afl.RLock()
	d := u.active[key]
	u.afl.RUnlock()

	u.afl.Lock()
	atomic.AddInt64(&d, 1)
	u.active[key] = d
	u.afl.Unlock()

	return nil
}

// ErrInvalidRefKey is returned when the reference key is not found in the
// provided record.
var ErrInvalidRefKey = errors.New("Invalid Reference Key")

// AdjustRef adjusts the reference data within the record lists.
func (u *under) AdjustRef(reckey string, refKey string) error {
	if !u.Has(reckey) {
		return ErrInvalidRecKey
	}

	rec, err := u.Get(reckey)
	if err != nil {
		return err
	}

	return u.ModRefBy(rec, refKey, false)
}

// ModRef adds the record into the store, modding as necessary and adjusts the
// ref details.
func (u *under) ModRef(rec Record, refKey string) error {
	if !u.ValidRecord(rec) {
		return ErrNoKeyInRecord
	}

	return u.ModRefBy(rec, refKey, true)
}

// AddRef adds the record into the map if its new and adjusts its reference
// index lists with the needed keyed index. If a record already exists, only
// the reference information is stored with no data modified.
func (u *under) AddRef(rec Record, refKey string) error {
	if !u.ValidRecord(rec) {
		return ErrNoKeyInRecord
	}

	var new bool
	recKey := rec[u.key].(string)

	if !u.Has(recKey) {
		new = true
	}

	return u.ModRefBy(rec, refKey, new)
}

// ModRefBy adds the record if new into the storage map and adds new reference
// using this type of key and its corresponding value, updating any records found
// in its lists that matches it.
func (u *under) ModRefBy(rec Record, refKey string, new bool) error {
	if !u.ValidRecord(rec) {
		return ErrNoKeyInRecord
	}

	coreKey := rec[u.key].(string)

	if new {
		u.Add(rec)
	}

	recVal, ok := PullKeys(rec, refKey)
	if !ok {
		return ErrInvalidRefKey
	}

	var scanning int64

	us, ok := u.scans[refKey]
	if ok {
		scanning = atomic.LoadInt64(&us)
	} else {
		u.scans[refKey] = scanning
	}

	u.rfl.RLock()
	refs := u.recordRefs.Add(refKey)
	u.rfl.RUnlock()

	// If we are scanning for this key already then add and skip this record
	if scanning > 1 {
		refs.Set(recVal, coreKey)
		return nil
	}

	atomic.StoreInt64(&us, 1)
	u.scans[refKey] = us

	u.rl.RLock()
	defer u.rl.RUnlock()

	// Run through the lists and set it as scanning.
	for recKey, rec := range u.records {
		if recValf, ok := PullKeys(rec, refKey); ok {
			if recVal == recValf {
				refs.Set(recVal, recKey)
			}
		}
	}

	atomic.StoreInt64(&us, 0)

	u.afl.RLock()
	d := u.active[coreKey]
	u.afl.RUnlock()

	u.afl.Lock()
	atomic.AddInt64(&d, 1)
	u.active[coreKey] = d
	u.afl.Unlock()

	return nil
}

//==============================================================================

// PullKeys will pull out a key values even when presented by a period delimited
// depth keys. It returns true/false as second value if all keys where found.
func PullKeys(rec Record, key string) (interface{}, bool) {
	keys := strings.Split(key, ".")

	last := len(keys) - 1
	lastKey := keys[last]
	subs := keys[:last]

	val, ok := finder(rec, subs)
	if !ok {
		return nil, false
	}

	mval, ok := val.(map[string]interface{})
	if !ok {
		return nil, false
	}

	gval, ok := mval[lastKey]
	if !ok {
		return nil, false
	}

	return gval, true
}

func finder(target map[string]interface{}, ks []string) (interface{}, bool) {
	mv, ok := target[ks[0]]
	if !ok {
		return nil, false
	}

	if len(ks) == 1 {
		return mv, true
	}

	mt, ok := mv.(map[string]interface{})
	if !ok {
		return nil, false
	}

	return finder(mt, ks[1:])
}

// RemoveValuesDiff removes all properties according to their corresponding level
// from the diff map checking if the values match,if found within the first map.
func RemoveValuesDiff(target, diff map[string]interface{}) {
	for key, value := range diff {
		switch value.(type) {
		case map[string]interface{}:
			item, ok := target[key]
			if !ok {
				continue
			}

			mo, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			RemoveMapDiff(mo, value.(map[string]interface{}))
			continue
		default:
			item, ok := target[key]
			if !ok {
				continue
			}

			if item != value {
				continue
			}

			delete(target, key)
		}
	}
}

// RemoveMapDiff removes all properties according to their corresponding level
// from the diff map, if found within the first map.
func RemoveMapDiff(target, diff map[string]interface{}) {
	for key, value := range diff {
		switch value.(type) {
		case map[string]interface{}:
			item, ok := target[key]
			if !ok {
				continue
			}

			mo, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			RemoveMapDiff(mo, value.(map[string]interface{}))
			continue
		default:
			_, ok := target[key]
			if !ok {
				continue
			}

			delete(target, key)
		}
	}
}

// MergeMaps merges the the first map with the contents of the second map if
// the second map types match those of the first or if the first lacks an item
// from the second map. If both keys exists in both maps and their types are
// different then that key is excluded from merging.
func MergeMaps(to, from map[string]interface{}) {
	for key, value := range from {

		switch value.(type) {

		case bson.M:
			valMap := value.(bson.M)

			var tom map[string]interface{}

			item, ok := to[key]
			if !ok {
				tom = make(map[string]interface{})
			} else {
				if mo, ok := item.(map[string]interface{}); ok {
					tom = mo
				} else {
					continue
				}
			}

			MergeMaps(tom, BSONtoMap(valMap))
			to[key] = tom
			continue

		case map[string]interface{}:
			valMap := value.(map[string]interface{})
			var tom map[string]interface{}

			item, ok := to[key]
			if !ok {
				tom = make(map[string]interface{})
			} else {
				if mo, ok := item.(map[string]interface{}); ok {
					tom = mo
				} else {
					continue
				}
			}

			MergeMaps(tom, valMap)
			to[key] = tom
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

// CopyMap copies a map into a raw map structure.
func CopyMap(m map[string]interface{}) map[string]interface{} {
	to := make(map[string]interface{})
	mapCopy(to, m)
	return to
}

// BSONtoMap copies a bson.M map into a raw map structure.
func BSONtoMap(m bson.M) map[string]interface{} {
	to := make(map[string]interface{})
	bsonCopy(to, m)
	return to
}

// bsonCopy copies one bson.M file, cloning as necessary down the data trees.
func bsonCopy(to map[string]interface{}, from bson.M) {
	for key, value := range from {
		switch value.(type) {
		case bson.M:
			mn := make(map[string]interface{})
			bsonCopy(mn, value.(bson.M))
			to[key] = mn
			continue
		case map[string]interface{}:
			mapCopy(to, value.(map[string]interface{}))
			continue
		default:
			to[key] = value
			continue
		}
	}
}

// mapCopy copies one map details, cloning as necessary down the data trees.
func mapCopy(to, from map[string]interface{}) {
	for key, value := range from {
		switch value.(type) {
		case bson.M:
			mn := make(map[string]interface{})
			bsonCopy(mn, value.(bson.M))
			to[key] = mn
			continue
		case map[string]interface{}:
			mn := make(map[string]interface{})
			mapCopy(mn, value.(map[string]interface{}))
			to[key] = mn
			continue
		default:
			to[key] = value
			continue
		}
	}
}

//==============================================================================
