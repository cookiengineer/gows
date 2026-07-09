package gows

import "strings"

// parseExtensionHeader parses a single Sec-WebSocket-Extensions header element
// like "permessage-deflate; server_no_context_takeover; server_max_window_bits=10"
// into the extension name and a parameter map.
func parseExtensionHeader(header string) (name string, params map[string]string) {

	params = make(map[string]string)

	parts := strings.SplitN(strings.TrimSpace(header), ";", 2)
	name = strings.TrimSpace(parts[0])

	if len(parts) == 2 {

		for _, param := range strings.Split(parts[1], ";") {

			param = strings.TrimSpace(param)

			if param == "" {
				continue
			}

			kv := strings.SplitN(param, "=", 2)

			key := strings.TrimSpace(kv[0])

			if len(kv) == 2 {
				params[key] = strings.TrimSpace(kv[1])
			} else {
				params[key] = ""
			}

		}

	}

	return name, params

}

