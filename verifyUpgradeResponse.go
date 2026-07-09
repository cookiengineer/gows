package gows

import "errors"
import "fmt"
import "net/http"

func verifyUpgradeResponse(expected_nonce string, response *http.Response) error {

	if response.StatusCode == http.StatusSwitchingProtocols {

		if containsHeader(response.Header, "Upgrade", "websocket") {

			if containsHeader(response.Header, "Connection", "Upgrade") {

				expected_accept := generateAcceptKey(expected_nonce)

				if response.Header.Get("Sec-WebSocket-Accept") == expected_accept {
					return nil
				} else {
					return errors.New("websocket: invalid Sec-WebSocket-Accept header")
				}

			} else {
				return errors.New("websocket: invalid Connection header")
			}

		} else {
			return errors.New("websocket: invalid Upgrade header")
		}

	} else {
		return fmt.Errorf("websocket: unexpected HTTP status: %s", response.Status)
	}

}
