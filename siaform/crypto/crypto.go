package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

type Cipher struct {
	b cipher.Block
}

func New(key []byte) (*Cipher, error) {
	h := sha256.Sum256(key)
	b, err := aes.NewCipher(h[:])
	if err != nil {
		return nil, fmt.Errorf("aes.NewCipher: %v", err)
	}
	return &Cipher{
		b: b,
	}, nil
}

func (c *Cipher) Encrypt(sectorID int64, data []byte) {
	iv := make([]byte, c.b.BlockSize())
	binary.LittleEndian.PutUint64(iv[:8], uint64(sectorID))
	s := cipher.NewCTR(c.b, iv)
	s.XORKeyStream(data, data)
}

func (c *Cipher) Decrypt(sectorID int64, data []byte) {
	c.Encrypt(sectorID, data)
}
