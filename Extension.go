package gows

type Extension interface {

	// Name returns the extension identifier (e.g., "permessage-deflate").
	Name() string

	// Parameters returns the extension's parameters to include in the
	// client's Sec-WebSocket-Extensions offer header.
	// Returns nil if the extension should not be offered.
	Parameters() map[string]string

	// Negotiate is called by the server with the client's offered parameters.
	// Returns agreed parameters and true if accepted, or nil and false if declined.
	Negotiate(offer map[string]string) (agreed map[string]string, accepted bool)

	// Accept is called by the client with the server's response parameters.
	// Returns true if the client accepts the server's configuration.
	Accept(response map[string]string) bool

	// Compress transforms an outgoing data message payload before framing.
	// Only called for non-control frames when extensions are active.
	Compress(payload []byte) ([]byte, error)

	// Decompress transforms an incoming data message payload after reassembly.
	// Only called for data messages where the first frame had RSV1 set.
	Decompress(payload []byte) ([]byte, error)

}

