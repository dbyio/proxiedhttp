package proxiedhttp

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"sync"
	"time"
)

// Conn extends the standard net.Conn type with additional information extracted from the proxy data
type Conn struct {
	net.Conn
	Reader           *bufio.Reader
	srcAddr          *net.TCPAddr
	dstAddr          *net.TCPAddr
	useRemoteAddress bool
	readTimeout      time.Duration
	once             sync.Once
}

// Read scans for proxy connexion data on the first call, and simply return HTTP connexion data on subsequent ones
func (c *Conn) Read(b []byte) (int, error) {
	var err error
	c.once.Do(func() { err = c.scanProxyData() })
	if err != nil {
		return 0, err // return if an error occured while reading prefix
	}

	return c.Reader.Read(b)
}

// RemoteAddr returns the address of the client, either from socket peer or from proxy data
func (c *Conn) RemoteAddr() net.Addr {
	var err error

	c.once.Do(func() {
		err = c.scanProxyData()
	})
	if err != nil {
		return nil
	}

	if c.srcAddr != nil && c.useRemoteAddress == true {
		return c.srcAddr
	}
	return c.Conn.RemoteAddr()
}

// scanProxyData check if proxy information is available, returns Conn untouched if a proxy header was not found
func (c *Conn) scanProxyData() error {
	var err error

	if c.readTimeout != 0 {
		c.Conn.SetReadDeadline(time.Now().Add(c.readTimeout))
	}

	var sig []byte
	sig, err = c.Reader.Peek(len(v2sig))
	if err != nil {
		if err == bufio.ErrBufferFull {
			return nil
		}
		return err
	}

	// check if the first bytes match the proxy protocol signature, return Conn untouched otherwise
	if !bytes.Equal(sig, v2sig) {
		return nil
	}

	var header proxyHdrV2
	err = binary.Read(c.Reader, binary.BigEndian, &header)
	if err != nil {
		c.Close()
		return err
	}

	if header.VerCmd == 0x20 { // LOCAL command
		return nil

	} else if header.VerCmd == 0x21 { // PROXY command
		if header.Fam == 0x11 { // TCPv4
			var addr ip4
			err = binary.Read(c.Reader, binary.BigEndian, &addr)
			if err != nil {
				c.Close()
				if err == io.ErrUnexpectedEOF {
					return ErrMalformedProxyHeader
				}
				return err
			}
			c.srcAddr = &net.TCPAddr{IP: addr.SrcAddr[:], Port: int(addr.SrcPort)}
			c.dstAddr = &net.TCPAddr{IP: addr.DstAddr[:], Port: int(addr.DstPort)}
			return nil

		} else if header.Fam == 0x21 { // TCPv6
			var addr ip6
			err = binary.Read(c.Reader, binary.BigEndian, &addr)
			if err != nil {
				c.Close()
				if err == io.ErrUnexpectedEOF {
					return ErrMalformedProxyHeader
				}
				return err
			}
			c.srcAddr = &net.TCPAddr{IP: addr.SrcAddr[:], Port: int(addr.SrcPort)}
			c.dstAddr = &net.TCPAddr{IP: addr.DstAddr[:], Port: int(addr.DstPort)}
			return nil

		} else { // Unsupported protocol
			hdrLen := int(header.Len)
			if hdrLen < c.Reader.Buffered() {
				c.Close()
				return ErrMalformedProxyHeader
			}
			c.Reader.Discard(int(header.Len))
			return nil
		}

	} else { // Unknown PROXY PROTO command
		return ErrUnsupportedProto
	}
}
