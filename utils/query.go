package utils

import "encoding/json"

//==========================================================================================

// Query provides a json stringifier.
var Query query

type query struct{}

// QueryIndent returns the stringified version of the giving data and indents
// its result. Uses json.Marshal underneath.
func (q query) QueryIndent(ms interface{}) string {
	data, err := json.MarshalIndent(ms, "", "\n")
	if err != nil {
		return ""
	}

	return string(data)
}

// Query returns a stringified version of the provided argument
// using json.Marshal.
func (q query) Query(ms interface{}) string {
	data, err := json.Marshal(ms)
	if err != nil {
		return ""
	}

	return string(data)
}
