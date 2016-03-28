package coquery

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

// Identity provides a interface that defines a request ID member method.
type Identity interface {
	RequestID() string
}

//==============================================================================

// Response provides a response struct for replies to coquery requests.
type Response struct {
	Req  RecordRequest `json:"-" bson:"-"`
	Data Parameters    `json:"reply" bson:"reply"`
}

// RequestID returns the request id for this response.
func (r *Response) RequestID() string {
	return r.Req.RequestID()
}

//==============================================================================

// ResponseError defines an interface for the error response for a coquery
// request.
type ResponseError interface {
	Identity
	Error() string
	Message() string
}

//==============================================================================
