package app

import (
	"log"
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
			result.InfoHash = []byte(strings.TrimPrefix(part, "xt=urn:btih:"))
		} else if strings.HasPrefix(part, "tr=") {
			log.Println("here", part)
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
