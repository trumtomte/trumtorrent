package metadata

import (
	"math"
	_ "trumtorrent/bencode"
	"trumtorrent/bitfield"
	"trumtorrent/extension"
)

// Metadata is used to store the downloaded MetaInfo data (which is needed when
// downloading a torrent via a magnet link)
type Metadata struct {
	Data     []byte
	written  int
	received bitfield.Bitfield
	Pieces   chan int
	Wait     chan struct{}
}

func (m *Metadata) Receive(msg extension.Message) {
	offset := msg.Piece * 16384
	m.written += copy(m.Data[offset:], msg.Metadata)
	m.received.SetPiece(msg.Piece)
}

func (m *Metadata) HasPiece(piece int) bool {
	return m.received.HasPiece(piece)
}

func (m *Metadata) Complete() bool {
	return m.written == len(m.Data)
}

func New(size int) *Metadata {
	n := int(math.Ceil(float64(size) / 16384))
	pieces := make(chan int, n)

	for piece := 0; piece < n; piece++ {
		pieces <- piece
	}

	bs := int(math.Ceil(float64(n) / 8))

	return &Metadata{
		Data:     make([]byte, n),
		received: make([]byte, bs),
		Wait:     make(chan struct{}),
		Pieces:   pieces,
	}
}
