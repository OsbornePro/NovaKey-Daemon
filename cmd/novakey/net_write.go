// cmd/novakey/io_helpers.go
package main

import "net"

func writeAll(conn net.Conn, b []byte) error {
	for len(b) > 0 {
		n, err := conn.Write(b)
		if err != nil {
			return err
		}
		if n == 0 {
			return nil
		}
		b = b[n:]
	}
	return nil
}

