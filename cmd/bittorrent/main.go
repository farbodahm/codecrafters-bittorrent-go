package main

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"net"
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

// GetPeers returns list of all available peers asking from the tracker.
func GetPeers(info MetaInfo) ([]string, error) {
	params := url.Values{}
	params.Add("info_hash", string(info.InfoHash))
	params.Add("peer_id", GenerateRandomID(20))
	params.Add("port", "6881")
	params.Add("uploaded", "0")
	params.Add("downloaded", "0")
	params.Add("left", fmt.Sprintf("%d", info.Length))
	params.Add("compact", "1")

	fullUrl := info.TrackerUrl + "?" + params.Encode()

	resp, err := http.Get(fullUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	decoded, err := DecodeBencode(string(body))
	if err != nil {
		return nil, fmt.Errorf("failed to decode response body: %v", err)
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

// HandshakePeer establishes a TCP connection with a peer and performs the BitTorrent handshake.
func HandshakePeer(addr string, info MetaInfo) ([]byte, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to peer: %v", err)
	}
	defer conn.Close()

	// Write handshake message
	var msg bytes.Buffer
	msg.WriteByte(19)
	msg.WriteString("BitTorrent protocol")
	msg.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0})
	msg.Write(info.InfoHash)
	msg.WriteString(GenerateRandomID(20))

	_, err = conn.Write(msg.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to send handshake message: %v", err)
	}

	// According to BitTorrent spec, we expect 68 bytes as handshake answer.
	// https://www.bittorrent.org/beps/bep_0003.html#peer-protocol
	resp := make([]byte, 68)
	n, err := conn.Read(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to read handshake response: %v", err)
	}

	if n != 68 || resp[0] != 19 {
		return nil, fmt.Errorf("invalid handshake response")
	}
	peerID := resp[48:]

	return peerID, nil
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

		peerId, err := HandshakePeer(os.Args[3], metaInfo)
		if err != nil {
			log.Fatalf("Failed to handshake peer: %v", err)
		}
		fmt.Printf("Peer ID: %x\n", peerId)

	default:
		// Handle unknown commands
		log.Fatalf("Unknown command: %s", command)
	}
}
