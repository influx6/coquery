package coquery

// Logger provides a logging interface for the coquery package, allowing
// users to provide their own internal logging systems.
type Logger interface {
	User(context interface{}, funcName string, message string, format ...interface{})
	Dev(context interface{}, funcName string, message string, format ...interface{})
	Error(context interface{}, funcName string, err error, message string, format ...interface{})
}
