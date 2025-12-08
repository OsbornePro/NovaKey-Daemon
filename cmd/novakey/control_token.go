package main

import (
	"crypto/rand"
	"math/big"
)

var controlTokenCharset = []rune(
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz" +
		"0123456789" +
		"!@#$%^&*()-_=+[]{}:,.?",
)

func generateControlToken() (string, error) {
	// Random length between 55 and 64
	lenRange := int64(10)
	base := int64(55)

	n, err := rand.Int(rand.Reader, big.NewInt(lenRange))
	if err != nil {
		return "", err
	}
	length := base + n.Int64()

	tokenRunes := make([]rune, length)
	for i := range tokenRunes {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(controlTokenCharset))))
		if err != nil {
			return "", err
		}
		tokenRunes[i] = controlTokenCharset[idx.Int64()]
	}

	return string(tokenRunes), nil
}
