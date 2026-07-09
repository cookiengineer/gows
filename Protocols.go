package gows

import "strings"

type Protocols []string

func (protocols Protocols) Negotiate(requested string) string {

	if requested != "" {

		result := ""

		for _, requested_protocol := range strings.Split(requested, ",") {

			requested_protocol = strings.TrimSpace(requested_protocol)

			for _, supported_protocol := range protocols {

				if strings.EqualFold(requested_protocol, supported_protocol) {
					result = supported_protocol
					break
				}

			}

		}

		return result

	} else {
		return ""
	}

}
