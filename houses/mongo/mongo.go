package mongo

import (
	"fmt"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/influx6/coquery"
	"github.com/influx6/coquery/storage"
)

//==========================================================================================

// EventLog defines event logger that allows us to record events for a specific
// action that occured.
type EventLog interface {
	Log(context interface{}, name string, message string, data ...interface{})
	Error(context interface{}, name string, err error, message string, data ...interface{})
}

//==========================================================================================

// Mongo provides a interface that provides a execution method for a mongodb DB.
type Mongo interface {
	ExecuteDB(context interface{}, collectionName string, fn func(*mgo.Collection) error) error
}

//==========================================================================================

// Query provides a data stringifier which turns a giving argument into a
// stringed version.
type Query interface {
	Query(interface{}) string
	QueryIndent(interface{}) string
}

//==========================================================================================

// MError provides a custom error message for requests types.
type MError struct {
	Rid    string `json:"rid" bson:"rid"`
	Msg    string `json:"message" bson:"message"`
	IError error  `json:"error" bson:"error"`
}

// Message returns the internal message for this error
func (r MError) Message() string {
	return r.Msg
}

// RequestID returns the response error requestID
func (r MError) RequestID() string {
	return r.Rid
}

// Error returns the error message for this response error.
func (r MError) Error() string {
	if r.IError != nil {
		return r.Rid + " : " + r.Msg + " : " + r.IError.Error()
	}

	return r.Rid + " : " + r.Msg
}

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
		q := bson.M{}
		q[find.Key] = find.Value
		f.Log("MongoProvider.FindProc", "DBAction", "db.%s.find(%s)", c.Name, f.Query.Query(q))
		return c.Find(q).All(&res)
	}

	err = f.Mongo.ExecuteDB("MongoProvider.FindProc", find.Doc, fn)
	if err != nil {
		f.Error("MongoProvider.FindProc", "Do", err, "Completed")
		return nil, &MError{Rid: find.RID, Msg: "FindProc Failed", IError: err}
	}

	for _, record := range res {
		f.Store.Add((map[string]interface{})(record))
	}

	f.Log("MongoProvider.FindProc", "Do", "Completed")

	return &coquery.Response{
		Req:  find,
		Data: res,
	}, nil
}
