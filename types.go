package coquery

//==============================================================================

// Parameter defines the basic data type for all data received from the
// providers.
type Parameter map[string]interface{}

// Parameters defines a lists of Parameter types.
type Parameters []Parameter

// Response provides a response struct for replies to coquery requests.
type Response struct {
	Req  RecordRequest `json:"-" bson:"-"`
	ID   string        `json:"id" bson:"id"`
	Data Parameters    `json:"reply" bson:"reply"`
}

//==============================================================================

// RecordRequest defines a base type for the supported request types
type RecordRequest interface {
	Name() string
}

// RecordReplyChannel defines a channel type that provides the response for a
// coquery RecordRequest.
type RecordReplyChannel chan Response

//==============================================================================

// Find defines a record retrieve request based on the KV query.
type Find struct {
	Doc   string
	ID    string
	Key   string
	Value interface{}
	Next  []string // the next query to be executed after this query
	Reply RecordReplyChannel
}

// Name returns the name for the giving request type.
func (f Find) Name() string {
	return "find"
}

//==============================================================================

// Collect retrieves specific keyed items from the coquery stores.
type Collect struct {
	ID    string
	Keys  []string
	Next  []string // the next query to be executed after this query
	Reply RecordReplyChannel
}

// Name returns the name for the giving request type.
func (f Collect) Name() string {
	return "collect"
}

//==============================================================================

// Mutate provides json data to be saved/augmented into a new version of the
// current document.
type Mutate struct {
	ID    string
	Data  []byte
	Next  []string // the next query to be executed after this query
	Reply RecordReplyChannel
}

// Name returns the name for the giving request type.
func (f Mutate) Name() string {
	return "mutate"
}

//==============================================================================
