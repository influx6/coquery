package coquery

import "github.com/influx6/coquery/data"

// Identity provides a interface that defines a request ID member method.
type Identity interface {
	RequestID() string
}

//==============================================================================

// Response provides a response struct for replies to coquery requests.
type Response struct {
	Req  RecordRequest   `json:"-" bson:"-"`
	Data data.Parameters `json:"reply" bson:"reply"`
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
