package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	. "github.com/codecrafters-io/bittorrent-starter-go/app"
)

func main() {
	// Codecrafters read Stdout for answer! So we have to write logs to Stderr!
	fmt.Fprintln(os.Stderr, "Starting application...")
	command := os.Args[1]

	switch command {
	case "decode":
		bencodedValue := os.Args[2]
		decoded, err := DecodeBencode(bencodedValue)
		if err != nil {
			log.Fatalf("Failed to decode bencoded value: %v", err)
		}

		// Marshal the decoded value to JSON
		jsonOutput, err := MarshalBNode(&decoded)
		if err != nil {
			log.Fatalf("Failed to marshal decoded value: %v", err)
		}

		fmt.Println(string(jsonOutput))

	case "info":
		torrentFilePath := os.Args[2]

		// Parse the torrent file
		metaInfo, err := ParseTorrentFile(torrentFilePath)
		if err != nil {
			log.Fatalf("Failed to parse torrent file: %v", err)
		}

		// Print torrent file info
		fmt.Println("Tracker URL:", metaInfo.TrackerUrl)
		fmt.Println("Length:", metaInfo.Length)
		fmt.Printf("Info Hash: %x\n", metaInfo.InfoHash)
		fmt.Println("Piece Length:", metaInfo.PieceLength)
		fmt.Println("Piece Hashes:")
		for _, piece := range metaInfo.Pieces {
			fmt.Printf("%x\n", piece)
		}

	case "peers":
		torrentFilePath := os.Args[2]
		metaInfo, err := ParseTorrentFile(torrentFilePath)
		if err != nil {
			log.Fatalf("Failed to parse torrent file: %v", err)
		}

		peers, err := GetPeers(metaInfo)
		if err != nil {
			log.Fatalf("Failed to get peers: %v", err)
		}
		for _, peer := range peers {
			fmt.Println(peer)
		}

	case "handshake":
		torrentFilePath := os.Args[2]
		metaInfo, err := ParseTorrentFile(torrentFilePath)
		if err != nil {
			log.Fatalf("Failed to parse torrent file: %v", err)
		}

		conn, err := net.Dial("tcp", os.Args[3])
		if err != nil {
			log.Fatalf("Failed to connect to peer: %v", err)
		}
		defer conn.Close()

		peerId, err := HandshakePeer(conn, metaInfo)
		if err != nil {
			log.Fatalf("Failed to handshake peer: %v", err)
		}
		fmt.Printf("Peer ID: %x\n", peerId)

	case "download_piece":
		torrentFilePath := os.Args[4]
		resultFilePath := os.Args[3]

		pieceIndex, err := strconv.Atoi(os.Args[5])
		if err != nil {
			log.Fatalf("Failed to parse piece index: %v", err)
		}

		metaInfo, err := ParseTorrentFile(torrentFilePath)
		if err != nil {
			log.Fatalf("Failed to parse torrent file: %v", err)
		}

		err = DownloadPiece(metaInfo, pieceIndex, resultFilePath)
		if err != nil {
			log.Fatalf("Download failed: %v", err)
		}

	case "download":
		resultFilePath := os.Args[3]
		torrentFilePath := os.Args[4]

		metaInfo, err := ParseTorrentFile(torrentFilePath)
		if err != nil {
			log.Fatalf("Failed to parse torrent file: %v", err)
		}

		err = DownloadFile(metaInfo, resultFilePath)
		if err != nil {
			log.Fatalf("Download failed: %v", err)
		}

	case "magnet_parse":
		magnetLink := os.Args[2]
		metaInfo, err := ParseMagnetLink(magnetLink)
		if err != nil {
			log.Fatalf("Failed to parse magnet link: %v", err)
		}

		fmt.Println("Tracker URL:", metaInfo.TrackerUrl)
		fmt.Println("Info Hash:", string(metaInfo.InfoHash))

	default:
		// Handle unknown commands
		log.Fatalf("Unknown command: %s", command)
	}
}
