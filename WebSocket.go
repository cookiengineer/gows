package gows

import "encoding/binary"
import "io"
import "net"
import "sync"
import "time"
import "unicode/utf8"

type WebSocket struct {
	Connection          net.Conn     `json:"connection"`
	Frames              []Packet     `json:"frames"`
	Type                SocketType   `json:"type"`
	Server              *Server      `json:"server"`
	Extensions          []Extension  `json:"-"`
	OnMessage           func([]byte) `json:"-"`
	OnClose             func()       `json:"-"`
	fragment_operation  Operation
	fragment_payload    []byte
	fragment_compressed bool
	is_closed           bool
	is_destroyed        bool
	mutex               sync.Mutex
}

func NewWebSocket(connection net.Conn, server *Server) *WebSocket {

	typ := SocketTypeClient

	if server != nil {
		typ = SocketTypeServer
	}

	return &WebSocket{
		Connection:         connection,
		Frames:             make([]Packet, 0),
		Type:               typ,
		Server:             server,
		OnMessage:          nil,
		OnClose:            nil,
		fragment_operation: Operation(0),
		fragment_payload:   nil,
		is_closed:          false,
		is_destroyed:       false,
		mutex:              sync.Mutex{},
	}

}

func (websocket *WebSocket) Destroy() {

	websocket.mutex.Lock()

	if websocket.is_destroyed == false {

		websocket.is_destroyed = true

		if websocket.Connection != nil {
			websocket.Connection.Close()
			websocket.Connection = nil
		}

		websocket.mutex.Unlock()

		if websocket.Server != nil {
			websocket.Server.RemoveSocket(websocket)
		}

		if websocket.OnClose != nil {
			websocket.OnClose()
		}

	} else {
		websocket.mutex.Unlock()
	}

}

func (websocket *WebSocket) Init() {

FrameLoop:
	for {

		frame, err0 := websocket.readFrame()

		if err0 == nil {

			is_masked := (frame[1] & 0x80) != 0

			if websocket.Type == SocketTypeServer && is_masked == false {

				websocket.Close(StatusProtocolError, "received unmasked frame")
				break FrameLoop

			} else if websocket.Type == SocketTypeClient && is_masked == true {

				websocket.Close(StatusProtocolError, "received masked frame")
				break FrameLoop

			}

			packet, err1 := ParsePacket(frame)

			if err1 == nil {

				err_rsv := websocket.validateRSV(packet)

				if err_rsv != nil {
					websocket.Close(StatusProtocolError, err_rsv.Error())
					break FrameLoop
				}

				packet.Type = websocket.Type

				websocket.mutex.Lock()
				websocket.Frames = append(websocket.Frames, packet)
				websocket.mutex.Unlock()

				switch packet.Operation {

				case OperationClose:

					websocket.handleCloseFrame(packet)
					break FrameLoop

				case OperationPing:

					websocket.handlePingFrame(packet)

				case OperationPong:

					// Do Nothing

				case OperationText, OperationBinary:

					if packet.Final == true {

						if websocket.fragment_payload != nil {
							websocket.Close(StatusProtocolError, "expected continuation frame")
							break FrameLoop
						}

						payload := packet.Payload

						if packet.Reserved[0] && len(websocket.Extensions) > 0 {
							var err0 error
							for _, ext := range websocket.Extensions {
								payload, err0 = ext.Decompress(payload)
								if err0 != nil {
									websocket.Close(StatusProtocolError, "decompression failed: "+err0.Error())
									break FrameLoop
								}
							}
						}

						if packet.Operation == OperationText && utf8.Valid(payload) == false {
							websocket.Close(StatusInvalidPayload, "invalid UTF-8")
							break FrameLoop
						}

						if websocket.OnMessage != nil {
							websocket.OnMessage(payload)
						}

					} else {

						if websocket.fragment_payload != nil {
							websocket.Close(StatusProtocolError, "expected continuation frame")
							break FrameLoop
						}

						websocket.fragment_operation = packet.Operation
						websocket.fragment_compressed = packet.Reserved[0]
						websocket.fragment_payload = make([]byte, len(packet.Payload))
						copy(websocket.fragment_payload, packet.Payload)

					}

				case OperationContinue:

					if websocket.fragment_payload == nil {
						websocket.Close(StatusProtocolError, "unexpected continuation frame")
						break FrameLoop
					}

					if packet.Final == true {

						websocket.fragment_payload = append(websocket.fragment_payload, packet.Payload...)

						payload := websocket.fragment_payload

						if websocket.fragment_compressed && len(websocket.Extensions) > 0 {
							var err0 error
							for _, ext := range websocket.Extensions {
								payload, err0 = ext.Decompress(payload)
								if err0 != nil {
									websocket.Close(StatusProtocolError, "decompression failed: "+err0.Error())
									break FrameLoop
								}
							}
						}

						operation := websocket.fragment_operation

						websocket.fragment_payload = nil
						websocket.fragment_operation = Operation(0)
						websocket.fragment_compressed = false

						if operation == OperationText && utf8.Valid(payload) == false {
							websocket.Close(StatusInvalidPayload, "invalid UTF-8")
							break FrameLoop
						}

						if websocket.OnMessage != nil {
							websocket.OnMessage(payload)
						}

					} else {
						websocket.fragment_payload = append(websocket.fragment_payload, packet.Payload...)
					}

				}

			} else {

				websocket.Close(StatusProtocolError, err1.Error())
				break FrameLoop

			}

		} else {
			break FrameLoop
		}

	}

	time.Sleep(100 * time.Millisecond)

	websocket.Destroy()

}

