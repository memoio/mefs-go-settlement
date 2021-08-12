package utils

import (
	"crypto/ecdsa"
	"encoding/hex"
	"io"

	"github.com/btcsuite/btcd/btcec"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	blake2b "github.com/minio/blake2b-simd"
)

const (
	AddressLength = 20
)

type Address [AddressLength]byte

// NilAddress is a nil
var NilAddress Address

func BytesToAddress(b []byte) Address {
	var a Address
	a.SetBytes(b)
	return a
}

func (a *Address) SetBytes(b []byte) {
	if len(b) > len(a) {
		b = b[len(b)-AddressLength:]
	}

	copy(a[AddressLength-len(b):], b)
}

func (a Address) String() string {
	return a.hex()
}

func (a *Address) hex() string {
	return "0x" + hex.EncodeToString(a[:])
}

func HexToAddress(s string) Address {
	return BytesToAddress(FromHex(s))
}

func has0xPrefix(str string) bool {
	return len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X')
}

func FromHex(s string) []byte {
	if has0xPrefix(s) {
		s = s[2:]
	}
	if len(s)%2 == 1 {
		s = "0" + s
	}

	res, _ := hex.DecodeString(s)
	return res
}

type Key struct {
	SecretKey []byte
	PubKey    []byte
}

func GenerateKey(rand io.Reader) (*Key, error) {
	key, err := ecdsa.GenerateKey(btcec.S256(), rand)
	if err != nil {
		return nil, err
	}

	secretKey := (*btcec.PrivateKey)(key).Serialize()

	pk := (*btcec.PublicKey)(&key.PublicKey)

	k := &Key{
		SecretKey: secretKey,
		PubKey:    pk.SerializeUncompressed(),
	}

	return k, nil
}

func ToAddress(pk []byte) Address {
	if len(pk) == 65 {
		pk = pk[1:]
	}
	h := blake2b.Sum256(pk)
	return BytesToAddress(h[12:])
}

func GetContractAddress(addr Address, method []byte) Address {
	h := blake2b.New512()

	h.Write(addr[:])
	h.Write(method)

	b := h.Sum(nil)

	return ToAddress(b[:])
}

// Sign signs the given message, which must be 32 bytes long.
func Sign(sk, msg []byte) ([]byte, error) {
	return secp256k1.Sign(msg, sk)
}

// Verify2 checks the given signature and returns true if it is valid.
func Verify2(pk, msg, signature []byte) bool {
	if len(signature) > 64 {
		signature = signature[:64]
	}
	return secp256k1.VerifySignature(pk[:], msg, signature)
}

// Verify uses ECRecover
func Verify(addr Address, msg, signature []byte) bool {
	pk, err := secp256k1.RecoverPubkey(msg, signature)
	if err != nil {
		return false
	}

	nAddr := ToAddress(pk)
	return addr == nAddr
}

// EcRecover recovers the public key from a message, signature pair.
func EcRecover(msg, signature []byte) ([]byte, error) {
	return secp256k1.RecoverPubkey(msg, signature)
}
