package db

// Db defines an interface that defines base level methods for dbs.
type Db interface {
	Shutdown(context interface{})
}
