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
	"strconv"

	. "github.com/codecrafters-io/bittorrent-starter-go/app"
)

// PieceLength is the length of each piece in bytes.
const PieceLength = 16 * 1024

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

// HandshakePeer performs the BitTorrent handshake and retyrns the peer ID.
func HandshakePeer(conn net.Conn, info MetaInfo) ([]byte, error) {
	// Write handshake message
	var msg bytes.Buffer
	msg.WriteByte(19)
	msg.WriteString("BitTorrent protocol")
	msg.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0})
	msg.Write(info.InfoHash)
	msg.WriteString(GenerateRandomID(20))

	_, err := conn.Write(msg.Bytes())
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

// PeerMessage represents a message from/to a peer.
type PeerMessage struct {
	Length  int
	ID      int
	Payload []byte
}

// recuievePeerMessage reads a PeerMessage from a peer.
func recievePeerMessage(conn net.Conn) (PeerMessage, error) {
	// Read message length
	lengthBytes := make([]byte, 4)
	_, err := conn.Read(lengthBytes)
	if err != nil {
		return PeerMessage{}, fmt.Errorf("failed to read message length: %v", err)
	}
	length := int(lengthBytes[0])<<24 | int(lengthBytes[1])<<16 | int(lengthBytes[2])<<8 | int(lengthBytes[3])

	// Read message ID
	idBytes := make([]byte, 1)
	_, err = conn.Read(idBytes)
	if err != nil {
		return PeerMessage{}, fmt.Errorf("failed to read message ID: %v", err)
	}
	id := int(idBytes[0])

	// Read message payload
	payload := make([]byte, length-1)
	n, err := conn.Read(payload)
	if err != nil {
		return PeerMessage{}, fmt.Errorf("failed to read message payload: %v", err)
	}

	log.Println("-- Expected length:", length)
	log.Println("-- Actual length:", n+1)

	return PeerMessage{Length: length, ID: id, Payload: payload}, nil
}

// sendPeerMessage sends a PeerMessage to a peer.
func sendPeerMessage(conn net.Conn, msg PeerMessage) error {
	// Write message length
	lengthBytes := []byte{byte(msg.Length >> 24), byte(msg.Length >> 16), byte(msg.Length >> 8), byte(msg.Length)}
	_, err := conn.Write(lengthBytes)
	if err != nil {
		return fmt.Errorf("failed to write message length: %v", err)
	}

	// Write message ID
	_, err = conn.Write([]byte{byte(msg.ID)})
	if err != nil {
		return fmt.Errorf("failed to write message ID: %v", err)
	}

	// Write message payload
	_, err = conn.Write(msg.Payload)
	if err != nil {
		return fmt.Errorf("failed to write message payload: %v", err)
	}

	return nil
}

// readBitFieldMessage reads the bitfield message from a peer.
func readBitFieldMessage(conn net.Conn) (PeerMessage, error) {
	msg, err := recievePeerMessage(conn)
	if err != nil {
		return PeerMessage{}, fmt.Errorf("failed to read bitfield message: %v", err)
	}

	if msg.ID != 5 {
		return PeerMessage{}, fmt.Errorf("expected bitfield message 5, got: %d", msg.ID)
	}

	return msg, nil
}

// sendInterestedMessage sends the interested message to a peer.
func sendInterestedMessage(conn net.Conn) error {
	msg := PeerMessage{Length: 1, ID: 2}
	return sendPeerMessage(conn, msg)
}

// receiveUnchokeMessage reads the unchoke message from a peer.
func receiveUnchokeMessage(conn net.Conn) (PeerMessage, error) {
	msg, err := recievePeerMessage(conn)
	if err != nil {
		return PeerMessage{}, fmt.Errorf("failed to read unchoke message: %v", err)
	}

	if msg.ID != 1 {
		return PeerMessage{}, fmt.Errorf("expected unchoke message 1, got: %d", msg.ID)
	}

	return msg, nil
}

