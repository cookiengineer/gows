package gows

import "crypto/rand"
import "encoding/binary"

type Packet struct {
	Operation Operation  `json:"operation"`
	Type      SocketType `json:"type"`
	Mask      []byte     `json:"mask"`
	Payload   []byte     `json:"payload"`
	Reserved  [3]bool    `json:"reserved"`
	Final     bool       `json:"final"`
}

func ParsePacket(frame []byte) (Packet, error) {

	packet := &Packet{}
	err := packet.Unmarshal(frame)

	if err == nil {
		return *packet, nil
	} else {
		return *packet, err
	}

}

func (packet *Packet) Unmarshal(frame []byte) error {

	if len(frame) >= 2 {

		// First byte: FIN (1 bit), RSV1-3 (3 bits), Opcode (4 bits)
		byte0 := frame[0]
		packet.Final = (byte0 & 0x80) != 0
		packet.Reserved[0] = (byte0 & 0x40) != 0
		packet.Reserved[1] = (byte0 & 0x20) != 0
		packet.Reserved[2] = (byte0 & 0x10) != 0
		packet.Operation = Operation(byte0 & 0x0F)

		// RFC 6455 Section 5.2: RSV bit validation is done at the
		// WebSocket level where negotiated extensions are known.

		// RFC 6455 Section 5.2: unknown opcodes MUST fail the connection
		if packet.Operation.IsValid() == true {

			// Second byte: MASK (1 bit), Payload Length (7 bits)
			byte1 := frame[1]
			is_masked := (byte1 & 0x80) != 0
			length := uint64(byte1 & 0x7F)
			offset := 2

			// Extended payload length (16-bit or 64-bit)
			if length == 126 {

				if len(frame) < offset+2 {
					return ErrPacketIncomplete
				}

				length = uint64(binary.BigEndian.Uint16(frame[offset : offset+2]))
				offset += 2

			} else if length == 127 {

				if len(frame) < offset+8 {
					return ErrPacketIncomplete
				}

				length = binary.BigEndian.Uint64(frame[offset : offset+8])

				// RFC 6455 Section 5.2: most significant bit of 64-bit length MUST be 0
				if length&0x8000000000000000 != 0 {
					return ErrPacketTooLarge
				}

				offset += 8

			}

			if packet.Operation.IsControl() {

				if length > 125 {
					return ErrPacketControlFrameTooLarge
				}

				if packet.Final == false {
					return ErrPacketControlFrameFragmented
				}

			}

			if is_masked == true {

				if len(frame) < offset+4 {
					return ErrPacketIncomplete
				}

				packet.Mask = make([]byte, 4)
				copy(packet.Mask, frame[offset:offset+4])
				offset += 4

			} else {
				packet.Mask = nil
			}

			if uint64(len(frame)-offset) == length {

				payload := make([]byte, length)

				if length > 0 {
					copy(payload, frame[offset:offset+int(length)])
				}

				if is_masked == true && len(packet.Mask) == 4 {

					for p, _ := range payload {
						payload[p] ^= packet.Mask[p%4]
					}

				}

				packet.Payload = payload

				return nil

			} else {
				return ErrPacketIncomplete
			}

		} else {
			return ErrPacketInvalidOperation
		}

	} else {
		return ErrPacketIncomplete
	}

}

func (packet *Packet) Marshal() ([]byte, error) {

	var header []byte
	var mask [4]byte

	length := uint64(len(packet.Payload))

	// Validate control frame constraints
	if packet.Operation.IsControl() {

		if length > 125 {
			return nil, ErrPacketControlFrameTooLarge
		}

		packet.Final = true

	}

	// client frames MUST be masked, server frames MUST NOT (RFC 6455 Section 5.3)
	is_masked := packet.Type == SocketTypeClient

	if is_masked == true {

		_, err := rand.Read(mask[:])

		if err != nil {
			return nil, err
		}

		packet.Mask = make([]byte, 4)
		copy(packet.Mask, mask[:])

	} else {

		packet.Mask = nil

	}

	// First byte: FIN + RSV1-3 + Opcode
	byte0 := byte(packet.Operation & 0x0F)

	if packet.Final {
		byte0 |= 0x80
	}

	if packet.Reserved[0] {
		byte0 |= 0x40
	}

	if packet.Reserved[1] {
		byte0 |= 0x20
	}

	if packet.Reserved[2] {
		byte0 |= 0x10
	}

	// Second byte: MASK + Payload Length + Extended Payload Length
	if length <= 125 {

		byte1 := byte(length)

		if is_masked == true {
			byte1 |= 0x80
		}

		header = append(header, byte0)
		header = append(header, byte1)

	} else if length <= 65535 {

		byte1 := byte(126)

		if is_masked == true {
			byte1 |= 0x80
		}

		header = append(header, byte0)
		header = append(header, byte1)

		ext := make([]byte, 2)
		binary.BigEndian.PutUint16(ext, uint16(length))
		header = append(header, ext...)

	} else {

		byte1 := byte(127)

		if is_masked == true {
			byte1 |= 0x80
		}

		header = append(header, byte0)
		header = append(header, byte1)

		ext := make([]byte, 8)
		binary.BigEndian.PutUint64(ext, length)
		header = append(header, ext...)

	}

	payload := make([]byte, length)
	copy(payload, packet.Payload)

	if is_masked == true {

		header = append(header, mask[:]...)

		for p := range payload {
			payload[p] ^= mask[p%4]
		}

	}

	// Combine header and payload into final frame
	frame := make([]byte, 0, len(header)+len(payload))
	frame = append(frame, header...)
	frame = append(frame, payload...)

	return frame, nil

}
