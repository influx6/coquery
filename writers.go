package coquery

import (
	"github.com/influx6/coquery/data"
	"github.com/influx6/coquery/storage"
)

//==============================================================================

// BatchResponseWriter provides a response writer for batch queries.
type BatchResponseWriter struct {
	Res       ResponseWriter
	data      data.Parameters
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
		br.data = append(br.data, data.Parameter{"data": res.Data})
	} else {
		br.data = append(br.data, data.Parameter{
			"Error":   err.Error(),
			"Message": err.Message(),
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
	ctx   *data.RequestContext
	diff  Diffs
}

// Write writes out the json response for the received request.
func (br *JSONResponseWriter) Write(context interface{}, res *Response, err ResponseError) error {
	if err != nil {
		return br.res.Write(context, nil, err)
	}

	// Record the diff record and store it for reporting as needed.
	br.diff.Put(br.store.TaintedRecords())
	br.store.ClearTainted()

	// Create the map to hold our json response.
	mdata := make(data.Parameter)

	mdata["record_key"] = br.store.Key()
	mdata["request_id"] = br.ctx.RequestID
	mdata["batch"] = len(br.ctx.Queries) > 1

	if br.ctx.Diffs && br.ctx.DiffTag != "" {
		mdata["last_delta_id"] = br.ctx.DiffTag
	}

	var req RecordRequest

	mdata["results"] = res.Data
	mdata["total"] = len(res.Data)
	req = res.Req

	// if req == nil {
	// 	req = &dupReq{ResponseError: err}
	// }

	if !br.ctx.Diffs {
		return br.res.Write(context, &Response{
			Req:  req,
			Data: []data.Parameter{mdata},
		}, err)
	}

	var key string
	keys := br.diff.Keys()
	last := len(keys) - 1

	if last > -1 && last < len(keys) {
		key = keys[last]
		mdata["delta_id"] = keys[last]
	}

	// If we have no diffing tag then collect all the diffs and use the
	// last diff tag in the diff store as the new diff tag.
	if !br.diff.Has(br.ctx.DiffTag) {

		var diff []string

		if len(br.ctx.DiffWatch) > 0 {

			// Collect the changes map.
			changes := br.diff.Analyze(br.ctx.DiffWatch)

			// Collect only keys that indeed have changed.
			for key, status := range changes {
				if status {
					diff = append(diff, key)
				}
			}

		} else {
			diff = br.diff.Get(key)
		}

		mdata["deltas"] = diff

		return br.res.Write(context, &Response{
			Req:  res.Req,
			Data: []data.Parameter{mdata},
		}, err)
	}

	var diff []string

	if len(br.ctx.DiffWatch) > 0 {
		changes := br.diff.AnalyzeWith(br.ctx.DiffTag, br.ctx.DiffWatch)

		// Collect only keys that indeed have changed.
		for key, status := range changes {
			if status {
				diff = append(diff, key)
			}
		}

	} else {
		diff = br.diff.PullFrom(br.ctx.DiffTag)
	}

	mdata["delta_id"] = key
	mdata["deltas"] = diff

	return br.res.Write(context, &Response{
		Req:  res.Req,
		Data: []data.Parameter{mdata},
	}, err)
}

//==============================================================================
