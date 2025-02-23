package app

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

// PieceLength is the length of each piece in bytes.
const PieceLength = 16 * 1024

// PeerMessage represents a message from/to a peer.
type PeerMessage struct {
	Length  int
	ID      int
	Payload []byte
}

// Tries downloading a piece from multiple peers if needed.
func DownloadPiece(info MetaInfo, index int, path string) error {
	peers, err := GetPeers(info)
	if err != nil {
		return fmt.Errorf("failed to get peers: %v", err)
	}

	for _, peer := range peers {
		log.Printf("Trying peer: %s", peer)
		err := downloadPieceFromPeer(peer, info, index, path)
		if err == nil {
			log.Printf("Successfully downloaded piece %d from peer %s", index, peer)
			return nil
		}
		log.Printf("Failed with peer %s, trying next...", peer)
	}

	return fmt.Errorf("failed to download piece %d from all peers", index)
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

// HandshakePeer performs the BitTorrent handshake and returns the peer ID.
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

	// Read handshake response (exactly 68 bytes)
	resp, err := readExactBytes(conn, 68)
	if err != nil {
		return nil, fmt.Errorf("failed to read handshake response: %v", err)
	}

	if resp[0] != 19 {
		return nil, fmt.Errorf("invalid handshake response")
	}
	peerID := resp[48:]

	return peerID, nil
}

// readExactBytes reads exactly 'size' bytes from the connection.
func readExactBytes(conn net.Conn, size int) ([]byte, error) {
	buf := make([]byte, size)
	_, err := io.ReadFull(conn, buf)
	if err != nil {
		return nil, fmt.Errorf("failed to read %d bytes: %v", size, err)
	}
	return buf, nil
}

// recievePeerMessage reads a PeerMessage from a peer.
func recievePeerMessage(conn net.Conn) (PeerMessage, error) {
	// Read message length (4 bytes)
	lengthBytes, err := readExactBytes(conn, 4)
	if err != nil {
		return PeerMessage{}, fmt.Errorf("failed to read message length: %v", err)
	}
	length := int(lengthBytes[0])<<24 | int(lengthBytes[1])<<16 | int(lengthBytes[2])<<8 | int(lengthBytes[3])

	// Read message ID (1 byte)
	idBytes, err := readExactBytes(conn, 1)
	if err != nil {
		return PeerMessage{}, fmt.Errorf("failed to read message ID: %v", err)
	}
	id := int(idBytes[0])

	// Read message payload (length - 1 bytes)
	payload, err := readExactBytes(conn, length-1)
	if err != nil {
		return PeerMessage{}, fmt.Errorf("failed to read message payload: %v", err)
	}

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

	if msg.ID != 7 {
		return nil, fmt.Errorf("expected piece message 7, got: %d", msg.ID)
	}

	// return block data of the payload; first 8 bytes are index, begin and block data
	return msg.Payload[8:], nil
}

// downloadPiece downloads a piece from a peer.
func downloadPiece(conn net.Conn, index, pieceLength, totalFileSize int) ([]byte, error) {
	// Check if this is the last piece
	isLastPiece := (index+1)*pieceLength >= totalFileSize

	// Determine actual piece size for this index
	totalPieceLen := pieceLength // Default to full size
	if isLastPiece {
		totalPieceLen = totalFileSize - (index * pieceLength) // Calculate remaining bytes
	}

	log.Printf("Downloading piece %d - Total Piece Length: %d, Is Last Piece: %v\n", index, totalPieceLen, isLastPiece)

	buffer := make([]byte, totalPieceLen)

	for i := 0; i < totalPieceLen; i += PieceLength {
		// Determine block size dynamically
		remainingSize := totalPieceLen - i
		downloadLen := min(PieceLength, remainingSize)

		log.Printf("Requesting block - Start: %d, Length: %d, Remaining: %d\n", i, downloadLen, remainingSize)

		err := sendRequestMessage(conn, index, i, downloadLen)
		if err != nil {
			return nil, fmt.Errorf("failed to send request message: %v", err)
		}

		block, err := recievePieceMessage(conn)
		if err != nil {
			return nil, fmt.Errorf("failed to receive piece message: %v", err)
		}

		if len(block) != downloadLen {
			return nil, fmt.Errorf("unexpected block size: expected %d, got %d", downloadLen, len(block))
		}

		copy(buffer[i:i+downloadLen], block)
	}

	return buffer, nil
}

// Attempts to download a piece from a single peer.
func downloadPieceFromPeer(peer string, info MetaInfo, index int, path string) error {
	conn, err := net.DialTimeout("tcp", peer, 5*time.Second)
	if err != nil {
		log.Printf("Failed to connect to peer %s: %v", peer, err)
		return err
	}
	defer conn.Close()

	_, err = HandshakePeer(conn, info)
	if err != nil {
		log.Printf("Handshake failed with peer %s: %v", peer, err)
		return err
	}
	log.Println("Handshake successful, downloading piece...")

	// **Preserving all steps before downloading the piece**
	_, err = readBitFieldMessage(conn)
	if err != nil {
		log.Printf("Failed to read bitfield from peer %s: %v", peer, err)
		return err
	}

	err = sendInterestedMessage(conn)
	if err != nil {
		log.Printf("Failed to send interested message to peer %s: %v", peer, err)
		return err
	}

	_, err = receiveUnchokeMessage(conn)
	if err != nil {
		log.Printf("Failed to receive unchoke message from peer %s: %v", peer, err)
		return err
	}

	// Create a file to save the downloaded piece
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	// Download the piece
	piece, err := downloadPiece(conn, index, info.PieceLength, info.Length)
	if err != nil {
		log.Printf("Failed to download piece from peer %s: %v", peer, err)
		return err
	}

	// Write the downloaded piece to the file
	_, err = file.Write(piece)
	if err != nil {
		return fmt.Errorf("failed to write piece to file: %v", err)
	}

	return nil
}
