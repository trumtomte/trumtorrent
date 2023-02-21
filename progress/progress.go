package progress

import (
	"fmt"
	"log"
	"time"
	"trumtorrent/piece"
	"trumtorrent/torrent"
)

type Progress struct {
	start      time.Time
	torrent    *torrent.Torrent
	downloaded int
	percent    string
}

func (p *Progress) timeElapsed() time.Duration {
	return time.Now().Sub(p.start)
}

func (p *Progress) Complete() bool {
	return p.torrent.Length() > 0 && p.torrent.Length() == p.downloaded
}

func (p *Progress) CalculateProgress(piece *piece.Piece) {
	p.downloaded += piece.Length
	percent := fmt.Sprintf("%.2f", float64(p.downloaded)/float64(p.torrent.Length())*100)

	if percent != p.percent {
		log.Printf("%v%% downloaded so far", percent)
	}

	p.percent = percent
}

func (p *Progress) Done() {
	log.Printf("Download finished after %v", p.timeElapsed())
}

func New(t *torrent.Torrent) *Progress {
	return &Progress{
		start:   time.Now(),
		torrent: t,
	}
}
