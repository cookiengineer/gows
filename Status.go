package gows

type Status uint16

// https://www.rfc-editor.org/rfc/rfc6455#section-7.4

const (
	StatusNormalClosure     Status = 1000
	StatusGoingAway         Status = 1001
	StatusProtocolError     Status = 1002
	StatusUnsupportedData   Status = 1003
	StatusNoStatusReceived  Status = 1005
	StatusAbnormalClosure   Status = 1006
	StatusInvalidPayload    Status = 1007
	StatusPolicyViolation   Status = 1008
	StatusMessageTooBig     Status = 1009
	StatusExtensionRequired Status = 1010
	StatusInternalError     Status = 1011
)
