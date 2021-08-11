package utils

import (
	"crypto/rand"
	"testing"

	"github.com/minio/blake2b-simd"
)

func TestAddress(t *testing.T) {
	key, err := GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	addr := ToAddress(key.PubKey[1:])
	t.Log(addr.String())

	var a Address
	var b Address

	if a != b {
		t.Fatal("not equal")
	}

	if a == addr {
		t.Fatal("equal")
	}

	t.Fatal("end")
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

	ok := Verify(addr, msg[:], sig)
	if !ok {
		t.Fatal("verify fail")
	}

	t.Fatal("end")
}
