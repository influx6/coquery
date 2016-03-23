package house

import (
	"errors"
	"fmt"

	"github.com/influx6/coquery"
	"github.com/influx6/coquery/storage"
	"github.com/influx6/coquery/utils"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

//==========================================================================================

// Find provides a find working for handling find requests.
type Find struct {
	EventLog
	Mongo Mongo
	Query Query
	Store storage.Store
}

// Do performs the necessary tasks passed to FindProc
func (f *Find) Do(data interface{}, err error) (interface{}, error) {
	f.Log("MongoProvider.FindProc", "Do", "Started : %s", f.Query.Query(data))

	if err != nil {
		f.Error("MongoProvider.FindProc", "Do", err, "Completed")
		return nil, err
	}

	req, ok := data.(*coquery.Request)
	if !ok {
		f.Error("MongoProvider.FindProc", "Do", coquery.ErrInvalidRequestType, "Completed")
		return nil, coquery.ErrInvalidRequestType
	}

	find, ok := req.R.(*coquery.Find)
	if !ok {
		f.Error("MongoProvider.FindProc", "Do", coquery.ErrInvalidRequestType, "Completed")
		return nil, coquery.ErrInvalidRequestType
	}

	var val interface{}

	if utils.IsDigits(find.Value) {
		val, _ = utils.ParseInt(find.Value)
	}

	var res coquery.Parameters
	found := true

	records, err := f.Store.GetByRef(find.Key, val)
	if err != nil {
		found = false
		f.Error("MongoProvider.FindProc", "Do", err, "Completed : Store : Not Found")
	}

	if found {
		for _, recs := range records {
			res = append(res, coquery.Parameter(recs))
		}

		f.Log("MongoProvider.FindProc", "Do", "Info : Store : Record Found")

		f.Log("MongoProvider.FindProc", "Do", "Completed")
		return &coquery.Response{
			Req:  find,
			Data: res,
		}, nil
	}

	fn := func(c *mgo.Collection) error {
		q := bson.M{find.Key: val}
		f.Log("MongoProvider.FindProc", "DBAction", "db.%s.find(%s)", c.Name, f.Query.Query(q))
		return c.Find(q).All(&res)
	}

	err = f.Mongo.ExecuteDB("MongoProvider.FindProc", find.Doc, fn)
	if err != nil {
		f.Error("MongoProvider.FindProc", "Do", err, "Completed")
		return nil, &MError{Rid: find.RID, Msg: "FindProc Failed", IError: err}
	}

	f.Log("MongoProvider.FindProc", "Do", "Info : Response : %s", f.Query.Query(res))

	for _, record := range res {
		if err := f.Store.AddRef((map[string]interface{})(record), find.Key); err != nil {
			f.Error("MongoProvider.FindProc", "Do", err, "Info : Store.AddRef : Key[%s]", find.Key)
		}
	}

	f.Log("MongoProvider.FindProc", "Do", "Completed")

	return &coquery.Response{
		Req:  find,
		Data: res,
	}, nil
}

//==========================================================================================

// Mutate provides a record mutator for the mongo storage system.
type Mutate struct {
	EventLog
	Mongo Mongo
	Query Query
	Store storage.Store
}

// Do performs the operations for mutating a record within the internal coqery
// store and if successfully, send it into the db for storage.
func (m *Mutate) Do(data interface{}, err error) (interface{}, error) {
	m.Log("MongoProvider.Mutate", "Do", "Started : %s", m.Query.Query(data))

	req, ok := data.(*coquery.Request)
	if !ok {
		m.Error("MongoProvider.Mutate", "Do", coquery.ErrInvalidRequestType, "Completed")
		return nil, coquery.ErrInvalidRequestType
	}

	mux, ok := req.R.(*coquery.Mutate)
	if !ok {
		m.Error("MongoProvider.Mutate", "Do", coquery.ErrInvalidRequestType, "Completed")
		return nil, coquery.ErrInvalidRequestType
	}

	if req.LastResponse == nil {
		err := errors.New("Mutate Only works on already selected records")
		m.Error("MongoProvider.Mutate", "Do", err, "Completed")
		return nil, err
	}

	param := (map[string]interface{})(mux.Parameter)

	records := req.LastResponse.Data

	// Mutate all provided records and attempt to store back into cache store.
	for index, rec := range records {
		mrec := (map[string]interface{})(rec)
		storage.MergeMaps(mrec, param)

		// If the error failed, then stop and return as failure.
		if err := m.Store.Add(mrec); err != nil {
			m.Error("MongoProvider.Mutate", "Do", err, "Completed")
			return nil, &MError{
				Rid:    mux.RequestID(),
				Msg:    fmt.Sprintf("Mutate Failed: Record : %s", m.Query.Query(rec)),
				IError: err,
			}
		}

		records[index] = mrec
	}

	// If we get here, then everything updated fine, lets push this into the db.
	for _, newRec := range records {
		key := m.Store.Key()
		val := newRec[m.Store.Key()]

		err := (func(k string, v interface{}, d interface{}) error {
			return m.Mongo.ExecuteDB("MongoProvider.FindProc", mux.Doc, func(c *mgo.Collection) error {
				qry := bson.M{k: v}
				m.Log("MongoProvider.Mutate", "DBAction", "db.%s.upsert(%s,%s)", c.Name, m.Query.Query(qry), m.Query.Query(d))
				_, err := c.Upsert(qry, d)
				return err
			})
		})(key, val, newRec)

		if err != nil {
			m.Error("MongoProvider.Mutate", "Do.DBAction", err, "Completed")
		}
	}

	m.Log("MongoProvider.Mutate", "Do", "Completed")
	return &coquery.Response{
		Req:  mux,
		Data: records,
	}, nil
}

//==========================================================================================

// All provides a find working for handling find requests.
type All struct {
	EventLog
	Mongo Mongo
	Query Query
	Store storage.Store
}

// Do performs the necessary tasks passed to FindProc
func (a *All) Do(data interface{}, err error) (interface{}, error) {
	a.Log("MongoProvider.All", "Do", "Started : %s", a.Query.Query(data))

	if err != nil {
		a.Error("MongoProvider.All", "Do", err, "Completed")
		return nil, err
	}

	req, ok := data.(*coquery.Request)
	if !ok {
		a.Error("MongoProvider.All", "Do", coquery.ErrInvalidRequestType, "Completed")
		return nil, coquery.ErrInvalidRequestType
	}

	find, ok := req.R.(*coquery.FindN)
	if !ok {
		a.Error("MongoProvider.All", "Do", coquery.ErrInvalidRequestType, "Completed")
		return nil, coquery.ErrInvalidRequestType
	}

	var res coquery.Parameters
	var total int

	if find.Skip < 0 {
		find.Skip = 0
	}

	// If we had a previous response, then we are dealing with a concatenated
	// operation on the last request, so we build our strategy on this.
	if req.LastResponse != nil {
		data := req.LastResponse.Data[find.Skip:]
		total = len(data)

		if find.Amount < 0 {
			find.Amount = total
		}

		data = data[:find.Amount]

		for _, recs := range data {
			res = append(res, coquery.Parameter(recs))
		}

		a.Log("MongoProvider.All", "Do", "Info : Store : Record Found")

		a.Log("MongoProvider.All", "Do", "Completed")
		return &coquery.Response{
			Req:  find,
			Data: res,
		}, nil
	}

	fn := func(c *mgo.Collection) error {
		a.Log("MongoProvider.All", "DBAction", "db.%s.find({}).count()", c.Name)
		count, err := c.Find(nil).Count()
		if err != nil {
			return err
		}

		total = count
		return nil
	}

	err = a.Mongo.ExecuteDB("MongoProvider.FindProc", find.Doc, fn)
	if err != nil {
		a.Error("MongoProvider.All", "Do", err, "Completed")
		return nil, &MError{Rid: find.RID, Msg: "FindProc Failed", IError: err}
	}

	if find.Amount < 0 {
		find.Amount = total
	}

	if find.Amount+find.Skip <= a.Store.Length() {
		records := a.Store.Select(find.Amount, find.Skip)

		for _, recs := range records {
			res = append(res, coquery.Parameter(recs))
		}

		a.Log("MongoProvider.All", "Do", "Info : Store : Record Found")

		a.Log("MongoProvider.All", "Do", "Completed")
		return &coquery.Response{
			Req:  find,
			Data: res,
		}, nil
	}

	fn = func(c *mgo.Collection) error {
		a.Log("MongoProvider.All", "DBAction", "db.%s.find({}).skip(%d).limit(%d)", c.Name, find.Amount, find.Skip)
		return c.Find(nil).Skip(find.Skip).Limit(find.Amount).All(&res)
	}

	err = a.Mongo.ExecuteDB("MongoProvider.FindProc", find.Doc, fn)
	if err != nil {
		a.Error("MongoProvider.All", "Do", err, "Completed")
		return nil, &MError{Rid: find.RID, Msg: "FindProc Failed", IError: err}
	}

	a.Log("MongoProvider.All", "Do", "Info : Response : %s", a.Query.Query(res))

	for _, record := range res {
		if err := a.Store.Add((map[string]interface{})(record)); err != nil {
			a.Error("MongoProvider.All", "Do", err, "Info : Store.Add")
		}
	}

	a.Log("MongoProvider.All", "Do", "Completed")

	return &coquery.Response{
		Req:  find,
		Data: res,
	}, nil
}
