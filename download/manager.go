package download

import (
	"errors"
	"fmt"
	"log"
	"syscall"
	"time"
	"trumtorrent/client"
	"trumtorrent/peer"
	"trumtorrent/piece"
	"trumtorrent/progress"
	"trumtorrent/torrent"
	"trumtorrent/tracker"
)

const ConnectionLimit int = 30

type Manager struct {
	torrent     *torrent.Torrent
	progress    *progress.Progress
	trackers    []tracker.Tracker
	clients     map[string]*client.Client
	peers       chan *peer.Peer
	downloaded  chan *piece.Piece
	connections int
	connWait    chan struct{}
}

func (m *Manager) wait() {
	// FIXME: Write something that does batch writes instead, and also handles
	// 		  i/o errors
	for !m.progress.Complete() {
		select {
		case p := <-m.downloaded:
			if err := p.Write(); err != nil {
				fmt.Println(err)
			}

			m.progress.CalculateProgress(p)
		}
	}

	m.progress.Done()
}

func (m *Manager) connectToPeer(c *client.Client) {
	log.Printf("Connecting to peer '%v'", c.Peer.String())

	// TODO: we could most likely do this in a better way
	for retries := 0; retries <= 5; retries++ {
		if err := c.Connect(); err != nil {
			if errors.Is(err, syscall.EPIPE) || errors.Is(err, syscall.ECONNRESET) {
				continue
			}

			return
		}

		break
	}

	for retries := 0; retries <= 5; retries++ {
		if err := c.Download(m.downloaded); err != nil {
			if errors.Is(err, syscall.EPIPE) || errors.Is(err, syscall.ECONNRESET) {
				continue
			}

			return
		}

		break
	}

	log.Printf("Disconnecting from peer '%v'", c.Peer.String())

	m.connections--
	if m.connections < ConnectionLimit {
		m.connWait <- struct{}{}
	}
}

func (m *Manager) connectToPeers() {
	for _, c := range m.clients {
		// NOTE: We currently only connect to each peer once, even if it
		// 		 disconnects for some reason.
		if c.State != client.Idle {
			continue
		}

		if m.connections >= ConnectionLimit {
			<-m.connWait
			continue
		}

		m.connections++
		go m.connectToPeer(c)
	}

	if m.progress.Complete() {
		return
	}

	time.Sleep(2 * time.Second)
	m.connectToPeers()
}

func (m *Manager) waitForPeers() {
	for {
		select {
		case p := <-m.peers:
			// Only allow a unique set of peers
			if _, exists := m.clients[p.String()]; !exists {
				m.clients[p.String()] = client.New(p, m.torrent)
			}
		case <-time.After(5 * time.Second):
			if m.progress.Complete() {
				return
			}
		}
	}
}

func (m *Manager) announceToTracker(tr tracker.Tracker) {
	log.Printf("Announcing to tracker '%v'", tr.String())

	if err := tr.Announce(); err != nil {
		fmt.Println(err)
		return
	}

	for _, p := range tr.Peers() {
		m.peers <- p
	}
}

func (m *Manager) announceToTrackers() {
	// NOTE: We connect to all trackers at once (and only once), we might want
	// 		 to wait if we've currently got enough peers.
	for _, tr := range m.trackers {
		go m.announceToTracker(tr)
	}
}

func (m *Manager) setupTrackers() {
	for _, addr := range m.torrent.Trackers() {
		t, err := tracker.New(addr, m.torrent)
		if err != nil {
			continue
		}

		m.trackers = append(m.trackers, t)
	}
}

func (m *Manager) Download() {
	m.setupTrackers()
	go m.announceToTrackers()
	go m.waitForPeers()
	go m.connectToPeers()
	m.wait()
}

func NewManager(t *torrent.Torrent) *Manager {
	return &Manager{
		torrent:    t,
		progress:   progress.New(t),
		peers:      make(chan *peer.Peer, 64),
		downloaded: make(chan *piece.Piece, 128),
		clients:    make(map[string]*client.Client),
		connWait:   make(chan struct{}),
	}
}
