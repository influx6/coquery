package storage

import "sync"

// records define a lists of record items within the understore.
type record map[string]interface{}

// underStore provides an adequate means of storing full large scale json/json graph
// documents which allows us to cache.
type underStore struct {
	rl      sync.RWMutex
	records map[string]record
	all     []record
	tainted []record
}
