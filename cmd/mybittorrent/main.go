package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode"
)

func decodeBencodeString(s string) (string, int, error) {
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
		return "", 0, err
	}

	return s[firstColonIndex+1 : firstColonIndex+1+length], firstColonIndex + length + 1, nil
}

func decodeBencodeInt(s string) (int, int, error) {
	endDelimiter := strings.Index(s, "e")
	s = s[1:endDelimiter]
	i, err := strconv.Atoi(s)

	if err != nil {
		return -1, 0, err
	}

	return i, endDelimiter + 1, nil
}

func decodeBencodeList(s string) ([]interface{}, int, error) {
	list := make([]interface{}, 0)
	totalSize := 2
	s = s[1:]

	for s[0] != 'e' {

		if s[0] == 'i' {
			item, l, err := decodeBencodeInt(s)
			if err != nil {
				return nil, 0, err
			}
			list = append(list, item)
			s = s[l:]
			totalSize += l
		} else if unicode.IsDigit(rune(s[0])) {
			item, l, err := decodeBencodeString(s)
			if err != nil {
				return nil, 0, err
			}
			list = append(list, item)
			s = s[l:]
			totalSize += l
		} else if s[0] == 'l' {
			item, l, err := decodeBencodeList(s)
			if err != nil {
				return nil, 0, err
			}
			list = append(list, item)
			totalSize += l
			s = s[l:]
		} else {
			return nil, 0, fmt.Errorf("unknown bencoded value: %c", s[0])
		}
	}

	return list, totalSize, nil
}

func decodeBencodeDict(s string) (map[string]interface{}, int, error) {
	dict := make(map[string]interface{})
	totalSize := 2
	s = s[1:]

	for s[0] != 'e' {
		key, l, err := decodeBencodeString(s)
		if err != nil {
			return nil, 0, err
		}
		s = s[l:]
		totalSize += l

		if s[0] == 'i' {
			item, l, err := decodeBencodeInt(s)
			if err != nil {
				return nil, 0, err
			}
			dict[key] = item
			s = s[l:]
			totalSize += l
		} else if unicode.IsDigit(rune(s[0])) {
			item, l, err := decodeBencodeString(s)
			if err != nil {
				return nil, 0, err
			}
			dict[key] = item
			s = s[l:]
			totalSize += l
		} else if s[0] == 'l' {
			item, l, err := decodeBencodeList(s)
			if err != nil {
				return nil, 0, err
			}
			dict[key] = item
			s = s[l:]
			totalSize += l
		} else if s[0] == 'd' {
			item, l, err := decodeBencodeDict(s)
			if err != nil {
				return nil, 0, err
			}
			dict[key] = item
			s = s[l:]
		} else {
			return nil, 0, fmt.Errorf("unknown bencoded value: %c", s[0])
		}
	}

	return dict, totalSize, nil
}

// Example:
// - 5:hello -> hello
// - 10:hello12345 -> hello12345
func decodeBencode(bencodedString string) (interface{}, error) {
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
		return nil, fmt.Errorf("unknown bencoded value: %c", ch)
	}
}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Fprintln(os.Stderr, "Starting application...")

	command := os.Args[1]

	// l5:helloi52ee
	// [“hello”,52]

	if command == "decode" {
		bencodedValue := os.Args[2]

		decoded, err := decodeBencode(bencodedValue)
		if err != nil {
			fmt.Println(err)
			return
		}

		jsonOutput, _ := json.Marshal(decoded)
		fmt.Println(string(jsonOutput))
	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