func (websocket *WebSocket) Close(status Status, reason string) {

	websocket.mutex.Lock()

	if websocket.is_closed == false {

		websocket.mutex.Unlock()

		payload := make([]byte, 2+len(reason))
		binary.BigEndian.PutUint16(payload[0:2], uint16(status))
		copy(payload[2:], reason)

		websocket.SendPacket(Packet{
			Operation: OperationClose,
			Type:      websocket.Type,
			Payload:   payload,
			Final:     true,
		})

		websocket.mutex.Lock()
		websocket.is_closed = true
		websocket.mutex.Unlock()

	} else {
		websocket.mutex.Unlock()
	}

}

func (websocket *WebSocket) handleCloseFrame(packet Packet) {

	websocket.mutex.Lock()

	if websocket.is_closed == false {

		websocket.is_closed = true
		websocket.mutex.Unlock()

		websocket.SendPacket(Packet{
			Operation: OperationClose,
			Type:      websocket.Type,
			Payload:   packet.Payload,
			Final:     true,
		})

	} else {
		websocket.mutex.Unlock()
	}

}

func (websocket *WebSocket) handlePingFrame(packet Packet) {

	websocket.SendPacket(Packet{
		Operation: OperationPong,
		Type:      websocket.Type,
		Payload:   packet.Payload,
		Final:     true,
	})

}

func (websocket *WebSocket) readFrame() ([]byte, error) {

	var header [14]byte

	_, err0 := io.ReadFull(websocket.Connection, header[0:2])

	if err0 == nil {

		is_masked := (header[1] & 0x80) != 0
		length := uint64(header[1] & 0x7f)
		offset := 2

		if length == 126 {

			_, err1 := io.ReadFull(websocket.Connection, header[offset:offset+2])

			if err1 == nil {

				length = uint64(binary.BigEndian.Uint16(header[offset : offset+2]))
				offset += 2

			} else {
				return nil, err1
			}

		} else if length == 127 {

			_, err1 := io.ReadFull(websocket.Connection, header[offset:offset+8])

			if err1 == nil {

				length = binary.BigEndian.Uint64(header[offset : offset+8])
				offset += 8

			} else {
				return nil, err1
			}

		}

		if is_masked == true {

			_, err1 := io.ReadFull(websocket.Connection, header[offset:offset+4])

			if err1 == nil {

				offset += 4

			} else {
				return nil, err1
			}

		}

		payload := make([]byte, length)

		if length > 0 {

			_, err1 := io.ReadFull(websocket.Connection, payload)

			if err1 == nil {

				frame := make([]byte, offset+int(length))
				copy(frame[0:offset], header[0:offset])
				copy(frame[offset:], payload)

				return frame, nil

			} else {
				return nil, err1
			}

		} else {

			frame := make([]byte, offset+int(length))
			copy(frame[0:offset], header[0:offset])
			copy(frame[offset:], payload)

			return frame, nil

		}

	} else {
		return nil, err0
	}

}

func (websocket *WebSocket) validateRSV(packet Packet) error {

	// RFC 6455 Section 5.2: RSV bits MUST be 0 unless extension negotiated
	if !packet.Reserved[0] && !packet.Reserved[1] && !packet.Reserved[2] {
		return nil
	}

	// RSV bits on control frames are always invalid
	if packet.Operation.IsControl() {
		return ErrPacketReservedBits
	}

	// RSV bits on continuation frames: RSV1 must not be set (RFC 7692 Section 6)
	if packet.Operation == OperationContinue && packet.Reserved[0] {
		return ErrPacketReservedBits
	}

	// No extensions negotiated — any RSV bit is an error
	if len(websocket.Extensions) == 0 {
		return ErrPacketReservedBits
	}

	return nil

}

func (websocket *WebSocket) Send(data []byte) error {

	packet := Packet{
		Operation: OperationText,
		Type:      websocket.Type,
		Payload:   data,
		Final:     true,
	}

	return websocket.SendPacket(packet)

}

func (websocket *WebSocket) SendBinary(data []byte) error {

	packet := Packet{
		Operation: OperationBinary,
		Type:      websocket.Type,
		Payload:   data,
		Final:     true,
	}

	return websocket.SendPacket(packet)

}

func (websocket *WebSocket) SendPacket(packet Packet) error {

	websocket.mutex.Lock()
	defer websocket.mutex.Unlock()

	if websocket.is_closed == false && websocket.Connection != nil {

		if !packet.Operation.IsControl() && len(websocket.Extensions) > 0 {

			payload := packet.Payload

			for _, ext := range websocket.Extensions {
				tmp, err2 := ext.Compress(payload)
				if err2 != nil {
					return err2
				}
				payload = tmp
			}

			packet.Payload = payload
			packet.Reserved[0] = true

		}

		frame, err0 := packet.Marshal()

		if err0 == nil {

			websocket.Frames = append(websocket.Frames, packet)

			_, err1 := websocket.Connection.Write(frame)

			if err1 == nil {
				return nil
			} else {
				return err1
			}

		} else {
			return err0
		}

	} else {
		return io.ErrClosedPipe
	}

}
