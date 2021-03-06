package mongodocs

import (
	"github.com/influx6/coquery"
	"github.com/influx6/coquery/data"
	"github.com/influx6/coquery/storage"
	"github.com/influx6/coquery/utils"
)

// All provides a find working for handling find requests.
type All struct {
	Events
	Db    DB
	Store storage.Store
}

// Do performs the necessary tasks passed to FindProc
func (a *All) Do(dataReq interface{}, err error) (interface{}, error) {
	a.Log("mongodocs", "All.Do", "Started : %s", utils.Query.Query(dataReq))

	if err != nil {
		a.Error("mongodocs", "All.Do", err, "Completed")
		return nil, err
	}

	req, ok := dataReq.(*coquery.Request)
	if !ok {
		a.Error("mongodocs", "All.Do", coquery.ErrInvalidRequestType, "Completed")
		return nil, coquery.ErrInvalidRequestType
	}

	find, ok := req.R.(*coquery.FindN)
	if !ok {
		a.Error("mongodocs", "All.Do", coquery.ErrInvalidRequestType, "Completed")
		return nil, coquery.ErrInvalidRequestType
	}

	var res data.Parameters
	var total int

	if find.Skip < 0 {
		find.Skip = 0
	}

	// If we had a previous response, then we are dealing with a concatenated
	// operation on the last request, so we build our strategy on this.
	if req.LastResponse != nil {
		datam := req.LastResponse.Data[find.Skip:]
		total = len(datam)

		if find.Amount < 0 {
			find.Amount = total
		}

		datam = datam[:find.Amount]

		for _, recs := range datam {
			res = append(res, data.Parameter(recs))
		}

		a.Log(find.RequestID(), "All.Do", "Info : Store : Record Found")

		a.Log(find.RequestID(), "All.Do", "Completed")
		return &coquery.Response{
			Req:  find,
			Data: res,
		}, nil
	}

	db, session, err := a.Db.New(find.RequestID())
	if err != nil {
		a.Error(find.RequestID(), "db.New", err, "Completed : New Session")
		return nil, &MError{Rid: find.RID, Msg: "New Session Failed", IError: err}
	}

	defer session.Close()

	a.Log(find.RequestID(), "DBAction", "db.%s.find({}).count()", find.Doc)

	total, err = db.C(find.Doc).Find(nil).Count()
	if err != nil {
		a.Error(find.RequestID(), "All.Do", err, "Completed")
		return nil, &MError{Rid: find.RID, Msg: "FindProc Failed", IError: err}
	}

	if find.Amount < 0 {
		find.Amount = total
	}

	if find.Amount+find.Skip <= a.Store.Length() {
		records := a.Store.Select(find.Amount, find.Skip)

		for _, recs := range records {
			res = append(res, data.Parameter(recs))
		}

		a.Log(find.RequestID(), "All.Do", "Info : Store : Record Found")

		a.Log(find.RequestID(), "All.Do", "Completed")
		return &coquery.Response{
			Req:  find,
			Data: res,
		}, nil
	}

	a.Log(find.RequestID(), "DBAction", "db.%s.find({}).skip(%d).limit(%d)", find.Doc, find.Skip, find.Amount)

	if err := db.C(find.Doc).Find(nil).Skip(find.Skip).Limit(find.Amount).All(&res); err != nil {
		a.Error(find.RequestID(), "DBAction", err, "Completed")
		return nil, &MError{Rid: find.RID, Msg: "All Failed", IError: err}
	}

	a.Log(find.RequestID(), "All.Do", "Info : Response : %s", utils.Query.Query(res))

	for _, record := range res {
		if err := a.Store.Add((map[string]interface{})(record)); err != nil {
			a.Error(find.RequestID(), "All.Do", err, "Info : Store.Add")
		}
	}

	a.Log(find.RequestID(), "All.Do", "Completed")
	a.Log("mongodocs", "All.Do", "Completed")

	return &coquery.Response{
		Req:  find,
		Data: res,
	}, nil
}
