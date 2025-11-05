package slug

import (
	"crypto/rand"
	"encoding/binary"
	"time"
)

type ShortIDGenerator struct{}

func NewShortID() *ShortIDGenerator {
	return &ShortIDGenerator{}
}

func (g *ShortIDGenerator) Next() (string, error) {
	now := time.Now().UnixMilli()
	entropy := make([]byte, 4)
	if _, err := rand.Read(entropy); err != nil {
		return "", err
	}
	entropyUint32 := binary.BigEndian.Uint32(entropy)
	id := uint64(now)<<32 | uint64(entropyUint32)
	enc := encodeBase62(id)
	if len(enc) < 8 {
		for len(enc) < 8 {
			enc = string(base62Alphabet[0]) + enc
		}
	}
	return enc, nil
}
