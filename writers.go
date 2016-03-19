package coquery

import "github.com/influx6/coquery/storage"

//==============================================================================

// BatchResponseWriter provides a response writer for batch queries.
type BatchResponseWriter struct {
	Res       ResponseWriter
	data      Parameters
	total     int
	collected int
}

//==============================================================================

// dupReq provides a decorator for a response error which disguses it as a
// RecordRequest.
type dupReq struct {
	ResponseError
}

// RequestName returns a empty string for a dupReq.
func (d *dupReq) RequestName() string {
	return ""
}

// Write writes the response for batch request, keeping count until all responses
// are received and writes them in order of reception.
func (br *BatchResponseWriter) Write(context interface{}, res *Response, err ResponseError) error {
	var r RecordRequest

	if res != nil {
		r = res.Req
	} else {
		r = &dupReq{err}
	}

	if br.collected >= br.total {
		return br.Res.Write(context, &Response{
			Req:  r,
			Data: br.data,
		}, nil)
	}

	// Add the data response to the response list.
	if res != nil {
		br.data = append(br.data, Parameter{"data": res.Data})
	} else {
		br.data = append(br.data, Parameter{
			"Error":   err.Error(),
			"Message": err.Message(),
			// "Request": err.RequestID(),
		})
	}

	br.collected++

	return nil
}

//==============================================================================

// JSONResponseWriter provides the coquery API JSON spec writer, which ensures
// we adequately provide proper response for our API requests.
type JSONResponseWriter struct {
	Res   ResponseWriter
	store storage.Store
	Ctx   *RequestContext
	diff  Diffs
}

// Write writes out the json response for the received request.
func (br *JSONResponseWriter) Write(context interface{}, res *Response, err ResponseError) error {

	return nil
}

//==============================================================================
