package coquery

//==============================================================================

// Parameter defines the basic data type for all data received from the
// providers.
type Parameter map[string]interface{}

// Parameters defines a lists of Parameter types.
type Parameters []Parameter

//==============================================================================

// Identity provides a interface that defines a request ID member method.
type Identity interface {
	RequestID() string
}

//==============================================================================

// Response provides a response struct for replies to coquery requests.
type Response struct {
	Req  RecordRequest `json:"-" bson:"-"`
	RID  string        `json:"rid" bson:"rid"`
	Data Parameters    `json:"reply" bson:"reply"`
}

// RequestID returns the request id for this response.
func (r *Response) RequestID() string {
	return r.RID
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

// Find defines a record retrieve request based on the KV query.
type Find struct {
	Doc   string      `json:"doc" bson:"doc"`
	RID   string      `json:"rid" bson:"rid"`
	Key   string      `json:"key" bson:"key"`
	Value interface{} `json:"value" bson:"value"`
}

// RequestName returns the name for the giving request type.
func (f Find) RequestName() string {
	return "find"
}

//==============================================================================

// Collect retrieves specific keyed items from the coquery stores.
type Collect struct {
	RID  string   `json:"rid" bson:"rid"`
	Keys []string `json:"keys" bson:"keys"`
}

// RequestName returns the name for the giving request type.
func (f Collect) RequestName() string {
	return "collect"
}

//==============================================================================

// Mutate provides json data to be saved/augmented into a new version of the
// current document.
type Mutate struct {
	RID  string `json:"rid" bson:"rid"`
	Data []byte `json:"mutate" bson:"mutate"`
}

// RequestName returns the name for the giving request type.
func (f Mutate) RequestName() string {
	return "mutate"
}

//==============================================================================
