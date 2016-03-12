package mongo

import (
	"errors"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/influx6/coquery"
	"github.com/influx6/coquery/storage"
)

//==========================================================================================

// Logger provides a logging interface for the coquery package, allowing
// users to provide their own internal logging systems.
type Logger interface {
	User(context interface{}, funcName string, message string, format ...interface{})
	Dev(context interface{}, funcName string, message string, format ...interface{})
	Error(context interface{}, funcName string, err error, message string, format ...interface{})
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

// FindProc provides a find working for handling find requests.
type FindProc struct {
	Log   Logger
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
	f.Log.Dev("MongoProvider.FindProc", "Do", "Started : %s", f.Query.Query(data))

	if err != nil {
		f.Log.Error("MongoProvider.FindProc", "Do", err, "Completed")
		return nil, err
	}

	find, ok := data.(*coquery.Find)
	if !ok {
		err = errors.New("Invalid Query Type")
		f.Log.Error("MongoProvider.FindProc", "Do", err, "Completed")
		return nil, err
	}

	var res coquery.Parameters

	fn := func(c *mgo.Collection) error {
		q := bson.M{}
		q[find.Key] = find.Value
		f.Log.Dev("MongoProvider.FindProc", "DBAction", "db.%s.find(%s)", f.Query.Query(q))
		return c.Find(q).All(&res)
	}

	err = f.Mongo.ExecuteDB("MongoProvider.FindProc", find.Doc, fn)
	if err != nil {
		f.Log.Error("MongoProvider.FindProc", "Do", err, "Completed")
		return nil, &coquery.ResponseError{RID: find.RID, Message: "FindProc Failed", IError: err}
	}

	f.Log.Dev("MongoProvider.FindProc", "Do", "Completed")

	return &coquery.Response{
		Req:  find,
		RID:  find.RID,
		Data: res,
	}, nil
}
