package app

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// DecodeBencodeString decodes a bencoded string and returns a BNode, parsed length, and an error if any.
func DecodeBencodeString(s string) (BNode, int, error) {
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

// DecodeBencodeInt decodes a bencoded integer and returns a BNode, parsed length, and an error if any.
func DecodeBencodeInt(s string) (BNode, int, error) {
	endDelimiter := strings.Index(s, "e")
	s = s[1:endDelimiter]
	i, err := strconv.Atoi(s)

	if err != nil {
		return BNode{}, 0, err
	}

	return BNode{Type: BInt, Int: i}, endDelimiter + 1, nil
}

// DecodeBencodeList decodes a bencoded list and returns a BNode, parsed length, and an error if any.
func DecodeBencodeList(s string) (BNode, int, error) {
	r := BNode{Type: BList, List: make([]*BNode, 0)}
	totalSize := 2
	s = s[1:]

	for s[0] != 'e' {

		if s[0] == 'i' {
			item, l, err := DecodeBencodeInt(s)
			if err != nil {
				return r, 0, err
			}
			r.List = append(r.List, &item)
			s = s[l:]
			totalSize += l
		} else if unicode.IsDigit(rune(s[0])) {
			item, l, err := DecodeBencodeString(s)
			if err != nil {
				return r, 0, err
			}
			r.List = append(r.List, &item)
			s = s[l:]
			totalSize += l
		} else if s[0] == 'l' {
			item, l, err := DecodeBencodeList(s)
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

// DecodeBencodeDict decodes a bencoded dictionary and returns a BNode, parsed length, and an error if any.
func DecodeBencodeDict(s string) (BNode, int, error) {
	r := BNode{Type: BDict, Dict: make(map[string]*BNode)}
	totalSize := 2
	s = s[1:]

	for s[0] != 'e' {
		key, l, err := DecodeBencodeString(s)
		if err != nil {
			return r, 0, err
		}
		s = s[l:]
		totalSize += l

		if s[0] == 'i' {
			item, l, err := DecodeBencodeInt(s)
			if err != nil {
				return r, 0, err
			}
			r.Dict[key.Str] = &item
			s = s[l:]
			totalSize += l
		} else if unicode.IsDigit(rune(s[0])) {
			item, l, err := DecodeBencodeString(s)
			if err != nil {
				return r, 0, err
			}
			r.Dict[key.Str] = &item
			s = s[l:]
			totalSize += l
		} else if s[0] == 'l' {
			item, l, err := DecodeBencodeList(s)
			if err != nil {
				return r, 0, err
			}
			r.Dict[key.Str] = &item
			s = s[l:]
			totalSize += l
		} else if s[0] == 'd' {
			item, l, err := DecodeBencodeDict(s)
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

// DecodeBencode decodes a complete bencoded string and returns a BNode and an error if any.
func DecodeBencode(bencodedString string) (BNode, error) {
	ch := bencodedString[0]
	switch ch {
	case 'i':
		i, _, err := DecodeBencodeInt(bencodedString)
		return i, err
	case 'l':
		i, _, err := DecodeBencodeList(bencodedString)
		return i, err
	case 'd':
		i, _, err := DecodeBencodeDict(bencodedString)
		return i, err
	default:
		if unicode.IsDigit(rune(ch)) {
			s, _, err := DecodeBencodeString(bencodedString)
			return s, err
		}
		return BNode{}, fmt.Errorf("unknown bencoded value: %c", ch)
	}
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

// EncodeBNode encodes a BNode into a bencode-encoded byte slice based on its type.
func EncodeBNode(node BNode) []byte {
	switch node.Type {
	case BString:
		return EncodeBencodeString(node.Str)
	case BInt:
		return EncodeBencodeInt(node.Int)
	case BList:
		return EncodeBencodeList(node)
	case BDict:
		return EncodeBencodeDict(node)
	default:
		log.Fatalf("Unknown node type: %d", node.Type)
	}
	return nil
}

// EncodeBencodeString encodes a string into bencode format.
func EncodeBencodeString(s string) []byte {
	// TODO: Check if we have to store as []byte instead of string in the BNode
	return []byte(fmt.Sprintf("%d:%s", len(s), s))
}

// EncodeBencodeInt encodes an integer into bencode format.
func EncodeBencodeInt(i int) []byte {
	return []byte(fmt.Sprintf("i%de", i))
}

// EncodeBencodeList encodes a list of BNodes into bencode format.
func EncodeBencodeList(node BNode) []byte {
	encodedList := []byte("l")
	for _, item := range node.List {
		encodedItem := EncodeBNode(*item)
		encodedList = append(encodedList, encodedItem...)
	}
	encodedList = append(encodedList, 'e')
	return encodedList
}

// EncodeBencodeDict encodes a dictionary of BNodes into bencode format.
func EncodeBencodeDict(node BNode) []byte {
	encodedDict := []byte("d")

	// Extract and sort keys lexicographically
	keys := make([]string, 0, len(node.Dict))
	for key := range node.Dict {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Encode each key-value pair in sorted order
	for _, key := range keys {
		encodedKey := EncodeBencodeString(key)
		encodedValue := EncodeBNode(*node.Dict[key])
		encodedDict = append(encodedDict, encodedKey...)
		encodedDict = append(encodedDict, encodedValue...)
	}

	encodedDict = append(encodedDict, 'e')
	return encodedDict
}
