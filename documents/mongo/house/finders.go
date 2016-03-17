package house

import (
	"github.com/influx6/coquery"
	"github.com/influx6/coquery/storage"
	"github.com/influx6/coquery/utils"
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
