package gows

import "net"

type pipe_connection struct {
	net.Conn
	buffer []byte
}

func (connection *pipe_connection) Read(bytes []byte) (int, error) {

	if len(connection.buffer) > 0 {

		copied := copy(bytes, connection.buffer)
		connection.buffer = connection.buffer[copied:]

		return copied, nil

	} else {

		return connection.Conn.Read(bytes)

	}

}
