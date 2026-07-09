package gows

type Operation byte

// https://www.rfc-editor.org/rfc/rfc6455.txt

const (
	OperationContinue Operation = 0x00
	OperationText     Operation = 0x01
	OperationBinary   Operation = 0x02
	// Reserved Control Frames: 0x03-0x07
	OperationClose Operation = 0x08
	OperationPing  Operation = 0x09
	OperationPong  Operation = 0x0a
	// Reserved Control Frames: 0x0b-0x0f
)

func (operation Operation) IsControl() bool {

	if operation&0x08 != 0 {
		return true
	}

	return false

}

func (operation Operation) IsValid() bool {

	switch operation {
	case OperationContinue:
		return true
	case OperationText:
		return true
	case OperationBinary:
		return true
	case OperationClose:
		return true
	case OperationPing:
		return true
	case OperationPong:
		return true
	}

	return false

}
