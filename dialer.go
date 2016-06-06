package hpkp

import (
	"crypto/tls"
	"errors"
	"net"
	"strings"
)

// Storage is threadsafe hsts storage interface
type Storage interface {
	Lookup(host string) *Header
	Add(host string, d *Header)
}

// NewPinDialer returns a function suitable for use as DialTLS
func NewPinDialer(s Storage, pinOnly bool, defaultTLSConfig *tls.Config) func(network, addr string) (net.Conn, error) {
	return func(network, addr string) (net.Conn, error) {
		// might need to strip ":https" from addr as well
		h := s.Lookup(strings.TrimRight(addr, ":443"))

		if h != nil {
			c, err := tls.Dial(network, addr, &tls.Config{InsecureSkipVerify: pinOnly})
			if err != nil {
				return c, err
			}
			validPin := false
			// intermediates can be pinned as well, loop through leaf-> root looking
			// for pins
			for _, peercert := range c.ConnectionState().PeerCertificates {
				peerPin := Fingerprint(peercert)
				if h.Matches(peerPin) {
					validPin = true
					break
				}
			}
			if validPin == false {
				return nil, errors.New("pin was not valid")
			}
			return c, nil
		}
		// do a normal dial
		return tls.Dial(network, addr, defaultTLSConfig)
	}
}
