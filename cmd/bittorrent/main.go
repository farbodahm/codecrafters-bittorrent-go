package main

import (
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	. "github.com/codecrafters-io/bittorrent-starter-go/app"
)

// MetaInfo holds all metadata related information for the given torrent.
type MetaInfo struct {
	TrackerUrl  string
	Length      int
	InfoHash    []byte
	PieceLength int
	Pieces      []string
}

// CalculateInfoHash calculates the SHA1 hash of the Bencoded value of `info` dictionary from torrent file.
func CalculateInfoHash(infoDict BNode) []byte {
	encodedInfo := EncodeBNode(infoDict)
	h := sha1.New()
	h.Write([]byte(encodedInfo))
	encodedInfo = h.Sum(nil)

	return encodedInfo
}

// ParseTorrentFile parses a torrent file to a MetaInfo object.
func ParseTorrentFile(filePath string) (MetaInfo, error) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return MetaInfo{}, err
	}

	decodedTorrent, _, err := DecodeBencodeDict(string(file))
	if err != nil {
		return MetaInfo{}, err
	}
	info := decodedTorrent.Dict["info"]

	// Separate each piece, Each piece is 20 bytes long
	piecesStr := info.Dict["pieces"].Str
	pieces := make([]string, 0)
	for i := 0; i < len(piecesStr); i += 20 {
		pieces = append(pieces, piecesStr[i:i+20])
	}

	result := MetaInfo{
		TrackerUrl:  decodedTorrent.Dict["announce"].Str,
		Length:      info.Dict["length"].Int,
		InfoHash:    CalculateInfoHash(*info),
		PieceLength: info.Dict["piece length"].Int,
		Pieces:      pieces,
	}

	return result, nil
}

func GetPeers(info MetaInfo) ([]string, error) {
	params := url.Values{}
	log.Println(len(info.InfoHash))
	log.Println(info.InfoHash)
	log.Println(string(info.InfoHash))
	params.Add("info_hash", url.QueryEscape(string(info.InfoHash)))
	params.Add("peer_id", "XX9911AA22ZZAAFFII22") // TODO: Replace with a proper random generator
	params.Add("port", "6881")
	params.Add("uploaded", "0")
	params.Add("downloaded", "0")
	params.Add("left", fmt.Sprintf("%d", info.Length))
	params.Add("compact", "1")

	fullUrl := info.TrackerUrl + "?" + params.Encode()

	resp, err := http.Get(fullUrl)
	if err != nil {
		log.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response: %v", err)
	}

	decoded, err := DecodeBencode(string(body))
	if err != nil {
		log.Fatalf("Failed to decode response: %v", err)
	}

	// Parse peers URL with ports; Each peer is 6 bytes, 4 bytes URL 2 bytes port
	peersStr := decoded.Dict["peers"].Str
	peers := make([]string, 0)
	for i := 0; i < len(peersStr); i += 6 {
		peer := peersStr[i : i+6]
		peerUrl := fmt.Sprintf("%d.%d.%d.%d", peer[0], peer[1], peer[2], peer[3])
		peerPort := int(peer[4])*256 + int(peer[5])
		peers = append(peers, fmt.Sprintf("%s:%d", peerUrl, peerPort))
	}

	return peers, nil
}

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
		fmt.Printf("Info Hash:%x\n", metaInfo.InfoHash)
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

	default:
		// Handle unknown commands
		log.Fatalf("Unknown command: %s", command)
	}
}
