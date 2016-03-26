package mongodocs

import (
	"github.com/influx6/coquery"
	"github.com/influx6/coquery/storage"
	"github.com/influx6/coquery/utils"
	"gopkg.in/mgo.v2/bson"
)

//==========================================================================================

// Find provides a find working for handling find requests.
type Find struct {
	EventLog
	Db    DB
	Store storage.Store
}

// Do performs the necessary tasks passed to FindProc
func (f *Find) Do(data interface{}, err error) (interface{}, error) {
	f.Log("mongodocs.Find", "Do", "Started : %s", utils.Query.Query(data))

	if err != nil {
		f.Error("mongodocs.Find", "Do", err, "Completed")
		return nil, err
	}

	req, ok := data.(*coquery.Request)
	if !ok {
		f.Error("mongodocs.Find", coquery.ErrInvalidRequestType, "Completed")
		return nil, coquery.ErrInvalidRequestType
	}

	find, ok := req.R.(*coquery.Find)
	if !ok {
		f.Error(req.R.RequestID(), "Do", coquery.ErrInvalidRequestType, "Completed")
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
		f.Error(req.R.RequestID(), "Do", coquery.ErrInvalidRequestType, "Completed")
	}

	if found {
		for _, recs := range records {
			res = append(res, coquery.Parameter(recs))
		}

		f.Log(find.RequestID(), "Do", "Info : Store : Record Found")

		f.Log(find.RequestID(), "Do", "Completed")
		return &coquery.Response{
			Req:  find,
			Data: res,
		}, nil
	}

	db, session, err := f.Db.New(find.RequestID())
	if err != nil {
		f.Error(find.RequestID(), "db.New", err, "Completed : New Session")
		return nil, &MError{Rid: find.RID, Msg: "New Session Failed", IError: err}
	}

	defer session.Close()

	q := bson.M{find.Key: val}
	f.Log(find.RequestID(), find.RequestID(), "DBAction : db.%s.find(%s)", find.Doc, utils.Query.Query(q))

	if err := db.C(find.Doc).Find(q).All(&res); err != nil {
		f.Error(find.RequestID(), "DBAction", err, "Completed")
		return nil, &MError{Rid: find.RID, Msg: "FindProc Failed", IError: err}
	}

	f.Log(find.RequestID(), "Do", "Info : Response : %s", utils.Query.Query(res))

	for _, record := range res {
		if err := f.Store.AddRef((map[string]interface{})(record), find.Key); err != nil {
			f.Error(find.RequestID(), "Do", err, "Info : Store.AddRef : Key[%s]", find.Key)
		}
	}

	f.Log(find.RequestID(), "Do", "Completed")

	return &coquery.Response{
		Req:  find,
		Data: res,
	}, nil
}

//==========================================================================================
