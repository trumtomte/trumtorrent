package torrent

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/url"
	"strings"
	"trumtorrent/bencode"
	"trumtorrent/extension"
	"trumtorrent/metadata"
	"trumtorrent/piece"
)

// InfoFile represents one of multiples file within a torrent
type InfoFile struct {
	Length int      `bencode:"length"`
	Path   []string `bencode:"path"`
}

// Info contains the practical data of a torrent
type Info struct {
	Files       []InfoFile `bencode:"files"`
	Length      int        `bencode:"length"`
	Name        string     `bencode:"name"`
	PieceLength int        `bencode:"piece length"`
	Pieces      string     `bencode:"pieces"`
	Private     int        `bencode:"private"`
}

// MetaInfo represents the container for meta data of a torrent
type MetaInfo struct {
	AnnounceList [][]string `bencode:"announce-list"`
	Announce     string     `bencode:"announce"`
	Comment      string     `bencode:"comment"`
	CreatedBy    string     `bencode:"created by"`
	CreationDate int        `bencode:"creation date"`
	Encoding     string     `bencode:"encoding"`
	Info         Info       `bencode:"info"`
}

// Incomplete is used in order to check if we need to download the metadata or
// not (e.g. when we're downloading from a magnet link), this could most likely
// be done in a better fashion
func (m MetaInfo) Incomplete() bool {
	return m.Info.Pieces == "" || m.Info.PieceLength == 0
}

type Torrent struct {
	MetaInfo MetaInfo
	InfoHash []byte
	PeerId   []byte
	Pieces   chan *piece.Piece
	Metadata *metadata.Metadata
	// length is simply a cache of the torrent size (since lots of torrents are
	// in multiple file mode)
	length int
}

func (t Torrent) Name() string {
	return t.MetaInfo.Info.Name
}

func (t *Torrent) cacheLength() {
	if t.MetaInfo.Info.Length > 0 {
		t.length = t.MetaInfo.Info.Length
	}

	size := 0
	for _, file := range t.MetaInfo.Info.Files {
		size += file.Length
	}

	t.length = size
}

func (t Torrent) Length() int {
	if t.length == 0 {
		t.cacheLength()
	}

	return t.length
}

func (t Torrent) PieceLength() int {
	return t.MetaInfo.Info.PieceLength
}

func (t Torrent) IsMultipleFileMode() bool {
	return len(t.MetaInfo.Info.Files) > 0
}

func (t Torrent) PieceHash(index int) []byte {
	begin := index * 20
	length := begin + 20

	if length > len(t.MetaInfo.Info.Pieces) {
		return nil
	}

	hash := t.MetaInfo.Info.Pieces[begin:length]
	return []byte(hash)
}

func (t Torrent) IsValidPieceHash(p *piece.Piece) bool {
	pieceHash := sha1.Sum(p.Data)
	return bytes.Equal(t.PieceHash(p.Index), pieceHash[:])
}

// destinations calculates where data from a Piece is supposed to be written (in
// many cases it is split across multiple files)
func (t Torrent) destinations(offset int, length int) []piece.Destination {
	// If we're downloading a single file, we've only got one destination
	if !t.IsMultipleFileMode() {
		dst := piece.Destination{
			Path:   t.Name(),
			Offset: offset,
			Start:  0,
			End:    length,
		}

		return []piece.Destination{dst}
	}

	var (
		currOffset   int
		overlap      int
		destinations []piece.Destination
	)

	// FIXME: we need to check to see if this works with a piece split across
	//        more then two files
	for _, file := range t.MetaInfo.Info.Files {
		boundary := currOffset + file.Length

		dst := piece.Destination{
			Path: t.Name() + "/" + strings.Join(file.Path, "/"),
		}

		// Whole piece fits within this file
		if offset+length <= boundary {
			if overlap > 0 {
				dst.Offset = 0
				dst.Start = overlap
			} else {
				dst.Offset = offset - currOffset
				dst.Start = 0
			}

			dst.End = length
			destinations = append(destinations, dst)
			break
		}

		// Piece overlap file boundaries
		if offset < boundary {
			overlap = boundary - offset
			dst.Offset = offset - currOffset
			dst.Start = 0
			dst.End = overlap
			destinations = append(destinations, dst)
		}

		currOffset += file.Length - overlap
	}

	return destinations
}

func (t Torrent) pieces() []*piece.Piece {
	pieces := make([]*piece.Piece, len(t.MetaInfo.Info.Pieces)/20)

	if len(pieces) == 0 {
		return pieces
	}

	for index := range pieces {
		length := t.PieceLength()
		offset := index * t.PieceLength()

		// Last piece might be truncated
		if index*length+length > t.Length() {
			length = t.Length() - index*length
		}

		pieces[index] = &piece.Piece{
			Index:        index,
			Length:       length,
			Offset:       offset,
			Destinations: t.destinations(offset, length),
		}
	}

	// NOTE: It's not always the case that downloading in ascending order is the
	// fastest (nothing I've tested though but from reading stuff online simply
	// downloading in random order might prove to be more performant)
	rand.Shuffle(len(pieces), func(i, j int) {
		pieces[i], pieces[j] = pieces[j], pieces[i]
	})

	return pieces
}

