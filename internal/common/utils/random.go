package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

func GenerateCode(length int) string {
	const digits = "0123456789"
	code := make([]byte, length)

	for i := range code {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		code[i] = digits[n.Int64()]
	}

	return string(code)
}

func GenerateRoomName() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("room_%x", b)
}
