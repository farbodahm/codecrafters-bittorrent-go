package app

import (
	"crypto/sha1"
	"os"
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
