package peer

import (
	"net"
	"strconv"
	"trumtorrent/bitfield"
	"trumtorrent/extension"
	"trumtorrent/handshake"
)

// Addr represents the connection details to a Peer
type Addr struct {
	IP   net.IP
	Port uint16
}

func (a Addr) String() string {
	return net.JoinHostPort(a.IP.String(), strconv.Itoa(int(a.Port)))
}

// Peer is used in order to store information related to a Peer connection
type Peer struct {
	Addr      Addr
	bitfield  bitfield.Bitfield
	handshake handshake.Handshake
	extension extension.Handshake
}

func (p Peer) HasBitfield() bool {
	return len(p.bitfield) > 0
}

func (p *Peer) SetBitfield(data []byte) {
	p.bitfield = data
}

func (p Peer) HasPiece(piece int) bool {
	return p.bitfield.HasPiece(piece)
}

func (p Peer) SetPiece(piece int) {
	p.bitfield.SetPiece(piece)
}

func (p *Peer) SetHandshake(hs handshake.Handshake) {
	p.handshake = hs
}

func (p Peer) SupportsExtensionProtocol() bool {
	return p.handshake.SupportsExtensionProtocol()
}

func (p *Peer) SetExtensionHandshake(hs extension.Handshake) {
	p.extension = hs
}

func (p Peer) SupportsMetadataExtension() bool {
	return p.extension.SupportsMetadataExtension()
}

func (p Peer) MetadataMessageId() int {
	return p.extension.MetadataMessageId()
}

func (p Peer) MetadataSize() int {
	return p.extension.MetadataSize
}

func (p Peer) String() string {
	return p.Addr.String()
}

func New(ip []byte, port uint16) *Peer {
	return &Peer{Addr: Addr{IP: net.IP(ip), Port: port}}
}
