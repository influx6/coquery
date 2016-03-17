package house

import "gopkg.in/mgo.v2"

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
