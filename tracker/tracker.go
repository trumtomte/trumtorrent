package tracker

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"syscall"
	"time"
	"trumtorrent/bencode"
	"trumtorrent/peer"
	"trumtorrent/torrent"
)

// TODO: make the trackers send proper values for how much we've downloaded so
// 	     far etc., perhaps as arguments to `Announce()`?

type Tracker interface {
	Scheme() string
	Announce() error
	Peers() []*peer.Peer
	String() string
}

type HTTPTracker struct {
	url      *url.URL
	torrent  *torrent.Torrent
	peers    []*peer.Peer
	response time.Time
	interval int
	port     int
}

type httpResponse struct {
	Interval      int    `bencode:"interval"`
	Peers         string `bencode:"peers"`
	FailureReason string `bencode:"failure reason"`
	// Unused/optional fields
	TrackerId      string `bencode:"tracker id"`
	Seeders        int    `bencode:"complete"`
	Leechers       int    `bencode:"incomplete"`
	WarningMessage string `bencode:"warning message"`
	MinInterval    int    `bencode:"min interval"`
}

func (t *HTTPTracker) buildHttpQuery() string {
	query := t.url.Query()
	query.Add("info_hash", string(t.torrent.InfoHash))
	query.Add("peer_id", string(t.torrent.PeerId))
	query.Add("port", strconv.Itoa(t.port))
	query.Add("uploaded", "0")
	query.Add("downloaded", "0")
	query.Add("left", strconv.Itoa(t.torrent.Length()))
	query.Add("compact", "1")
	query.Add("numwant", "50")
	return query.Encode()
}

func (t *HTTPTracker) Announce() (err error) {
	client := &http.Client{
		Timeout: time.Duration(10 * time.Second),
	}

	t.url.RawQuery = t.buildHttpQuery()

	var (
		res     *http.Response
		retries int
	)

	for {
		if retries >= 5 {
			return err
		}

		res, err = client.Get(t.url.String())

		if err == nil {
			break
		}

		if errors.Is(err, context.DeadlineExceeded) {
			retries++
			continue
		}

		if err, ok := err.(net.Error); ok && err.Timeout() {
			retries++
			continue
		}

		return err
	}

	data, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return err
	}

	// result, _, err := bencode.Decode(data)
	// if err != nil {
	// 	return err
	// }

	hres := &httpResponse{}
	if err = bencode.Unmarshal(data, hres); err != nil {
		return err
	}

	if len(hres.FailureReason) > 0 {
		return fmt.Errorf("httptracker: announce failed '%s'", hres.FailureReason)
	}

	// TODO: Add support for non-compact peers
	peers, err := parseCompactPeers([]byte(hres.Peers))
	if err != nil {
		return err
	}

	t.peers = peers
	t.response = time.Now()
	t.interval = hres.Interval
	return nil
}

func (t *HTTPTracker) Peers() []*peer.Peer {
	return t.peers
}

func (t *HTTPTracker) Scheme() string {
	return t.url.Scheme
}

func (t *HTTPTracker) String() string {
	return t.url.Hostname()
}

func NewHTTPTracker(url *url.URL, t *torrent.Torrent) *HTTPTracker {
	return &HTTPTracker{url: url, torrent: t, port: 6881}
}

// UDP action IDs
const (
	ActionConnect uint32 = iota
	ActionAnnounce
	ActionScrape
	ActionError
)

type UDPTracker struct {
	url      *url.URL
	torrent  *torrent.Torrent
	conn     *net.UDPConn
	raddr    *net.UDPAddr
	peers    []*peer.Peer
	response time.Time
	interval int
	port     int
	timeout  float64
}

func (t *UDPTracker) transactionId() ([]byte, error) {
	transId := make([]byte, 4)

	if _, err := rand.Read(transId); err != nil {
		return nil, err
	}

	return transId, nil
}

func (t *UDPTracker) buildConnectPacket(transactionId []byte) []byte {
	payload := make([]byte, 16)
	binary.BigEndian.PutUint64(payload[0:8], 0x41727101980) // magic constant
	binary.BigEndian.PutUint32(payload[8:12], ActionConnect)
	binary.BigEndian.PutUint32(payload[12:16], binary.BigEndian.Uint32(transactionId))
	return payload
}

func (t *UDPTracker) buildAnnouncePacket(transactionId []byte, connectionId []byte) []byte {
	payload := make([]byte, 98)
	binary.BigEndian.PutUint64(payload[0:8], binary.BigEndian.Uint64(connectionId))
	binary.BigEndian.PutUint32(payload[8:12], ActionAnnounce)
	binary.BigEndian.PutUint32(payload[12:16], binary.BigEndian.Uint32(transactionId))
	copy(payload[16:36], t.torrent.InfoHash)
	copy(payload[36:56], t.torrent.PeerId)
	binary.BigEndian.PutUint64(payload[56:64], 0)                          // downloaded
	binary.BigEndian.PutUint64(payload[64:72], uint64(t.torrent.Length())) // left
	binary.BigEndian.PutUint64(payload[72:80], 0)                          // uploaded
	binary.BigEndian.PutUint32(payload[80:84], 0)                          // event
	binary.BigEndian.PutUint32(payload[84:88], 0)                          // ip
	binary.BigEndian.PutUint32(payload[88:92], 0)                          // key
	binary.BigEndian.PutUint32(payload[92:96], 50)                         // num want
	binary.BigEndian.PutUint16(payload[96:98], uint16(t.port))             // port
	return payload
}

