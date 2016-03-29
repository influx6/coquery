package crossdocs

//==============================================================================

// Events defines event logger that allows us to record events for a specific
// action that occured.
type Events interface {
	Log(context interface{}, name string, message string, data ...interface{})
	Error(context interface{}, name string, err error, message string, data ...interface{})
}

//==============================================================================
