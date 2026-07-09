package gows

import "strings"

// formatExtensionHeader formats an extension name and parameters into
// a Sec-WebSocket-Extensions header element.
func formatExtensionHeader(name string, params map[string]string) string {

	if len(params) == 0 {
		return name
	}

	var parts []string

	for key, value := range params {

		if value == "" {
			parts = append(parts, key)
		} else {
			parts = append(parts, key+"="+value)
		}

	}

	return name + "; " + strings.Join(parts, "; ")

}

