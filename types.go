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
