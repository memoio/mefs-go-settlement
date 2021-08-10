package utils

import (
	"crypto/rand"
	"encoding/hex"
	"github.com/minio/blake2b-simd"
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

func TestSign(t *testing.T) {
	key, err := GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	addr := ToAddress(key.PubKey[1:])
	t.Log(addr.String())

	msg := blake2b.Sum256([]byte("test"))
	sig, err := Sign(key.SecretKey, msg[:])
	if err != nil {
		t.Fatal(err)
	}

	ok := Verify(key.PubKey, msg[:], sig)
	if !ok {
		t.Fatal("verify fail")
	}

	t.Fatal(addr.String())
}
