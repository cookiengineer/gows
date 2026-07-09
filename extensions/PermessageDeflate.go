package extensions

import "github.com/cookiengineer/gows"
import "bytes"
import "compress/flate"
import "compress/zlib"
import "io"
import "strconv"

var _ gows.Extension = (*PermessageDeflate)(nil)

// PermessageDeflate implements the "permessage-deflate" WebSocket extension as defined in RFC 7692.
type PermessageDeflate struct {
	ServerNoContextTakeover bool
	ClientNoContextTakeover bool
	ServerMaxWindowBits     int // 8-15, 0 means default (15 = 32KB)
	ClientMaxWindowBits     int // 8-15, 0 means default (15 = 32KB)
	CompressionLevel        int // 0 (none) to 9 (best)
	compress_buffer         *bytes.Buffer
	decompress_buffer       *bytes.Buffer
}

func NewPermessageDeflate() *PermessageDeflate {

	return &PermessageDeflate{
		ClientNoContextTakeover: true,
		ServerNoContextTakeover: true,
		ClientMaxWindowBits:     15,
		ServerMaxWindowBits:     15,
		CompressionLevel:        flate.DefaultCompression,
		compress_buffer:         &bytes.Buffer{},
		decompress_buffer:       &bytes.Buffer{},
	}

}

func (extension *PermessageDeflate) Name() string {
	return "permessage-deflate"
}

func (extension *PermessageDeflate) Parameters() map[string]string {

	params := make(map[string]string)

	if extension.ClientNoContextTakeover {
		params["client_no_context_takeover"] = ""
	}

	if extension.ClientMaxWindowBits > 0 {
		params["client_max_window_bits"] = strconv.Itoa(extension.ClientMaxWindowBits)
	} else {
		params["client_max_window_bits"] = ""
	}

	if extension.ServerNoContextTakeover {
		params["server_no_context_takeover"] = ""
	}

	if extension.ServerMaxWindowBits > 0 {
		params["server_max_window_bits"] = strconv.Itoa(extension.ServerMaxWindowBits)
	}

	return params

}

func (extension *PermessageDeflate) Negotiate(offer map[string]string) (map[string]string, bool) {

	agreed := make(map[string]string)

	// client_max_window_bits: client informs its max window (hint for server)
	if value, ok := offer["client_max_window_bits"]; ok {

		if value == "" {

			// Client supports it, but no specific limit requested.
			// Server may include a limit in response.
			if extension.ClientMaxWindowBits > 0 {
				agreed["client_max_window_bits"] = strconv.Itoa(extension.ClientMaxWindowBits)
			}

		} else {

			// Client specified a limit; server must respect it
			bits, err := strconv.Atoi(value)

			if err != nil || bits < 8 || bits > 15 {
				return nil, false
			}

			// Apply the more restrictive limit
			if extension.ClientMaxWindowBits == 0 || bits < extension.ClientMaxWindowBits {
				extension.ClientMaxWindowBits = bits
			}

			agreed["client_max_window_bits"] = strconv.Itoa(extension.ClientMaxWindowBits)

		}

	}

	// server_no_context_takeover: client requests server not to use context takeover
	if _, ok := offer["server_no_context_takeover"]; ok {

		extension.ServerNoContextTakeover = true
		agreed["server_no_context_takeover"] = ""

	}

	// server_max_window_bits: client limits server's window size
	if value, ok := offer["server_max_window_bits"]; ok {

		bits, err := strconv.Atoi(value)

		if err != nil || bits < 8 || bits > 15 {
			return nil, false
		}

		extension.ServerMaxWindowBits = bits
		agreed["server_max_window_bits"] = value

	}

	// client_no_context_takeover: client hints it won't use context takeover
	if _, ok := offer["client_no_context_takeover"]; ok {

		if extension.ClientNoContextTakeover {
			agreed["client_no_context_takeover"] = ""
		}

	}

	return agreed, true

}

func (extension *PermessageDeflate) Accept(response map[string]string) bool {

	// client_no_context_takeover: server requests client not to use context takeover
	if _, ok := response["client_no_context_takeover"]; ok {
		extension.ClientNoContextTakeover = true
	}

	// client_max_window_bits: server limits client's window size
	if value, ok := response["client_max_window_bits"]; ok {

		bits, err := strconv.Atoi(value)

		if err != nil || bits < 8 || bits > 15 {
			return false
		}

		extension.ClientMaxWindowBits = bits

	}

	// server_no_context_takeover: server confirms it won't use context takeover
	if _, ok := response["server_no_context_takeover"]; ok {
		extension.ServerNoContextTakeover = true
	}

	// server_max_window_bits: server confirms its window size limit
	if value, ok := response["server_max_window_bits"]; ok {

		bits, err := strconv.Atoi(value)

		if err != nil || bits < 8 || bits > 15 {
			return false
		}

		extension.ServerMaxWindowBits = bits

	}

	return true

}

func (extension *PermessageDeflate) Compress(payload []byte) ([]byte, error) {

	if len(payload) == 0 {
		return []byte{}, nil
	}

	extension.compress_buffer.Reset()

	writer, err := zlib.NewWriterLevel(extension.compress_buffer, extension.CompressionLevel)

	if err != nil {
		return nil, err
	}

	_, err = writer.Write(payload)

	if err != nil {
		writer.Close()
		return nil, err
	}

	// Close writes final deflate block (BFINAL=1), empty stored block
	// for byte alignment (0x00 0x00 0xff 0xff), and zlib Adler-32 trailer
	err = writer.Close()

	if err != nil {
		return nil, err
	}

	raw := extension.compress_buffer.Bytes()

	// Strip zlib header (2 bytes) and Adler-32 trailer (4 bytes) to get raw DEFLATE
	if len(raw) <= 6 {
		return []byte{}, nil
	}

	deflateData := raw[2 : len(raw)-4]

	// RFC 7692 Section 7.2.1: append empty stored block if needed, then strip 4 bytes.
	// Go's zlib already appends the empty stored block; we just strip it.
	if len(deflateData) < 4 ||
		deflateData[len(deflateData)-4] != 0x00 ||
		deflateData[len(deflateData)-3] != 0x00 ||
		deflateData[len(deflateData)-2] != 0xff ||
		deflateData[len(deflateData)-1] != 0xff {

		deflateData = append(deflateData, 0x00, 0x00, 0xff, 0xff)

	}

	deflateData = deflateData[:len(deflateData)-4]

	return deflateData, nil

}

func (extension *PermessageDeflate) Decompress(payload []byte) ([]byte, error) {

	if len(payload) == 0 {
		return []byte{}, nil
	}

	// RFC 7692 Section 7.2.2: Append 0x00 0x00 0xff 0xff to reconstruct
	// the empty stored block that was stripped during compression
	payload = append(payload, 0x00, 0x00, 0xff, 0xff)

	// flate.NewReader works on raw DEFLATE data. It reads until the
	// final block (BFINAL=1) and ignores any trailing data.
	reader := flate.NewReader(bytes.NewReader(payload))
	defer reader.Close()

	extension.decompress_buffer.Reset()

	_, err := io.Copy(extension.decompress_buffer, reader)

	if err != nil {
		return nil, err
	}

	result := make([]byte, extension.decompress_buffer.Len())
	copy(result, extension.decompress_buffer.Bytes())

	return result, nil

}
