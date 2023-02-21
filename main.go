package main

import (
	"fmt"
	"os"
	_ "time"
	"trumtorrent/download"
	"trumtorrent/torrent"
)

// TODO: write more tests
// TODO: write the download.Writer
// TODO: add support for setting a custom output path

func main() {
	// path := "starwars.torrent"
	// path := "magnet:?xt=urn:btih:dd02dc8713ca6edfc7dd21d0bf5da58834559a7c&dn=bilder&tr=udp%3A%2F%2Ftracker.leechers-paradise.org%3A6969&tr=udp%3A%2F%2Ftracker.coppersurfer.tk%3A6969&tr=udp%3A%2F%2Ftracker.opentrackr.org%3A1337&tr=udp%3A%2F%2Fexplodie.org%3A6969&tr=udp%3A%2F%2Ftracker.empire-js.us%3A1337&tr=wss%3A%2F%2Ftracker.btorrent.xyz&tr=wss%3A%2F%2Ftracker.openwebtorrent.com"
	path := os.Args[1]

	t, err := torrent.Open(path)
	if err != nil {
		fmt.Println(err)
		return
	}

	manager := download.NewManager(t)
	manager.Download()
}
