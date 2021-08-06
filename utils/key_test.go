package utils

import (
	"crypto/rand"
	"encoding/hex"
	"testing"
)

func TestAddress(t *testing.T) {
	key, err := GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	addr := ToAddress(key.PubKey[1:])
	t.Log(hex.EncodeToString(addr[:]))
	t.Fatal(addr.String())
}
