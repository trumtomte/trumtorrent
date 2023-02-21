package piece

import (
	"errors"
	"io/fs"
	"os"
	"path"
)

// Destination represents where data from a `Piece` should be written, since
// some times data will be split across multiple files
type Destination struct {
	Path   string
	Offset int
	Start  int
	End    int
}

// Block represents one part of a `Piece`
type Block struct {
	Index uint32
	Begin uint32
	Data  []byte
}

// Piece represents one (full) part of the torrent data
type Piece struct {
	Index          int
	Length         int
	Offset         int
	Data           []byte
	Received       int
	Requested      int
	QueuedRequests int
	Destinations   []Destination
}

func (p Piece) Incomplete() bool {
	return p.Received < p.Length
}

// CanQueueRequest is used when we request the blocks of a piece, we queue up to
// 5 requests at a time
func (p Piece) CanQueueRequest() bool {
	return p.Requested < p.Length && p.QueuedRequests < 5
}

// Reset is used when something unexpected (e.g. an error) happens and we need
// to put back the Piece in order for someone else (i.e. a Client) to take it
func (p *Piece) Reset() {
	p.Data = make([]byte, p.Length)
	p.Received = 0
	p.Requested = 0
	p.QueuedRequests = 0
}

func (p Piece) Write() error {
	// TODO: This will most likely be moved, so we can do batch writes instead
	for _, dst := range p.Destinations {
		err := os.MkdirAll(path.Dir(dst.Path), 0750)
		if err != nil && !errors.Is(err, fs.ErrExist) {
			return err
		}

		f, err := os.OpenFile(dst.Path, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}

		defer f.Close()

		data := p.Data[dst.Start:dst.End]
		if _, err := f.WriteAt(data, int64(dst.Offset)); err != nil {
			return err
		}
	}

	return nil
}
