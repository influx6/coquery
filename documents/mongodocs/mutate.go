package mongodocs

import (
	"fmt"

	"github.com/influx6/coquery"
	"github.com/influx6/coquery/data"
	"github.com/influx6/coquery/storage"
	"github.com/influx6/coquery/utils"
	"gopkg.in/mgo.v2/bson"
)

// Mutate provides a record mutator for the mongo storage system.
type Mutate struct {
	Events
	Db    DB
	Store storage.Store
}

// Do performs the operations for mutating a record within the internal coqery
// store and if successfully, send it into the db for storage.
func (m *Mutate) Do(dataReq interface{}, err error) (interface{}, error) {
	m.Log("mongodocs", "Mutate.Do", "Started : %s", utils.Query.Query(dataReq))

	req, ok := dataReq.(*coquery.Request)
	if !ok {
		m.Error("mongodocs", "Mutate.Do", coquery.ErrInvalidRequestType, "Completed")
		return nil, coquery.ErrInvalidRequestType
	}

	mux, ok := req.R.(*coquery.Mutate)
	if !ok {
		m.Error("mongodocs", "Mutate.Do", coquery.ErrInvalidRequestType, "Completed")
		return nil, coquery.ErrInvalidRequestType
	}

	// if req.LastResponse == nil {
	// 	err := errors.New("Mutate Only works on already selected records")
	// 	m.Error(mux.RequestID(), "Mutate.Do", err, "Completed")
	// 	return nil, err
	// }

	db, session, err := m.Db.New(mux.RequestID())
	if err != nil {
		m.Error(mux.RequestID(), "db.New", err, "Completed : New Session")
		return nil, &MError{Rid: mux.RequestID(), Msg: "New Session Failed", IError: err}
	}

	defer session.Close()

	param := (map[string]interface{})(mux.Parameter)
	new := true

	// If there were previous records then update those and save them.
	if records := req.LastResponse.Data; len(records) > 0 {
		new = false

		// Mutate all provided records and attempt to store back into cache store.
		for index, rec := range records {
			mrec := (map[string]interface{})(rec)
			storage.MergeMaps(mrec, param)

			// If the error failed, then stop and return as failure.
			if err := m.Store.Add(mrec); err != nil {
				m.Error(mux.RequestID(), "Mutate.Do", err, "Completed")
				return nil, &MError{
					Rid:    mux.RequestID(),
					Msg:    fmt.Sprintf("Mutate Failed: Record : %s", utils.Query.Query(rec)),
					IError: err,
				}
			}

			records[index] = mrec
		}

		// If we get here, then everything updated fine, lets push this into the db.
		for _, newRec := range records {
			val := newRec[m.Store.Key()]
			qry := bson.M{m.Store.Key(): val}

			m.Log(mux.RequestID(), "DBAction", "db.%s.upsert(%s,%s)", mux.Doc, utils.Query.Query(qry), utils.Query.Query(newRec))

			if _, err := db.C(mux.Doc).Upsert(qry, newRec); err != nil {
				m.Error(mux.RequestID(), "DBAction", err, "Completed")
				return nil, &MError{
					Rid:    mux.RequestID(),
					Msg:    fmt.Sprintf("Mutate DB Update: Record : %s", utils.Query.Query(newRec)),
					IError: err,
				}
			}
		}

		m.Log(mux.RequestID(), "Mutate.Do", "Completed")
		return &coquery.Response{
			Req:  mux,
			Data: records,
		}, nil
	}

	if new {
		val, ok := param[m.Store.Key()]
		if !ok {
			return nil, &MError{
				Rid:    mux.RequestID(),
				Msg:    utils.Query.Query(param),
				IError: fmt.Errorf("New Record Lacks Wanted Key: %s", m.Store.Key()),
			}
		}

		qry := bson.M{m.Store.Key(): val}

		m.Log(mux.RequestID(), "DBAction", "db.%s.upsert(%s,%s)", mux.Doc, utils.Query.Query(qry), utils.Query.Query(param))

		if _, err := db.C(mux.Doc).Upsert(qry, param); err != nil {
			m.Error(mux.RequestID(), "DBAction", err, "Completed")
			return nil, &MError{
				Rid:    mux.RequestID(),
				Msg:    fmt.Sprintf("Mutate DB Update: Record : %s", utils.Query.Query(param)),
				IError: err,
			}
		}

	}

	m.Log(mux.RequestID(), "Mutate.Do", "Completed")
	m.Log("mongodocs", "Mutate.Do", "Completed")

	return &coquery.Response{
		Req:  mux,
		Data: data.Parameters{mux.Parameter},
	}, nil
}

//==========================================================================================
