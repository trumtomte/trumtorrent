package client

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"net"
	"time"
	"trumtorrent/extension"
	"trumtorrent/handshake"
	"trumtorrent/message"
	"trumtorrent/metadata"
	"trumtorrent/peer"
	"trumtorrent/piece"
	"trumtorrent/torrent"
)

type state int

const (
	Idle state = iota
	Connecting
	Connected
	Disconnected
	Downloading
	Done
)

// Client represents the connection towards one peer
type Client struct {
	conn    net.Conn
	torrent *torrent.Torrent
	Peer    *peer.Peer
	piece   *piece.Piece
	// haveBuf is used to buffer received HAVE messages, so we can insert
	// them into our bitfield later on when we've got a complete torrent
	haveBuf    []int
	State      state
	choked     bool
	interested bool
}

// BlockSize is the default size for requests (i.e. block length)
const BlockSize = 16384

func (c Client) send(m *message.Message) error {
	c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	defer c.conn.SetWriteDeadline(time.Time{})

	if _, err := c.conn.Write(m.Bytes()); err != nil {
		return err
	}

	return nil
}

func (c Client) sendRequest(index, begin, length int) error {
	return c.send(message.NewRequest(index, begin, length))
}

func (c Client) sendInterested() error {
	return c.send(message.NewInterested())
}

func (c Client) sendUnchoke() error {
	return c.send(message.NewUnchoke())
}

func (c Client) sendMetadataRequest(piece int) error {
	return c.send(message.NewMetadataRequest(c.Peer.MetadataMessageId(), piece))
}

func (c *Client) flushHaveBuffer() {
	for _, p := range c.haveBuf {
		c.Peer.SetPiece(p)
	}

	c.haveBuf = nil
}

func (c *Client) handleHaveMessage(msg *message.Message) error {
	piece, err := message.ParseHave(msg)
	if err != nil {
		return err
	}

	// Lets buffer the HAVE's so we can reinstate them later on (if they're
	// received during the download of the metadata)
	if c.torrent.MetaInfo.Incomplete() {
		c.haveBuf = append(c.haveBuf, piece)
	}

	// If we've received HAVE messages but not a bitfield we'll make a empty
	// bitfield in order to store HAVE messages
	if !c.Peer.HasBitfield() && !c.torrent.MetaInfo.Incomplete() {
		size := int(math.Ceil(float64(cap(c.torrent.Pieces)) / 8))
		c.Peer.SetBitfield(make([]byte, size))
	}

	c.Peer.SetPiece(piece)
	return nil
}

func (c Client) handlePieceMessage(msg *message.Message) error {
	block, err := message.ParsePieceBlock(msg)
	if err != nil {
		return err
	}

	c.piece.Received += copy(c.piece.Data[block.Begin:], block.Data)
	c.piece.QueuedRequests--
	return nil
}

func (c Client) handleExtendedMessage(msg *message.Message) error {
	if message.Is(msg, extension.Handshake{}) {
		hs, err := message.ParseExtensionHandshake(msg)
		if err != nil {
			return err
		}

		c.Peer.SetExtensionHandshake(hs)

		if c.torrent.MetaInfo.Incomplete() && c.Peer.SupportsMetadataExtension() && c.torrent.Metadata == nil {
			c.torrent.Metadata = metadata.New(c.Peer.MetadataSize())
		}

		return nil
	}

	m, err := message.ParseExtensionMessage(msg)
	if err != nil {
		return err
	}

	// We only care for the `data` messages
	if m.Type == extension.MessageData {
		c.torrent.ReceiveMetadata(m)
	}

	return nil
}

func (c *Client) receive() (err error) {
	var (
		retries int
		msg     *message.Message
	)

	for {
		if retries >= 5 {
			return err
		}

		msg, err = message.Read(c.conn)

		if err == nil {
			break
		}

		if err, ok := err.(net.Error); ok && err.Timeout() {
			retries++
			continue
		}

		return err
	}

	// keep-alive
	if msg == nil {
		return nil
	}

	// TODO: store peer_choked and peer_interested

	switch msg.Id {
	case message.Choke:
		c.choked = true
	case message.Unchoke:
		c.choked = false
	case message.Bitfield:
		// NOTE: this will overwrite the HAVE's we've gotten so far (hopefully
		// 		 it will contain those anyway, we could merge them in the
		// 		 future)
		c.Peer.SetBitfield(msg.Payload)
	case message.Have:
		err = c.handleHaveMessage(msg)
	case message.Piece:
		err = c.handlePieceMessage(msg)
	case message.Extended:
		err = c.handleExtendedMessage(msg)
	}

	return err
}

func (c *Client) requestPiece(p *piece.Piece) error {
	fmt.Println("client: requesting piece", p.Index)

	c.piece = p
	defer func() { c.piece = nil }()

	// NOTE: We'll initialise the buffer here to save memory
	p.Data = make([]byte, p.Length)

	for p.Incomplete() {
		if !c.choked {
			for p.CanQueueRequest() {
				length := BlockSize

				// last block might be truncated
				if p.Requested+length > p.Length {
					length = p.Length % length
				}

				if err := c.sendRequest(p.Index, p.Requested, length); err != nil {
					return err
				}

				p.Requested += length
				p.QueuedRequests++
			}
		}

		if err := c.receive(); err != nil {
			return err
		}
	}

	return nil
}

