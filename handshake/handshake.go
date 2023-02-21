package handshake

import (
	"errors"
	"io"
	"net"
)

// Handshake represents a bittorrent handshake between a client and peer
type Handshake struct {
	// pstr is the protocol string (usually 'BitTorrent protocol')
	pstr string
	// reserved bytes for extensions (e.g. the metadata extension)
	reserved []byte
	// InfoHash is the sum of sha1-hashing the `Info` part of a torrent
	InfoHash []byte
	// peerId is the unique ID for the current peer
	peerId []byte
}

func (h Handshake) pstrlen() int {
	return len(h.pstr)
}

func (h Handshake) Bytes() []byte {
	// 49 = pstrlen + reserved + info_hash + peer_id
	buf := make([]byte, 49+h.pstrlen())
	buf[0] = byte(h.pstrlen())
	n := 1
	n += copy(buf[n:], []byte(h.pstr))
	n += copy(buf[n:], h.reserved)
	n += copy(buf[n:n+20], h.InfoHash)
	copy(buf[n:n+20], h.peerId)
	return buf
}

// SupportsExtensionProtocol returns true if the 20th bit from the right is set
func (h Handshake) SupportsExtensionProtocol() bool {
	if len(h.reserved) < 6 {
		return false
	}

	return h.reserved[5]&0x10 == 16
}

// Read tries to read a peer `Handshake` from `conn`
func Read(conn net.Conn) (Handshake, error) {
	buf := make([]byte, 1)

	_, err := io.ReadFull(conn, buf)
	if err != nil {
		return Handshake{}, err
	}

	pstrlen := int(buf[0])
	if pstrlen == 0 {
		return Handshake{}, errors.New("handshake: invalid protocol strlen (0)")
	}

	buf = make([]byte, pstrlen+48)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return Handshake{}, err
	}

	// offset from 'reserved'
	offset := pstrlen + 8

	hs := Handshake{
		pstr:     string(buf[0:pstrlen]),
		reserved: buf[pstrlen : pstrlen+8],
		InfoHash: buf[offset : offset+20],
		peerId:   buf[offset+20 : offset+40],
	}

	return hs, nil
}

// New is used to create our client handshakes
func New(infoHash, peerId []byte) Handshake {
	return Handshake{
		pstr:     "BitTorrent protocol",
		reserved: []byte{0, 0, 0, 0, 0, 16, 0, 0},
		InfoHash: infoHash,
		peerId:   peerId,
	}
}
