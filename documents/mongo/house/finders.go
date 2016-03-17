package house

import (
	"fmt"

	"github.com/influx6/coquery"
	"github.com/influx6/coquery/storage"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

//==========================================================================================

// FindProc provides a find working for handling find requests.
type FindProc struct {
	EventLog
	Mongo Mongo
	Query Query
	Store storage.Store
}

// Name returns the Id of this operation provider.
func (f *FindProc) Name() string {
	return "find"
}

// Do performs the necessary tasks passed to FindProc
func (f *FindProc) Do(data interface{}, err error) (interface{}, error) {
	f.Log("MongoProvider.FindProc", "Do", "Started : %s", f.Query.Query(data))

	if err != nil {
		f.Error("MongoProvider.FindProc", "Do", err, "Completed")
		return nil, err
	}

	find, ok := data.(*coquery.Find)
	if !ok {
		f.Error("MongoProvider.FindProc", "Do", coquery.ErrInvalidRequestType, "Completed")
		return nil, coquery.ErrInvalidRequestType
	}

	var res coquery.Parameters

	if records, err := f.Store.GetByRef(find.Key, fmt.Sprintf("%s", find.Value)); err == nil {

		for _, recs := range records {
			res = append(res, coquery.Parameter(recs))
		}

		f.Log("MongoProvider.FindProc", "Do", "Completed")
		return &coquery.Response{
			Req:  find,
			Data: res,
		}, nil
	}

	fn := func(c *mgo.Collection) error {
		q := bson.M{find.Key: find.Value}
		// q[find.Key] = find.Value
		f.Log("MongoProvider.FindProc", "DBAction", "db.%s.find(%s)", c.Name, f.Query.Query(q))
		return c.Find(q).All(&res)
	}

	err = f.Mongo.ExecuteDB("MongoProvider.FindProc", find.Doc, fn)
	if err != nil {
		f.Error("MongoProvider.FindProc", "Do", err, "Completed")
		return nil, &MError{Rid: find.RID, Msg: "FindProc Failed", IError: err}
	}

	for _, record := range res {
		f.Store.AddRef((map[string]interface{})(record), find.Key)
	}

	f.Log("MongoProvider.FindProc", "Do", "Completed")

	return &coquery.Response{
		Req:  find,
		Data: res,
	}, nil
}

//==========================================================================================