func (c Client) requestPieces(downloaded chan *piece.Piece) error {
	for {
		select {
		case p := <-c.torrent.Pieces:
			// FIXME: This could end up as an infinite loop unless we start
			// 		  receiving more HAVE messages. Maybe we should continously
			// 		  check if we are interested in our peer.
			if !c.Peer.HasPiece(p.Index) {
				fmt.Println("client: peer doesnt have", p.Index)
				c.torrent.Pieces <- p
				break
			}

			if err := c.requestPiece(p); err != nil {
				p.Reset()
				c.torrent.Pieces <- p
				return err
			}

			if !c.torrent.IsValidPieceHash(p) {
				p.Reset()
				c.torrent.Pieces <- p
				break
			}

			downloaded <- p
		default:
			return nil
		}
	}
}

func (c Client) receiveMetadataPiece(piece int) error {
	for !c.torrent.Metadata.HasPiece(piece) {
		if err := c.receive(); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) requestMetadataPieces() error {
	for {
		select {
		case piece := <-c.torrent.Metadata.Pieces:
			if err := c.sendMetadataRequest(piece); err != nil {
				c.torrent.Metadata.Pieces <- piece
				return err
			}

			if err := c.receiveMetadataPiece(piece); err != nil {
				c.torrent.Metadata.Pieces <- piece
				return err
			}
		default:
			return nil
		}
	}
}

// establishHandshake sends, receives and verifies the handshake between a client (chs) and peer (phs)
func (c Client) establishHandshake(chs handshake.Handshake) error {
	c.conn.SetDeadline(time.Now().Add(10 * time.Second))
	defer c.conn.SetDeadline(time.Time{})

	fmt.Println("client: establishing handshake")

	if _, err := c.conn.Write(chs.Bytes()); err != nil {
		return err
	}

	phs, err := handshake.Read(c.conn)
	if err != nil {
		return err
	}

	if !bytes.Equal(chs.InfoHash, phs.InfoHash) {
		return errors.New("client: received an invalid info hash")
	}

	fmt.Println("client: handshake established")

	c.Peer.SetHandshake(phs)
	return nil
}

func (c *Client) close(err error) {
	if err == nil {
		return
	}

	if c.conn != nil {
		c.conn.Close()
	}

	c.State = Disconnected
}

func (c *Client) Connect() (err error) {
	c.State = Connecting
	defer c.close(err)

	var (
		conn    net.Conn
		retries int
	)

	for {
		if retries >= 5 {
			return err
		}

		conn, err = net.DialTimeout("tcp", c.Peer.String(), 10*time.Second)

		if err == nil {
			c.conn = conn
			break
		}

		if err, ok := err.(net.Error); ok && err.Timeout() {
			retries++
			continue
		}

		return err
	}

	hs := handshake.New(c.torrent.InfoHash, c.torrent.PeerId)
	if err = c.establishHandshake(hs); err != nil {
		return err
	}

	c.State = Connected
	return nil
}

func (c Client) downloadMetadata() error {
	for c.torrent.MetaInfo.Incomplete() {
		if err := c.receive(); err != nil {
			return err
		}

		if !c.Peer.SupportsExtensionProtocol() {
			return errors.New("client: peer does not support the extension protocol")
		}

		// Wait for the metadata handshake
		if !c.Peer.SupportsMetadataExtension() {
			continue
		}

		if err := c.requestMetadataPieces(); err != nil {
			return err
		}

		// Wait for all metadata to be downloaded
		<-c.torrent.Metadata.Wait
	}

	return nil
}

func (c *Client) Download(downloaded chan *piece.Piece) (err error) {
	c.State = Downloading
	defer c.close(err)

	fmt.Println("client: starting download")

	// If our torrent is incomplete we need to download the metadata first
	if c.torrent.MetaInfo.Incomplete() {
		c.downloadMetadata()
	}

	if len(c.haveBuf) > 0 {
		c.flushHaveBuffer()
	}

	// NOTE: lets try to wait for a couple of (10) messages, to see if we get
	// 		 any HAVEs/Bitfield. This could most likely be its own function
	var tries int

	for {
		if c.Peer.HasBitfield() {
			break
		}

		if tries > 10 {
			return errors.New("client: waited to long for a BITFIELD or HAVE message")
		}

		if err = c.receive(); err != nil {
			return err
		}

		tries++
	}

	// NOTE: We presume the peer has pieces we need
	if err = c.sendInterested(); err != nil {
		return err
	}

	fmt.Println("client: requesting pieces")
	if err = c.requestPieces(downloaded); err != nil {
		return err
	}

	c.conn.Close()
	c.State = Done
	return nil
}

func New(peer *peer.Peer, t *torrent.Torrent) *Client {
	return &Client{
		State:      Idle,
		Peer:       peer,
		torrent:    t,
		choked:     true,
		interested: false,
	}
}
