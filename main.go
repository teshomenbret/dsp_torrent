package main

import (
	"log"
	"os"

	"github.com/teshomenbret/torrent/leecher"
)

func main() {
	inPath := os.Args[1]
	torrentFile, err := leecher.OpenTorrentFile(inPath)
	if err != nil {
		log.Fatal(err)
	}
	err = torrentFile.DownloadTorrentFile()
	if err != nil {
		log.Fatal(err)
	}

}
