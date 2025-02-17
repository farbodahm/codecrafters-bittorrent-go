package app

import (
	"crypto/rand"
	"log"
	"math/big"
)

// GenerateRandomID generates a random alphanumeric ID of the given length
func GenerateRandomID(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		index, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			log.Fatalf("failed to generate random ID: %v", err)
		}
		result[i] = charset[index.Int64()]
	}
	return string(result)
}