// setDeadline is used to control the duration between retransmits, the spec
// says it should be 15 * 2 ^ n seconds (where n is <= 8)
func (t *UDPTracker) setDeadline() error {
	timeout := time.Duration(15 * math.Pow(2, t.timeout))
	t.conn.SetReadDeadline(time.Now().Add(timeout * time.Second))
	t.timeout++

	if t.timeout > 8 {
		return errors.New("udptracker: retransmission limit")
	}

	return nil
}

func (t *UDPTracker) hasConnectionExpired() bool {
	return time.Since(t.response).Seconds() >= 60
}

func (t *UDPTracker) connect() error {
	// NOTE: We only test ports from 6881 to 6999
	for t.port < 7000 {
		laddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%v", t.port))
		if err != nil {
			return err
		}

		conn, err := net.ListenUDP("udp", laddr)
		if err != nil {
			if errors.Is(err, syscall.EADDRINUSE) {
				t.port++
				continue
			}

			return err
		}

		t.conn = conn
		return nil
	}

	return errors.New("udptracker: no port available")
}

func (t *UDPTracker) sendConnect(transactionId []byte) error {
	raddr, err := net.ResolveUDPAddr("udp", t.url.Host)
	if err != nil {
		return err
	}

	t.raddr = raddr

	packet := t.buildConnectPacket(transactionId)
	_, err = t.conn.WriteToUDP(packet, raddr)
	return err
}

func (t *UDPTracker) receiveConnect(transactionId []byte) ([]byte, error) {
	if err := t.setDeadline(); err != nil {
		return nil, err
	}

	buf := make([]byte, 16)
	read, err := t.conn.Read(buf)
	if err != nil {
		return nil, err
	}

	t.conn.SetReadDeadline(time.Time{})
	t.response = time.Now()

	if read < 16 {
		return nil, errors.New("tracker: UDP connect response was to small")
	}

	if !bytes.Equal(transactionId, buf[4:8]) {
		return nil, errors.New("tracker: received an invalid UDP connect transaction ID")
	}

	if binary.BigEndian.Uint32(buf[0:4]) != ActionConnect {
		return nil, errors.New("tracker: received an invalid UDP connect action")
	}

	// connection id
	return buf[8:16], nil
}

func (t *UDPTracker) sendAnnounce(transactionId []byte, connectionId []byte) error {
	packet := t.buildAnnouncePacket(transactionId, connectionId)
	_, err := t.conn.WriteToUDP(packet, t.raddr)
	return err
}

func (t *UDPTracker) receiveAnnounce(transactionId []byte) (int, []byte, error) {
	if err := t.setDeadline(); err != nil {
		return 0, nil, err
	}

	buf := make([]byte, 512)
	read, err := t.conn.Read(buf)
	if err != nil {
		return 0, nil, err
	}

	t.conn.SetReadDeadline(time.Time{})

	if read < 20 {
		return 0, nil, errors.New("tracker: UDP announce response was to small")
	}

	if !bytes.Equal(transactionId, buf[4:8]) {
		return 0, nil, errors.New("tracker: received an invalid UDP announce transaction ID")
	}

	if binary.BigEndian.Uint32(buf[0:4]) != ActionAnnounce {
		return 0, nil, errors.New("tracker: received an invalid UDP announce action")
	}

	// interval and peers
	return int(binary.BigEndian.Uint32(buf[8:12])), buf[20:read], nil
}

func (t *UDPTracker) Announce() (err error) {
	if err = t.connect(); err != nil {
		return
	}

	defer t.conn.Close()

CONNECT:

	var transId []byte

	if transId, err = t.transactionId(); err != nil {
		return
	}

	if err = t.sendConnect(transId); err != nil {
		return
	}

	var connId []byte

	for {
		connId, err = t.receiveConnect(transId)

		if err == nil {
			break
		}

		if err, ok := err.(net.Error); ok && err.Timeout() {
			continue
		}

		return err
	}

	// ANNOUNCE
	if transId, err = t.transactionId(); err != nil {
		return
	}

	if err = t.sendAnnounce(transId, connId); err != nil {
		return
	}

	var (
		interval     int
		compactPeers []byte
	)

	for {
		if t.hasConnectionExpired() {
			goto CONNECT
		}

		interval, compactPeers, err = t.receiveAnnounce(transId)

		if err == nil {
			break
		}

		if err, ok := err.(net.Error); ok && err.Timeout() {
			continue
		}

		return err
	}

	var peers []*peer.Peer

	if peers, err = parseCompactPeers(compactPeers); err != nil {
		return
	}

	t.peers = peers
	t.response = time.Now()
	t.interval = interval

	return
}

func (t *UDPTracker) Peers() []*peer.Peer {
	return t.peers
}

func (t *UDPTracker) Scheme() string {
	return t.url.Scheme
}

func (t *UDPTracker) String() string {
	return t.url.String()
}

func NewUDPTracker(url *url.URL, t *torrent.Torrent) *UDPTracker {
	return &UDPTracker{url: url, torrent: t, port: 6881}
}

func New(addr string, t *torrent.Torrent) (Tracker, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "udp":
		return NewUDPTracker(u, t), nil
	case "http", "https":
		return NewHTTPTracker(u, t), nil
	default:
		return nil, errors.New("tracker: unsupported scheme")
	}
}

func parseCompactPeers(data []byte) ([]*peer.Peer, error) {
	if len(data)%6 != 0 {
		return nil, errors.New("tracker: invalid peer binary length")
	}

	peers := make([]*peer.Peer, len(data)/6)

	for i := 0; i < len(data)/6; i++ {
		offset := i * 6
		ip := data[offset : offset+4]
		port := binary.BigEndian.Uint16(data[offset+4 : offset+6])
		peers[i] = peer.New(ip, port)
	}

	return peers, nil
}
