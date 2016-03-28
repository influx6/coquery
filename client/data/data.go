package data

// Response defines a type for the response retrieved from the endpoint.
type Response map[string]interface{}

//==============================================================================

// Pack defines the response to be recieved back from the API.
type Pack struct {
	RecordKey string     `json:"record_key"`
	RequestID string     `json:"request_id"`
	Batched   bool       `json:"batch"`
	DeltaID   string     `json:"delta_id"`
	Deltas    []string   `json:"delta_id"`
	Results   []Response `json:"results"`
}
