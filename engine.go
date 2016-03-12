package coquery

// Documents defines a interface that defines a means for registering
// document providers for request processing.
type Documents interface {
	Document(string)
}

// Engine defines a interface for a coquery service providers.
type Engine interface {
	Route() Documents
}

// PathEngine defines a coquery engine system for routing and management of
// coquery requests.
type PathEngine struct {
}
