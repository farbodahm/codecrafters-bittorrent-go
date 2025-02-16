package main

import (
	"crypto/sha1"
	"fmt"
	"log"
	"os"

	. "github.com/codecrafters-io/bittorrent-starter-go/app"
)

// MetaInfo holds all metadata related information for the given torrent.
type MetaInfo struct {
	TrackerUrl  string
	Length      int
	InfoHashHex string
	PieceLength int
	Pieces      []string
}

// CalculateInfoHash calculates the SHA1 hash of the Bencoded value of `info` dictionary from torrent file.
func CalculateInfoHash(infoDict BNode) string {
	encodedInfo := EncodeBNode(infoDict)
	h := sha1.New()
	h.Write([]byte(encodedInfo))
	encodedInfo = h.Sum(nil)

	return fmt.Sprintf("%x", encodedInfo)
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
		InfoHashHex: CalculateInfoHash(*info),
		PieceLength: info.Dict["piece length"].Int,
		Pieces:      pieces,
	}

	return result, nil
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
		fmt.Println("Info Hash:", metaInfo.InfoHashHex)
		fmt.Println("Piece Length:", metaInfo.PieceLength)
		fmt.Println("Piece Hashes:")
		for _, piece := range metaInfo.Pieces {
			fmt.Printf("%x\n", piece)
		}

	default:
		// Handle unknown commands
		log.Fatalf("Unknown command: %s", command)
	}
}
