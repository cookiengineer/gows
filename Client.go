package gows

import "bufio"
import "crypto/tls"
import "fmt"
import "net"
import net_http "net/http"
import net_url "net/url"
import "strings"

type Client struct {
	Socket     *WebSocket
	TLSConfig  *tls.Config
	URL        *net_url.URL
	nonce      string
	Extensions []Extension
}

func NewClient(raw_url string) (*Client, error) {

	url, err1 := net_url.Parse(raw_url)

	if err1 == nil {

		scheme := strings.ToLower(url.Scheme)

		if scheme == "ws" || scheme == "wss" {

			host := url.Host
			path := url.Path

			if strings.Contains(host, ":") == false {

				if scheme == "ws" {
					host = host + ":80"
				} else if scheme == "wss" {
					host = host + ":443"
				}

			}

			if path == "" {
				path = "/"
			}

			if url.RawQuery != "" {
				path = fmt.Sprintf("%s?%s", path, url.RawQuery)
			}

			nonce_key, err2 := generateNonceKey()

			if err2 == nil {

				parsed_url, err3 := net_url.Parse(fmt.Sprintf("%s://%s%s", scheme, host, path))

				if err3 == nil {

					return &Client{
						Socket:    nil,
						URL:       parsed_url,
						TLSConfig: &tls.Config{},
						nonce:     nonce_key,
					}, nil

				} else {
					return nil, fmt.Errorf("websocket: invalid URL: %s", err3)
				}

			} else {
				return nil, fmt.Errorf("websocket: failed to generate Nonce Key: %s", err2)
			}

		} else {
			return nil, fmt.Errorf("websocket: unsupported URL scheme: %s", url.Scheme)
		}

	} else {
		return nil, fmt.Errorf("websocket: invalid URL: %s", err1)
	}

}

func (client *Client) Connect() error {

	var connection net.Conn = nil
	var request_url string = ""

	if client.URL.Scheme == "wss" {

		tmp, err := tls.Dial("tcp", client.URL.Host, client.TLSConfig)

		if err == nil {

			request_url = fmt.Sprintf("https://%s%s", client.URL.Host, client.URL.Path)
			connection = tmp

		} else {
			return fmt.Errorf("websocket: tls dial failed: %s", err)
		}

	} else if client.URL.Scheme == "ws" {

		tmp, err := net.Dial("tcp", client.URL.Host)

		if err == nil {

			request_url = fmt.Sprintf("http://%s%s", client.URL.Host, client.URL.Path)
			connection = tmp

		} else {
			return fmt.Errorf("websocket: tcp dial failed: %s", err)
		}

	}

	if request_url != "" && connection != nil {

		request, err3 := net_http.NewRequest(net_http.MethodGet, request_url, nil)

		if err3 == nil {

			request.Header.Set("Upgrade", "websocket")
			request.Header.Set("Connection", "Upgrade")
			request.Header.Set("Sec-WebSocket-Key", client.nonce)
			request.Header.Set("Sec-WebSocket-Version", "13")

			if len(client.Extensions) > 0 {
				var offers []string
				for _, ext := range client.Extensions {
					params := ext.Parameters()
					if params != nil {
						offers = append(offers, formatExtensionHeader(ext.Name(), params))
					}
				}
				if len(offers) > 0 {
					request.Header.Set("Sec-WebSocket-Extensions", strings.Join(offers, ", "))
				}
			}

			err4 := request.Write(connection)

			if err4 == nil {

				pipe := &pipe_connection{Conn: connection}
				reader := bufio.NewReader(pipe)
				response, err5 := net_http.ReadResponse(reader, request)

				if err5 == nil {

					err6 := verifyUpgradeResponse(client.nonce, response)

					if err6 == nil {

						extension_header := response.Header.Get("Sec-WebSocket-Extensions")
						negotiated_extensions := make([]Extension, 0)

						if extension_header != "" && len(client.Extensions) > 0 {

							for _, element := range strings.Split(extension_header, ",") {

								name, parameters := parseExtensionHeader(strings.TrimSpace(element))

								if name == "" {
									continue
								}

								for _, extension := range client.Extensions {

									if extension.Name() == name {

										if extension.Accept(parameters) {
											negotiated_extensions = append(negotiated_extensions, extension)
										}

										break

									}

								}

							}

							if len(negotiated_extensions) == 0 {
								connection.Close()
								return ErrExtensionNegotiationFailed
							}

						}

						remaining := reader.Buffered()

						if remaining > 0 {
							// Recover remaining WebSocket frames
							pipe.buffer = make([]byte, remaining)
							reader.Read(pipe.buffer)
						}

						client.Socket = NewWebSocket(net.Conn(pipe), nil)

						if len(negotiated_extensions) > 0 {
							client.Socket.Extensions = negotiated_extensions
						}

						return nil

					} else {

						defer connection.Close()
						return fmt.Errorf("websocket: upgrade failed: %s", err6)

					}

				} else {

					defer connection.Close()
					return fmt.Errorf("websocket: failed to read response: %s", err5)

				}

			} else {

				defer connection.Close()
				return fmt.Errorf("websocket: failed to write request: %s", err4)

			}

		} else {

			defer connection.Close()
			return fmt.Errorf("websocket: failed to create HTTP request: %s", err3)

		}

	} else {
		return fmt.Errorf("websocket: net dial failed")
	}

}
