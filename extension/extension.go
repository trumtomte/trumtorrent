package extension

import (
	"trumtorrent/bencode"
)

// m represents the different extension ids used by the current connection,
// however; we currently only care about the metadata ID
type m struct {
	Metadata int `bencode:"ut_metadata"`
}

// Handshake represents the extension handshake between a client and peer
type Handshake struct {
	Ids          m   `bencode:"m"`
	MetadataSize int `bencode:"metadata_size"`
}

// SupportsMetadataExtension returns true if the `ut_metadata` field is set
func (hs Handshake) SupportsMetadataExtension() bool {
	return hs.Ids.Metadata > 0
}

// MetadataMessageId returns metadata ID used for a connection
func (hs Handshake) MetadataMessageId() int {
	return hs.Ids.Metadata
}

func NewHandshake(data []byte) (Handshake, error) {
	hs := Handshake{}
	if err := bencode.Unmarshal(data, &hs); err != nil {
		return Handshake{}, err
	}

	return hs, nil
}

// Piece represents the piece received from the `data` request, this is only
// used for type assertions
type Piece int

// MessageId represents either the handshake (0) or the extension ID (e.g. the
// one used for metadata)
type MessageId uint8

// These represent the different types of messages for the metadata extension
const (
	MessageRequest int = iota
	MessageData
	MessageReject
)

// Message represents the extension protocol message
type Message struct {
	Id        MessageId
	Type      int `bencode:"msg_type"`
	Piece     int `bencode:"piece"`
	TotalSize int `bencode:"total_size"`
	Metadata  []byte
}

func NewMessage(id MessageId, data []byte) (Message, error) {
	msg := Message{}
	if err := bencode.Unmarshal(data, &msg); err != nil {
		return Message{}, err
	}

	_, rest, err := bencode.Decode(data)
	if err != nil {
		return Message{}, err
	}

	msg.Id = id
	msg.Metadata = make([]byte, len(rest))
	copy(msg.Metadata, rest)
	return msg, nil
}
