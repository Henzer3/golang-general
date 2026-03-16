package cardcrypter

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"unsafe"
)

type CardNumber [sizeOfNumber]byte

const sizeOfNumber = 16

type Card struct {
	ID     string
	Number CardNumber
}

type Crypter interface {
	Encrypt(cards []Card, key []byte) ([]string, error)
}

type crypterImpl struct {
	workers int
}

func New(opts ...CrypterOption) *crypterImpl {
	c := &crypterImpl{
		workers: runtime.GOMAXPROCS(0),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type CrypterOption func(*crypterImpl)

func WithWorkers(workers int) CrypterOption {
	return func(s *crypterImpl) {
		s.workers = workers
	}
}

func (c *crypterImpl) Encrypt(cards []Card, key []byte) ([]string, error) {
	if c.workers <= 0 {
		return nil, errors.New("negative workers")
	}

	if len(cards) == 0 {
		return []string{}, nil
	}

	workers := c.workers

	if workers > len(cards) {
		workers = len(cards)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes.NewCipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("cipher.NewGCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	encryptedSize := nonceSize + sizeOfNumber + gcm.Overhead()

	wg := new(sync.WaitGroup)

	out := make([]string, len(cards))
	for i := 0; i < workers; i++ {
		start := (i * len(cards)) / workers
		end := ((i + 1) * len(cards)) / workers

		wg.Go(func() {
			for j := start; j < end; j++ {
				buf := make([]byte, encryptedSize+hex.EncodedLen(encryptedSize))
				if _, err := rand.Read(buf[:nonceSize]); err != nil {
					panic(err)
				}
				result := gcm.Seal(
					buf[:nonceSize],
					buf[:nonceSize],
					cards[j].Number[:],
					unsafe.Slice(unsafe.StringData(cards[j].ID), len(cards[j].ID)),
				)
				hexBuf := buf[encryptedSize:]
				hex.Encode(hexBuf, result)
				out[j] = unsafe.String(unsafe.SliceData(hexBuf), len(hexBuf))
			}
		})
	}
	wg.Wait()
	return out, nil
}
