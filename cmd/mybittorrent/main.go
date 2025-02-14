package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode"

	. "github.com/codecrafters-io/bittorrent-starter-go/app"
)

func decodeBencodeString(s string) (BNode, int, error) {
	var firstColonIndex int

	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			firstColonIndex = i
			break
		}
	}

	lengthStr := s[:firstColonIndex]

	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return BNode{}, 0, err
	}

	r := BNode{Type: BString, Str: s[firstColonIndex+1 : firstColonIndex+1+length]}

	return r, firstColonIndex + length + 1, nil
}

func decodeBencodeInt(s string) (BNode, int, error) {
	endDelimiter := strings.Index(s, "e")
	s = s[1:endDelimiter]
	i, err := strconv.Atoi(s)

	if err != nil {
		return BNode{}, 0, err
	}

	return BNode{Type: BInt, Int: i}, endDelimiter + 1, nil
}

func decodeBencodeList(s string) (BNode, int, error) {
	r := BNode{Type: BList, List: make([]*BNode, 0)}
	totalSize := 2
	s = s[1:]

	for s[0] != 'e' {

		if s[0] == 'i' {
			item, l, err := decodeBencodeInt(s)
			if err != nil {
				return r, 0, err
			}
			r.List = append(r.List, &item)
			s = s[l:]
			totalSize += l
		} else if unicode.IsDigit(rune(s[0])) {
			item, l, err := decodeBencodeString(s)
			if err != nil {
				return r, 0, err
			}
			r.List = append(r.List, &item)
			s = s[l:]
			totalSize += l
		} else if s[0] == 'l' {
			item, l, err := decodeBencodeList(s)
			if err != nil {
				return r, 0, err
			}
			r.List = append(r.List, &item)
			totalSize += l
			s = s[l:]
		} else {
			return r, 0, fmt.Errorf("unknown bencoded value: %c", s[0])
		}
	}

	return r, totalSize, nil
}

func decodeBencodeDict(s string) (BNode, int, error) {
	r := BNode{Type: BDict, Dict: make(map[string]*BNode)}
	totalSize := 2
	s = s[1:]

	for s[0] != 'e' {
		key, l, err := decodeBencodeString(s)
		if err != nil {
			return r, 0, err
		}
		s = s[l:]
		totalSize += l

		if s[0] == 'i' {
			item, l, err := decodeBencodeInt(s)
			if err != nil {
				return r, 0, err
			}
			r.Dict[key.Str] = &item
			s = s[l:]
			totalSize += l
		} else if unicode.IsDigit(rune(s[0])) {
			item, l, err := decodeBencodeString(s)
			if err != nil {
				return r, 0, err
			}
			r.Dict[key.Str] = &item
			s = s[l:]
			totalSize += l
		} else if s[0] == 'l' {
			item, l, err := decodeBencodeList(s)
			if err != nil {
				return r, 0, err
			}
			r.Dict[key.Str] = &item
			s = s[l:]
			totalSize += l
		} else if s[0] == 'd' {
			item, l, err := decodeBencodeDict(s)
			if err != nil {
				return r, 0, err
			}
			r.Dict[key.Str] = &item
			s = s[l:]
		} else {
			return r, 0, fmt.Errorf("unknown bencoded value: %c", s[0])
		}
	}

	return r, totalSize, nil
}

// Example:
// - 5:hello -> hello
// - 10:hello12345 -> hello12345
func decodeBencode(bencodedString string) (BNode, error) {
	ch := bencodedString[0]
	switch ch {
	case 'i':
		i, _, err := decodeBencodeInt(bencodedString)
		return i, err
	case 'l':
		i, _, err := decodeBencodeList(bencodedString)
		return i, err
	case 'd':
		i, s, err := decodeBencodeDict(bencodedString)
		log.Println(s)
		return i, err
	default:
		if unicode.IsDigit(rune(ch)) {
			s, i, err := decodeBencodeString(bencodedString)
			log.Println(s, i)
			return s, err
		}
		return BNode{}, fmt.Errorf("unknown bencoded value: %c", ch)
	}
}

// MetaInfo holds all metadata related information for the given torrent.
type MetaInfo struct {
	TrackerUrl string
	Length     int
}

// ParseTorrentFile parses a torrent file to a MetaInfo object.
func ParseTorrentFile(filePath string) (MetaInfo, error) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return MetaInfo{}, err
	}

	decodedTorrent, _, err := decodeBencodeDict(string(file))
	if err != nil {
		return MetaInfo{}, err
	}
	info := decodedTorrent.Dict["info"]

	result := MetaInfo{
		TrackerUrl: decodedTorrent.Dict["announce"].Str,
		Length:     info.Dict["length"].Int,
	}

	return result, nil
}

// MarshalBNode encodes a BNode into JSON format.
func MarshalBNode(node *BNode) ([]byte, error) {
	var b []byte
	var err error

	switch node.Type {
	case BString:
		b, err = json.Marshal(node.Str)

	case BInt:
		b, err = json.Marshal(node.Int)

	case BList:
		// Convert list elements to JSON
		encodedList := make([]json.RawMessage, 0)
		for _, item := range node.List {
			encodedItem, err := MarshalBNode(item)
			if err != nil {
				return nil, err
			}
			encodedList = append(encodedList, encodedItem)
		}
		b, err = json.Marshal(encodedList)

	case BDict:
		// Convert dictionary values to JSON
		encodedDict := make(map[string]json.RawMessage)
		for key, value := range node.Dict {
			encodedItem, err := MarshalBNode(value)
			if err != nil {
				return nil, err
			}
			encodedDict[key] = encodedItem
		}
		b, err = json.Marshal(encodedDict)
	}

	return b, err
}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Fprintln(os.Stderr, "Starting application...")

	command := os.Args[1]

	if command == "decode" {
		bencodedValue := os.Args[2]

		decoded, err := decodeBencode(bencodedValue)
		if err != nil {
			fmt.Println(err)
			return
		}

		jsonOutput, err := MarshalBNode(&decoded)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(string(jsonOutput))
	} else if command == "info" {
		torrentFilePath := os.Args[2]

		metaInfo, err := ParseTorrentFile(torrentFilePath)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Tracker URL:", metaInfo.TrackerUrl)
		fmt.Println("Length:", metaInfo.Length)
	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
