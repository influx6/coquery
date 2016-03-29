package crossdocs

import (
	"errors"
	"fmt"

	"github.com/influx6/coquery"
	"github.com/influx6/coquery/data"
	"github.com/influx6/coquery/storage"
	"github.com/influx6/coquery/utils"
)

// Collect provides a sumex.Proc implementing struct that allows spurning workers
// to provide `collect` features to query systems. It provides the ability to
// filter out the data to be returned to selected wanted fields and sub properties.
type Collect struct {
	Events
	Store storage.Store
}

// Do provides the member function for processing collect requests.
func (c *Collect) Do(req interface{}, err error) (interface{}, error) {
	c.Log("crossdocs", "Collect.Do", "Received Request : %s", utils.Query.Query(req))

	if err != nil {
		c.Error("crossdocs", "Collect.Do", err, "Completed")
		return nil, err
	}

	coreq, ok := req.(*coquery.Request)
	if !ok {
		c.Error("crossdocs", "Collect.Do", coquery.ErrInvalidRequestType, "Completed")
		return nil, err
	}

	cr, ok := coreq.R.(*coquery.Collects)
	if !ok {
		c.Error(coreq.R.RequestID(), "Collect.Do", coquery.ErrInvalidRequestType, "Completed")
		return nil, coquery.ErrInvalidRequestType
	}

	if coreq.LastResponse == nil {
		err := errors.New("Invalid Previous Response: Found Nil")
		c.Error(coreq.R.RequestID(), "Collect.Do", err, "Completed")
		return nil, &coquery.CoError{Rid: coreq.R.RequestID(), Msg: "No Previous Response", IError: err}
	}

	var records data.Parameters

	for _, record := range coreq.LastResponse.Data {

		rec := map[string]interface{}(record)

		item := make(data.Parameter)

		for _, key := range cr.Keys {

			// The rules for collecting record keys are not strict, hence if a key
			// is not found within a record it will be ignored and only the available
			// ones are collected. This allows flexibility when this behaviour
			// is intended.
			val, ok := storage.PullKeys(rec, key)
			if !ok {
				c.Error(cr.RequestID(), "Collect.Do", fmt.Errorf("Key not found"), "Key %s : Data %s : Failed", key, utils.Query.Query(rec))
				continue
			}

			lastKey := storage.LastKey(key)
			root, last := storage.BuildMap(key)
			last[lastKey] = val

			for key, part := range root {
				item[key] = part
			}

		}

		records = append(records, item)
	}

	c.Log(cr.RequestID(), "Collect.Do", "Completed")

	c.Log("crossdocs", "Collect.Do", "Completed")

	return &coquery.Response{
		Req:  cr,
		Data: records,
	}, nil
}
