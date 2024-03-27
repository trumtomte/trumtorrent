package message

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
	"trumtorrent/extension"
	"trumtorrent/piece"
)

// Id represents the message ID
type Id uint8

// These are the types of message IDs we currently support
const (
	Choke Id = iota
	Unchoke
	Interested
	NotInterested
	Have
	Bitfield
	Request
	Piece
	Cancel
	Port
	Extended Id = 20
)

// Message represents a bittorrent message (either from or to a peer)
type Message struct {
	Payload []byte
	Id      Id
}

// Bytes returns the serialized message as bytes
func (m *Message) Bytes() []byte {
	// keep-alive
	if m == nil {
		return []byte{}
	}

	// length + id is 5 bytes
	buf := make([]byte, 5+len(m.Payload))
	binary.BigEndian.PutUint32(buf[0:4], uint32(1+len(m.Payload)))
	buf[4] = byte(m.Id)
	copy(buf[5:], m.Payload)
	return buf
}

// Read reads a message from `conn` and returns it as a `Message`
func Read(conn net.Conn) (*Message, error) {
	defer conn.SetReadDeadline(time.Time{})
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// first 4 bytes is the message length
	buf := make([]byte, 4)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(buf)

	// keep-alive
	if length == 0 {
		return nil, nil
	}

	buf = make([]byte, length)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return nil, err
	}

	return &Message{Id: Id(buf[0]), Payload: buf[1:]}, nil
}

// ParsePieceBlock tries to parse the `Message` into a block (one part of a piece)
func ParsePieceBlock(m *Message) (piece.Block, error) {
	if !Is(m, Piece) || len(m.Payload) < 8 {
		return piece.Block{}, errors.New("message: unable to parse message as `piece`")
	}

	block := piece.Block{
		Index: binary.BigEndian.Uint32(m.Payload[0:4]),
		Begin: binary.BigEndian.Uint32(m.Payload[4:8]),
		Data:  m.Payload[8:],
	}

	return block, nil
}

func ParseHave(m *Message) (int, error) {
	if !Is(m, Have) || len(m.Payload) != 4 {
		return -1, errors.New("message: unable to parse message as `have`")
	}

	return int(binary.BigEndian.Uint32(m.Payload)), nil
}

func ParseExtensionHandshake(m *Message) (extension.Handshake, error) {
	if !Is(m, Extended) || len(m.Payload) == 0 {
		return extension.Handshake{}, errors.New("message: unable to parse message as `extended`")
	}

	id := extension.MessageId(m.Payload[0])
	if id != 0 {
		return extension.Handshake{}, errors.New("message: message is not an extended handshake")
	}

	return extension.NewHandshake(m.Payload[1:])
}

func ParseExtensionMessage(m *Message) (extension.Message, error) {
	if !Is(m, Extended) || len(m.Payload) == 0 {
		return extension.Message{}, errors.New("message: unable to parse message as `extended`")
	}

	id := extension.MessageId(m.Payload[0])
	if id == 0 {
		return extension.Message{}, errors.New("message: message is not a extension message")
	}

	return extension.NewMessage(id, m.Payload[1:])
}

func NewRequest(index, begin, length int) *Message {
	buf := make([]byte, 12)
	binary.BigEndian.PutUint32(buf[0:4], uint32(index))
	binary.BigEndian.PutUint32(buf[4:8], uint32(begin))
	binary.BigEndian.PutUint32(buf[8:12], uint32(length))
	return &Message{Id: Request, Payload: buf}
}

func NewInterested() *Message {
	return &Message{Id: Interested}
}

func NewUnchoke() *Message {
	return &Message{Id: Unchoke}
}

func NewExtensionHandshake() *Message {
	return &Message{Id: Extended}
}

func NewMetadataRequest(id int, piece int) *Message {
	// NOTE: We wont bother encode such a short message from a map
	data := []byte(fmt.Sprintf("d8:msg_typei0e5:piecei%vee", piece))
	buf := make([]byte, 1+len(data))
	buf[0] = byte(id)
	copy(buf[1:], data)
	return &Message{Id: Extended, Payload: buf}
}

func isExtensionHandshake(m *Message) bool {
	if !Is(m, Extended) || len(m.Payload) == 0 {
		return false
	}

	return extension.MessageId(m.Payload[0]) == 0
}

func isExtensionMessagePiece(m *Message, piece int) bool {
	msg, err := ParseExtensionMessage(m)
	if err != nil {
		return false
	}

	return piece == msg.Piece
}

// Is is a type-check helper
func Is(msg *Message, id any) bool {
	switch id.(type) {
	case Id:
		return msg != nil && msg.Id == id.(Id)
	case extension.Handshake:
		return isExtensionHandshake(msg)
	case extension.Piece:
		return isExtensionMessagePiece(msg, id.(int))
	default:
		return false
	}
}
