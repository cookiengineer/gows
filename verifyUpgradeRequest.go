package gows

import "errors"
import "fmt"
import "net/http"

func verifyUpgradeRequest(request *http.Request) (string, error) {

	if request.Method == http.MethodGet {

		if containsHeader(request.Header, "Upgrade", "websocket") {

			if containsHeader(request.Header, "Connection", "Upgrade") {

				if containsHeader(request.Header, "Sec-WebSocket-Version", "13") {

					nonce_key := request.Header.Get("Sec-WebSocket-Key")

					if nonce_key != "" {

						return nonce_key, nil

					} else {
						return "", errors.New("websocket: invalid Sec-WebSocket-Key header")
					}

				} else {
					return "", errors.New("websocket: unsupported version")
				}

			} else {
				return "", errors.New("websocket: invalid Connection header")
			}

		} else {
			return "", errors.New("websocket: invalid Upgrade header")
		}

	} else {
		return "", fmt.Errorf("websocket: unexpected HTTP method: %s", request.Method)
	}

}
