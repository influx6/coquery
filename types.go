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
	RID  string        `json:"rid" bson:"rid"`
	Data Parameters    `json:"reply" bson:"reply"`
}

//==============================================================================

// ResponseError provides a custom error message for requests types.
type ResponseError struct {
	RID     string `json:"rid" bson:"rid"`
	Message string `json:"message" bson:"message"`
	IError  error  `json:"error" bson:"error"`
}

// Error returns the error message for this response error.
func (r ResponseError) Error() string {
	if r.IError != nil {
		return r.RID + " : " + r.Message + " : " + r.IError.Error()
	}

	return r.RID + " : " + r.Message
}

//==============================================================================

// Find defines a record retrieve request based on the KV query.
type Find struct {
	Doc   string      `json:"doc" bson:"doc"`
	RID   string      `json:"rid" bson:"rid"`
	Key   string      `json:"key" bson:"key"`
	Value interface{} `json:"value" bson:"value"`
}

// Name returns the name for the giving request type.
func (f Find) Name() string {
	return "find"
}

//==============================================================================

// Collect retrieves specific keyed items from the coquery stores.
type Collect struct {
	RID  string   `json:"rid" bson:"rid"`
	Keys []string `json:"keys" bson:"keys"`
}

// Name returns the name for the giving request type.
func (f Collect) Name() string {
	return "collect"
}

//==============================================================================

// Mutate provides json data to be saved/augmented into a new version of the
// current document.
type Mutate struct {
	RID  string `json:"rid" bson:"rid"`
	Data []byte `json:"mutate" bson:"mutate"`
}

// Name returns the name for the giving request type.
func (f Mutate) Name() string {
	return "mutate"
}

//==============================================================================
