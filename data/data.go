package data

//==============================================================================

// RequestContext provides a request context which details the needed information
// a coquery.Request entails. It allows us organize the behaviour and response
// for a request.
// NoJSON allows a request avoid wrapping its writer with a JSONResponseWriter.
type RequestContext struct {
	RequestID string   `json:"request_id"`
	Queries   []string `json:"queries"`
	Diffs     bool     `json:"diffing"`
	DiffTag   string   `json:"diff_tag"`
	DiffWatch []string `json:"diff_watch"`
	NoJSON    bool     `json:"no_json"`
}

//==============================================================================

// Parameter defines the basic data type for all data received from the
// providers.
type Parameter map[string]interface{}

// Has returns true/false if the giving key exists there.
func (p Parameter) Has(k string) bool {
	_, ok := p[k]
	return ok
}

// Set sets the giving key with the provided value.
func (p Parameter) Set(k string, v interface{}) {
	p[k] = v
}

// Get retrieves the value of a giving key if it exists else nil is returned.
func (p Parameter) Get(k string) interface{} {
	return p[k]
}

// Parameters defines a lists of Parameter types.
type Parameters []Parameter

//==============================================================================

// ResponseMeta provides a meta record which provides specific information for
// a giving response.
type ResponseMeta struct {
	RecordKey string `json:"record_key"`
	RequestID string `json:"request_id"`
	DeltaID   string `json:"delta_id"`
}

// ResponsePack defines the response to be recieved back from the API.
type ResponsePack struct {
	RecordKey string     `json:"record_key"`
	RequestID string     `json:"request_id"`
	Batched   bool       `json:"batch"`
	DeltaID   string     `json:"delta_id"`
	Deltas    []string   `json:"delta_id"`
	Results   Parameters `json:"results"`
}

//==============================================================================
