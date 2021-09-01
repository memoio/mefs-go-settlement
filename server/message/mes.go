package message

import (
	"encoding/binary"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/memoio/go-settlement/utils"
)

type Message struct {
	Version uint32

	To   uint64
	From uint64 // fsID/userID/providerID/keeperID

	Nonce uint64

	Value *big.Int

	GasLimit uint64
	GasPrice *big.Int

	Method uint32
	Params []byte
}

type SignedMessage struct {
	Message
	Signature []byte // signed by Tx.From; bls sig
}

type MethodType uint32

const (
	Register MethodType = iota
	RegisterToken
	RegisterKeeper
	RegisterProvider
	RegisterUser
	CreateGroup
	SetReady
	AddKeeperToGroup
	AddProviderToGroup
	Pledge
	Withdraw
	Recharge
	WithdrawFromFs
	ProWithdraw
	AddOrder
	SubOrder
)

type ParasRegister struct {
	Addr utils.Address
	Sig  []byte
}

type ParasCommon struct {
	Index      uint64
	GIndex     uint64
	TokenIndex uint32
	Amount     *big.Int
	Extra      []byte
}

func (p *ParasCommon) Serialize() []byte {
	bufLen := 0
	if p.Index > 0 {
		bufLen += 8
	}

	if p.GIndex > 0 {
		bufLen += 8
	}

	if p.TokenIndex > 0 {
		bufLen += 4
	}

	if p.Amount.Sign() != 0 {
		bufLen += 32
	}
	amount := common.LeftPadBytes(p.Amount.Bytes(), 32)

	eLen := len(p.Extra)
	bufLen += eLen

	buf := make([]byte, bufLen)
	offset := 0
	if p.Index > 0 {
		binary.BigEndian.PutUint64(buf[offset:offset+8], p.Index)
		offset += 8
	}

	if p.GIndex > 0 {
		binary.BigEndian.PutUint64(buf[offset:offset+8], p.GIndex)
		offset += 8
	}

	if p.TokenIndex > 0 {
		binary.BigEndian.PutUint32(buf[offset:offset+4], p.TokenIndex)
		offset += 4
	}

	copy(buf[offset:offset+32], amount)
	offset += 32
	copy(buf[offset:offset+eLen], p.Extra)
	return buf
}

type SignedParasCommon struct {
	ParasCommon
	Auth [][]byte // signed by index; meta-transcation; or auth by keepers/admin
}

func (a *SignedParasCommon) Serialize() []byte {
	authLen := len(a.Auth)
	sigLen := 0
	if authLen > 0 {
		sigLen = len(a.Auth[0])
	}
	pbuf := a.ParasCommon.Serialize()
	pLen := len(pbuf)

	buf := make([]byte, pLen+sigLen*authLen)
	offset := 0
	copy(buf[:pLen], pbuf)
	offset += pLen

	for i := 0; i < authLen; i++ {
		copy(buf[offset:offset+sigLen], a.Auth[i])
		offset += sigLen
	}

	return buf
}

type ParasProWithdraw struct {
	Index      uint64
	TokenIndex uint32
	Amount     *big.Int
	Lost       *big.Int
}

func (w *ParasProWithdraw) Serialize() []byte {
	bufLen := 0
	if w.Index > 0 {
		bufLen += 8
	}

	if w.TokenIndex > 0 {
		bufLen += 4
	}

	if w.Amount.Sign() != 0 {
		bufLen += 32
	}

	amount := common.LeftPadBytes(w.Amount.Bytes(), 32)

	if w.Lost.Sign() != 0 {
		bufLen += 32
	}

	lost := common.LeftPadBytes(w.Lost.Bytes(), 32)

	buf := make([]byte, bufLen)
	offset := 0
	if w.Index > 0 {
		binary.BigEndian.PutUint64(buf[offset:offset+8], w.Index)
		offset += 8
	}

	if w.TokenIndex > 0 {
		binary.BigEndian.PutUint32(buf[offset:offset+4], w.TokenIndex)
		offset += 4
	}
	if w.Amount.Sign() != 0 {
		copy(buf[offset:offset+32], amount)
		offset += 32
	}

	if w.Lost.Sign() != 0 {
		copy(buf[offset:offset+32], lost)
	}
	return buf
}

type SignedParasProWithdraw struct {
	ParasProWithdraw
	Auth [][]byte
}

type ParasOrder struct {
	User       uint64
	Provider   uint64
	Start      uint64
	End        uint64
	Size       uint64
	Nonce      uint64
	TokenIndex uint32
	Price      *big.Int
	Usign      []byte
	Psign      []byte
	Auth       [][]byte
}

type SignedParasOrder struct {
	ParasOrder
	Usign []byte
	PSign []byte
	Auth  [][]byte
}
