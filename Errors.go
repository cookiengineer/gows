package gows

import "errors"

var (
	ErrPacketIncomplete             = errors.New("websocket: incomplete packet")
	ErrPacketTooLarge               = errors.New("websocket: packet too large")
	ErrPacketControlFrameFragmented = errors.New("websocket: fragmented control frame")
	ErrPacketControlFrameTooLarge   = errors.New("websocket: control frame too large")
	ErrPacketInvalidOperation       = errors.New("websocket: invalid opcode")
	ErrPacketReservedBits           = errors.New("websocket: reserved bits set without extension")
	ErrServerClosed                 = errors.New("websocket: server closed")
	ErrExtensionNegotiationFailed   = errors.New("websocket: extension negotiation failed")
)
