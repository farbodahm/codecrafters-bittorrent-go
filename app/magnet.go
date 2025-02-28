package app

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// MagnetMetaInfo represents metadata for a magnet link.
type MagnetMetaInfo struct {
	TrackerUrl string
	InfoHash   []byte
	FileName   string
}

// ParseMagnetLink parses a magnet link to a MagnetMetaInfo object.
func ParseMagnetLink(magnetLink string) (MagnetMetaInfo, error) {
	magnetLink = strings.TrimPrefix(magnetLink, "magnet:?")

	// Split the magnet link into parts
	parts := strings.Split(magnetLink, "&")

	result := MagnetMetaInfo{}

	for _, part := range parts {
		if strings.HasPrefix(part, "xt=urn:btih:") {
			hexInfohash := []byte(strings.TrimPrefix(part, "xt=urn:btih:"))
			result.InfoHash = make([]byte, hex.DecodedLen(len(hexInfohash)))
			_, err := hex.Decode(result.InfoHash, hexInfohash)
			if err != nil {
				return MagnetMetaInfo{}, err
			}
		} else if strings.HasPrefix(part, "tr=") {
			url, err := url.QueryUnescape(strings.TrimPrefix(part, "tr="))
			if err != nil {
				return MagnetMetaInfo{}, err
			}
			result.TrackerUrl = url
		} else if strings.HasPrefix(part, "dn=") {
			result.FileName = strings.TrimPrefix(part, "dn=")
		}
	}

	return result, nil
}

// GetMagnetPeers returns list of all available peers asking from the tracker.
func GetMagnetPeers(info MagnetMetaInfo) ([]string, error) {
	params := url.Values{}
	params.Add("info_hash", string(info.InfoHash))
	params.Add("peer_id", GenerateRandomID(20))
	params.Add("port", "6881")
	params.Add("uploaded", "0")
	params.Add("downloaded", "0")
	params.Add("left", "1")
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

// HandshakeMagnetPeer performs the BitTorrent handshake and returns the peer ID.
func HandshakeMagnetPeer(conn net.Conn, info MagnetMetaInfo) ([]byte, error) {
	var msg bytes.Buffer
	msg.WriteByte(19)
	msg.WriteString("BitTorrent protocol")
	msg.Write([]byte{0, 0, 0, 0, 0, 0x10, 0, 0})
	msg.Write(info.InfoHash)
	msg.WriteString(GenerateRandomID(20))

	_, err := conn.Write(msg.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to send handshake message: %v", err)
	}

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
