package client

import "io"

//==============================================================================

// ServeTransport defines an interface for requests transport, which allows us
// build custom transports based on different low-level systems(HTTP,Websocket).
type ServeTransport interface {
	Do(endpoint string, body io.Reader) (io.Reader, error)
}

// Server provides a central request manager for different query requests and
// subscriptions.
type Server interface {
	Register(query string) Requestor
	serve(query string, client Requestor) error
}

//==============================================================================