// sendRequestMessage sends the request message to a peer.
func sendRequestMessage(conn net.Conn, index, begin, length int) error {
	msg := PeerMessage{
		Length: 13,
		ID:     6,
		Payload: []byte{
			byte(index >> 24), byte(index >> 16), byte(index >> 8), byte(index),
			byte(begin >> 24), byte(begin >> 16), byte(begin >> 8), byte(begin),
			byte(length >> 24), byte(length >> 16), byte(length >> 8), byte(length),
		},
	}

	return sendPeerMessage(conn, msg)
}

// recievePieceMessage reads the piece message from a peer.
func recievePieceMessage(conn net.Conn) ([]byte, error) {
	msg, err := recievePeerMessage(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to read piece message: %v", err)
	}

	log.Println("Received piece message:", msg.ID)
	log.Println("Payload length:", len(msg.Payload))

	if msg.ID != 7 {
		return nil, fmt.Errorf("expected piece message 7, got: %d", msg.ID)
	}

	// return block data of the payload; first 8 bytes are index, begin and block data
	return msg.Payload[8:], nil
}

// downloadPiece downloads a piece from a peer.
func downloadPiece(conn net.Conn, index, totalPieceLen int) ([]byte, error) {
	buffer := make([]byte, totalPieceLen)
	for i := 0; i < totalPieceLen; i += PieceLength {
		log.Println("Downloading block:", i)
		log.Println("Total piece length:", totalPieceLen)
		log.Println("Piece length:", PieceLength)
		log.Println("Index:", index)
		log.Println("Remaining:", totalPieceLen-i)
		err := sendRequestMessage(conn, index, i, PieceLength)
		if err != nil {
			return nil, fmt.Errorf("failed to send request message: %v", err)
		}

		block, err := recievePieceMessage(conn)
		if err != nil {
			return nil, fmt.Errorf("failed to receive piece message: %v", err)
		}

		copy(buffer[i:], block)
	}

	return buffer, nil
}

// DownloadPiece downloads the piece from the peer after sending initial messages.
// It saves the downloaded piece into a file.
// NOTE: DownloadPiece assumes handshake is already done.
func DownloadPiece(conn net.Conn, index, totalPieceLen int, path string) error {
	log.Println("Total piece length:", totalPieceLen)
	_, err := readBitFieldMessage(conn)
	if err != nil {
		return fmt.Errorf("failed to read bitfield message: %v", err)
	}

	err = sendInterestedMessage(conn)
	if err != nil {
		return fmt.Errorf("failed to send interested message: %v", err)
	}

	_, err = receiveUnchokeMessage(conn)
	if err != nil {
		return fmt.Errorf("failed to receive unchoke message: %v", err)
	}

	// Create a file to save the downloaded piece
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	// Download the piece
	piece, err := downloadPiece(conn, index, totalPieceLen)
	if err != nil {
		return fmt.Errorf("failed to download piece: %v", err)
	}

	// Write the downloaded piece to the file
	_, err = file.Write(piece)
	if err != nil {
		return fmt.Errorf("failed to write piece to file: %v", err)
	}

	return nil
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

		peers, err := GetPeers(metaInfo)
		if err != nil {
			log.Fatalf("Failed to get peers: %v", err)
		}

		conn, err := net.Dial("tcp", peers[0])
		if err != nil {
			log.Fatalf("Failed to connect to peer: %v", err)
		}
		defer conn.Close()

		_, err = HandshakePeer(conn, metaInfo)
		if err != nil {
			log.Fatalf("Failed to handshake peer: %v", err)
		}
		log.Println("Handshake successful, downloading piece...")

		err = DownloadPiece(conn, pieceIndex, metaInfo.PieceLength, resultFilePath)
		if err != nil {
			log.Fatalf("Failed to download piece: %v", err)
		}

	default:
		// Handle unknown commands
		log.Fatalf("Unknown command: %s", command)
	}
}