func (t Torrent) Trackers() []string {
	if len(t.MetaInfo.AnnounceList) == 0 {
		if t.MetaInfo.Announce == "" {
			return nil
		}

		return []string{t.MetaInfo.Announce}
	}

	trackers := make([]string, len(t.MetaInfo.AnnounceList))
	for i, announce := range t.MetaInfo.AnnounceList {
		trackers[i] = announce[0]
	}

	return trackers
}

// ReceiveMetadata is used for collecting metadata messages
func (t *Torrent) ReceiveMetadata(msg extension.Message) {
	t.Metadata.Receive(msg)

	if t.Metadata.Complete() && t.MetaInfo.Incomplete() {
		// TODO: handle this error
		info := &Info{}
		if err := bencode.Unmarshal(t.Metadata.Data, info); err != nil {
			fmt.Println(err)
			return
		}

		t.MetaInfo.Info = *info
		t.populatePieceChannel()
		t.cacheLength()
		close(t.Metadata.Wait)
	}
}

func (t *Torrent) populatePieceChannel() {
	if t.MetaInfo.Incomplete() {
		return
	}

	pieces := t.pieces()
	t.Pieces = make(chan *piece.Piece, len(pieces))

	for _, piece := range pieces {
		t.Pieces <- piece
	}
}

func generateInfoHash(i Info) ([]byte, error) {
	data, err := bencode.Marshal(i)
	if err != nil {
		return nil, err
	}

	hash := sha1.Sum(data)
	return hash[:], nil
}

func generatePeerId() ([]byte, error) {
	buf := make([]byte, 20)
	// NOTE: Our peer ID, which is also the ID for this torrent client
	copy(buf[0:8], "-TM0001-")

	if _, err := rand.Read(buf[8:20]); err != nil {
		return nil, err
	}

	return buf, nil
}

// openTorrentFromMagnet creates a new 'incomplete' Torrent, which will require downloading
// the metadata from Peers before the actual torrent
func openTorrentFromMagnet(magnetLink string) (*Torrent, error) {
	url, err := url.Parse(magnetLink)
	if err != nil {
		return nil, err
	}

	values := url.Query()

	if !values.Has("xt") || !values.Has("dn") || !values.Has("tr") {
		return nil, errors.New("torrent: missing magnet link param (xt/dn/tr)")
	}

	xt := values.Get("xt")

	if !strings.HasPrefix(xt, "urn:btih:") {
		return nil, fmt.Errorf("torrent: invalid prefix for param 'xt', got %s", xt)
	}

	tr := values["tr"]

	if len(tr) == 0 {
		return nil, errors.New("torrent: no trackers")
	}

	trackers := make([][]string, len(tr))
	for i, tracker := range tr {
		trackers[i] = []string{tracker}
	}

	// len("urn:btih:")
	hashAsHex := []byte(xt[9:])
	hash := make([]byte, hex.DecodedLen(len(hashAsHex)))
	_, err = hex.Decode(hash, hashAsHex)
	if err != nil {
		return nil, errors.New("torrent: unable to decode info hash")
	}

	name := values.Get("dn")

	if len(name) == 0 {
		return nil, errors.New("torrent: empty name")
	}

	metainfo := &MetaInfo{
		AnnounceList: trackers,
		Announce:     trackers[0][0],
		Info:         Info{Name: name},
	}

	peerId, err := generatePeerId()
	if err != nil {
		return nil, err
	}

	return &Torrent{
		MetaInfo: *metainfo,
		InfoHash: hash,
		PeerId:   peerId,
	}, nil
}

// Open reads a torrent from disk (a .torrent file) or a incomplete torrent from
// a magnet link
func Open(path string) (*Torrent, error) {
	if strings.HasPrefix(path, "magnet:") {
		return openTorrentFromMagnet(path)
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	metainfo := &MetaInfo{}
	if err = bencode.Unmarshal(data, metainfo); err != nil {
		return nil, err
	}

	hash, err := generateInfoHash(metainfo.Info)
	if err != nil {
		return nil, err
	}

	peerId, err := generatePeerId()
	if err != nil {
		return nil, err
	}

	t := &Torrent{
		MetaInfo: *metainfo,
		InfoHash: hash,
		PeerId:   peerId,
	}

	t.populatePieceChannel()
	t.cacheLength()
	return t, nil
}
