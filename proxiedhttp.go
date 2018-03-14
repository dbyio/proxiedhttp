/*
Package proxiedhttp implements a wrapper for the net/http package to support
HAProxy's proxy protocol v2.

Reference: https://www.haproxy.org/download/1.9/doc/proxy-protocol.txt
*/
package proxiedhttp

import (
	"bufio"
	"errors"
	"net"
	"time"
)

var (
	v2sig = []byte{0x0D, 0x0A, 0x0D, 0x0A, 0x00, 0x0D, 0x0A, 0x51, 0x55, 0x49, 0x54, 0x0A}

	// ErrUnsupportedProto for anything but TCP connections
	ErrUnsupportedProto = errors.New("Unsupported protocol")

	// ErrMalformedProxyHeader if proxy protocol header is malformed
	ErrMalformedProxyHeader = errors.New("Malformed proxy protocol header")
)

type proxyHdrV2 struct {
	Sig    [12]byte
	VerCmd uint8
	Fam    uint8
	Len    uint16
}

type ip4 struct {
	SrcAddr [4]uint8
	DstAddr [4]uint8
	SrcPort uint16
	DstPort uint16
}

type ip6 struct {
	SrcAddr [16]uint8
	DstAddr [16]uint8
	SrcPort uint16
	DstPort uint16
}

// Listener wraps the main net.Listener, whose connection may use the underlying PROXY protocol v2.
// If the communication reies on the procotol, the net.Conn's RemoteAddr() will return the remote source address.
//
// Optionally define ProxyHeaderTimeout to set a maximum time to receive the Proxy Protocol Header. Zero means no timeout
// Optionnally define authSources with a list of authorized proxy source addresses (recommanded)
type Listener struct {
	Listener    net.Listener
	ReadTimeout time.Duration
	AuthSources []net.IP // a list of authorized proxy addresses
}

// Accept waits for and returns the next connection to the listener, after retrieving PROXY protocol data if present.
func (l *Listener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	var useRemAddr bool
	if l.AuthSources != nil {
		rHost, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
		rAddr := net.ParseIP(rHost)
		for _, addr := range l.AuthSources {
			if addr.Equal(rAddr) {
				useRemAddr = true
				break
			}
		}
	} else {
		useRemAddr = true
	}

	proxyConn := &Conn{
		Conn:             conn,
		useRemoteAddress: useRemAddr,
		Reader:           bufio.NewReader(conn),
		readTimeout:      l.ReadTimeout,
	}

	return proxyConn, nil
}

// Close closes the listener.
func (l *Listener) Close() error {
	return l.Listener.Close()
}

// Addr returns the underlying listener's network address.
func (l *Listener) Addr() net.Addr {
	return l.Listener.Addr()
}
