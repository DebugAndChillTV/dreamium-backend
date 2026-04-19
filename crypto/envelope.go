package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

const keySize = 32 // AES-256

// Encrypt encrypts plaintext using AES-256-GCM with a fresh random nonce.
// masterKey must be exactly 32 bytes.
// Output: base64StdEncoding( nonce[12] || ciphertext[N] || tag[16] )
func Encrypt(plaintext, masterKey []byte) (string, error) {
	if len(masterKey) != keySize {
		return "", errors.New("encrypt: master key must be 32 bytes")
	}

	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return "", fmt.Errorf("encrypt: cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("encrypt: gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize()) // 12 bytes
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("encrypt: nonce: %w", err)
	}

	// Seal(dst=nonce, ...) → nonce || ciphertext || tag
	sealed := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt decrypts a blob produced by Encrypt.
// masterKey must be exactly 32 bytes.
// Returns an error on wrong key, tampered data, or malformed input — never panics.
func Decrypt(ciphertext string, masterKey []byte) ([]byte, error) {
	if len(masterKey) != keySize {
		return nil, errors.New("decrypt: master key must be 32 bytes")
	}

	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decrypt: base64: %w", err)
	}

	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt: cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("decrypt: gcm: %w", err)
	}

	nonceSize := gcm.NonceSize() // 12
	if len(raw) < nonceSize+gcm.Overhead() {
		return nil, errors.New("decrypt: ciphertext too short")
	}

	nonce, sealed := raw[:nonceSize], raw[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: authentication failed (wrong key or tampered data): %w", err)
	}

	return plaintext, nil
}
