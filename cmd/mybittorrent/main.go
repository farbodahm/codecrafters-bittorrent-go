package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"unicode"
)

// Ensures gofmt doesn't remove the "os" encoding/json import (feel free to remove this!)
var _ = json.Marshal

func decodeBencodeString(s string) (string, error) {
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
		return "", err
	}

	return s[firstColonIndex+1 : firstColonIndex+1+length], nil
}

func decodeBencodeInt(s string) (int, error) {
	s = s[1 : len(s)-1]
	return strconv.Atoi(s)
}

// Example:
// - 5:hello -> hello
// - 10:hello12345 -> hello12345
func decodeBencode(bencodedString string) (interface{}, error) {
	ch := bencodedString[0]
	switch ch {
	case 'i':
		return decodeBencodeInt(bencodedString)
	case 'l':
		return nil, fmt.Errorf("lists are not supported at the moment")
	case 'd':
		return nil, fmt.Errorf("dictionaries are not supported at the moment")
	default:
		if unicode.IsDigit(rune(ch)) {
			return decodeBencodeString(bencodedString)
		}
		return nil, fmt.Errorf("unknown bencoded value: %c", ch)
	}
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

		jsonOutput, _ := json.Marshal(decoded)
		fmt.Println(string(jsonOutput))
	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
