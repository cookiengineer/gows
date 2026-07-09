package gows

import "context"
import "crypto/tls"
import "errors"
import "log"
import "net"
import "net/http"
import "sync"
import "sync/atomic"

type Handler func(*WebSocket)

type Server struct {
	Addr          string
	Handler       Handler
	TLSConfig     *tls.Config
	TLSNextProto  map[string]func(*Server, *tls.Conn, Handler)
	ErrorLog      *log.Logger
	BaseContext   func(net.Listener) context.Context
	ConnContext   func(context.Context, net.Conn) context.Context
	Protocols     *Protocols
	Extensions    []Extension
	Sockets       []*WebSocket
	inShutdown    atomic.Bool
	listeners     map[*net.Listener]struct{}
	listenerGroup sync.WaitGroup
	mutex         sync.Mutex
	onShutdown    []func()
}

func (server *Server) shuttingDown() bool {
	return server.inShutdown.Load()
}

func (server *Server) AddSocket(websocket *WebSocket) {

	found := false

	for s := 0; s < len(server.Sockets); s++ {

		if server.Sockets[s] == websocket {
			found = true
			break
		}

	}

	if found == false {
		server.Sockets = append(server.Sockets, websocket)
	}

}

func (server *Server) Close() error {

	server.inShutdown.Store(true)

	server.mutex.Lock()

	for listener, _ := range server.listeners {
		(*listener).Close()
	}

	for _, callback := range server.onShutdown {
		callback()
	}

	server.mutex.Unlock()
	server.listenerGroup.Wait()

	return ErrServerClosed

}

func (server *Server) RegisterOnShutdown(callback func()) {

	server.mutex.Lock()
	server.onShutdown = append(server.onShutdown, callback)
	server.mutex.Unlock()

}

func (server *Server) RemoveSocket(websocket *WebSocket) {

	server.mutex.Lock()

	for s := 0; s < len(server.Sockets); s++ {

		if server.Sockets[s] == websocket {
			server.Sockets = append(server.Sockets[:s], server.Sockets[s+1:]...)
			break
		}

	}

	server.mutex.Unlock()

}

func (server *Server) Shutdown(ctx context.Context) error {

	// TODO: Implement this, similar to net/http#Server.Shutdown(context.Context)

	return nil

}

func (server *Server) Upgrade(response http.ResponseWriter, request *http.Request) (*WebSocket, error) {

	if server.shuttingDown() {
		return nil, ErrServerClosed
	}

	nonce_key, err0 := verifyUpgradeRequest(request)

	if err0 == nil {

		accept_key := generateAcceptKey(nonce_key)
		accept_version := "13"
		accept_protocol := ""
		accept_extensions := ""
		negotiated_extensions := make([]Extension, 0)

		if server.Protocols != nil && len(*server.Protocols) > 0 {

			protocol := request.Header.Get("Sec-WebSocket-Protocol")

			if protocol != "" {
				accept_protocol = server.Protocols.Negotiate(protocol)
			}

		}

		if len(server.Extensions) > 0 {
			offers := request.Header.Values("Sec-WebSocket-Extensions")
			negotiated_extensions, accept_extensions = negotiateExtensions(offers, server.Extensions)
		}

		hijacker, ok := response.(http.Hijacker)

		if ok == true {

			connection, buffer, err1 := hijacker.Hijack()

			if err1 == nil {

				// Write HTTP 101 Switching Protocols response
				buffer.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
				buffer.WriteString("Upgrade: websocket\r\n")
				buffer.WriteString("Connection: Upgrade\r\n")
				buffer.WriteString("Sec-WebSocket-Accept: " + accept_key + "\r\n")
				buffer.WriteString("Sec-WebSocket-Version: " + accept_version + "\r\n")

				if accept_protocol != "" {
					buffer.WriteString("Sec-WebSocket-Protocol: " + accept_protocol + "\r\n")
				}

				if accept_extensions != "" {
					buffer.WriteString("Sec-WebSocket-Extensions: " + accept_extensions + "\r\n")
				}

				buffer.WriteString("\r\n")
				buffer.Flush()

				websocket := NewWebSocket(connection, server)
				websocket.Extensions = negotiated_extensions

				server.mutex.Lock()
				server.Sockets = append(server.Sockets, websocket)
				server.mutex.Unlock()

				if server.Handler != nil {
					go server.Handler(websocket)
				}

				go websocket.Init()

				return websocket, nil

			} else {
				return nil, err1
			}

		} else {
			return nil, errors.New("websocket: response does not implement http.Hijacker")
		}

	} else {

		http.Error(response, err0.Error(), http.StatusBadRequest)

		return nil, err0

	}

}

func (server *Server) Listen() error {

	address := server.Addr

	if address == "" {
		address = ":http"
	}

	listener, err := net.Listen("tcp", address)

	if err == nil {
		return server.Serve(listener)
	} else {
		return err
	}

}

func (server *Server) Serve(listener net.Listener) error {

	server.mutex.Lock()

	if server.listeners == nil {
		server.listeners = make(map[*net.Listener]struct{})
	}

	server.listeners[&listener] = struct{}{}
	server.mutex.Unlock()

	defer func() {
		server.mutex.Lock()
		delete(server.listeners, &listener)
		server.mutex.Unlock()
	}()

	var base_context context.Context

	if server.BaseContext != nil {
		base_context = server.BaseContext(listener)
	} else {
		base_context = context.Background()
	}

	var tls_config *tls.Config

	if server.TLSConfig != nil {
		tls_config = server.TLSConfig
	}

	http_server := &http.Server{
		Addr: listener.Addr().String(),
		Handler: http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {

			_, err := server.Upgrade(response, request)

			if err != nil {

				if server.ErrorLog != nil {
					server.ErrorLog.Printf("websocket: upgrade error: %v", err)
				}

				return

			}

		}),
		TLSConfig: tls_config,
		ErrorLog:  server.ErrorLog,
		BaseContext: func(l net.Listener) context.Context {
			return base_context
		},
		ConnContext: server.ConnContext,
	}

	return http_server.Serve(listener)

}

func (server *Server) ServeTLS(listener net.Listener, cert_file string, key_file string) error {

	config := &tls.Config{}

	if server.TLSConfig != nil {
		config = server.TLSConfig.Clone()
	}

	config_has_cert := len(config.Certificates) > 0 || config.GetCertificate != nil || config.GetConfigForClient != nil

	if config_has_cert == false && cert_file != "" && key_file != "" {

		tmp, err := tls.LoadX509KeyPair(cert_file, key_file)

		if err == nil {

			config.Certificates = make([]tls.Certificate, 1)
			config.Certificates = append(config.Certificates, tmp)

		} else {
			return err
		}

	}

	server.TLSConfig = config

	tls_listener := tls.NewListener(listener, config)

	return server.Serve(tls_listener)

}
