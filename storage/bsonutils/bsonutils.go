package bsonutils

import (
	"reflect"

	"gopkg.in/mgo.v2/bson"
)

//==============================================================================

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
