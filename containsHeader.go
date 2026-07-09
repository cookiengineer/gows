package gows

import "net/http"
import "strings"

func containsHeader(headers http.Header, name string, token string) bool {

	for _, value := range headers[http.CanonicalHeaderKey(name)] {

		for _, part := range strings.Split(value, ",") {

			if strings.EqualFold(strings.TrimSpace(part), token) {
				return true
			}

		}

	}

	return false

}
