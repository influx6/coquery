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
	res   ResponseWriter
	store storage.Store
	ctx   *RequestContext
	diff  Diffs
}

// Write writes out the json response for the received request.
func (br *JSONResponseWriter) Write(context interface{}, res *Response, err ResponseError) error {

	// Record the diff record and store it for reporting as needed.
	br.diff.Put(br.store.TaintedRecords())
	br.store.ClearTainted()

	// Create the map to hold our json response.
	data := make(Parameter)

	data["request_id"] = br.ctx.RequestID
	data["batch"] = br.ctx.Batched

	if br.ctx.Diffing && br.ctx.DiffTag != "" {
		data["last_delta_id"] = br.ctx.DiffTag
	}

	if br.ctx.Diffing {

		// If we have no diffing tag then collect all the diffs and use the
		// last diff tag in the diff store as the new diff tag.
		if br.ctx.DiffTag == "" {
			diffs, tags := br.diff.All()

			if len(tags) > 0 {
				last := len(tags) - 1
				data["delta_id"] = tags[last]
			}

			data["deltas"] = diffs
		}
	}

	if res != nil {

		// data["results"] = Parameter{
		// 	"Error":   err.Error(),
		// 	"Message": err.Message(),
		// }
		// data["total"] = 1

		data["results"] = res.Data
		data["total"] = len(res.Data)
	}

	return br.res.Write(context, &Response{
		Req:  res.Req,
		Data: Parameters{data},
	}, err)
}

//==============================================================================
