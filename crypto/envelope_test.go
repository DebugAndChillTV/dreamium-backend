package crypto

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"
)

// newKey fills a 32-byte key with the given byte value.
func newKey(b byte) []byte {
	key := make([]byte, 32)
	for i := range key {
		key[i] = b
	}
	return key
}

func TestRoundTrip(t *testing.T) {
	key := newKey(0x01)
	plaintext := []byte("I was flying over a dark forest and felt afraid")

	ct, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	got, err := Decrypt(ct, key)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Errorf("roundtrip mismatch: got %q want %q", got, plaintext)
	}
}

func TestEmptyPlaintext(t *testing.T) {
	key := newKey(0x02)
	ct, err := Encrypt([]byte{}, key)
	if err != nil {
		t.Fatalf("Encrypt empty: %v", err)
	}
	got, err := Decrypt(ct, key)
	if err != nil {
		t.Fatalf("Decrypt empty: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty plaintext, got %q", got)
	}
}

func TestUnicodePlaintext(t *testing.T) {
	key := newKey(0x03)
	plaintext := []byte("Düşümde uçuyordum 🌙 karanlık bir ormanda")
	ct, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt unicode: %v", err)
	}
	got, err := Decrypt(ct, key)
	if err != nil {
		t.Fatalf("Decrypt unicode: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Error("unicode roundtrip mismatch")
	}
}

func TestLargePlaintext(t *testing.T) {
	key := newKey(0x04)
	plaintext := []byte(strings.Repeat("a long dream description ", 200)) // ~5000 bytes
	ct, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt large: %v", err)
	}
	got, err := Decrypt(ct, key)
	if err != nil {
		t.Fatalf("Decrypt large: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Error("large plaintext roundtrip mismatch")
	}
}

func TestNonceRandomness(t *testing.T) {
	key := newKey(0x05)
	plaintext := []byte("same plaintext every time")
	ct1, _ := Encrypt(plaintext, key)
	ct2, _ := Encrypt(plaintext, key)
	if ct1 == ct2 {
		t.Error("two encryptions of the same plaintext must produce different ciphertexts")
	}
}

func TestWrongKeyDecryptFails(t *testing.T) {
	key1 := newKey(0x06)
	key2 := newKey(0x07)

	ct, err := Encrypt([]byte("secret dream"), key1)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	_, err = Decrypt(ct, key2)
	if err == nil {
		t.Error("expected decryption to fail with wrong key")
	}
}

func TestShortKeyEncryptReturnsError(t *testing.T) {
	_, err := Encrypt([]byte("dream"), []byte("tooshort"))
	if err == nil {
		t.Error("expected error for key < 32 bytes on Encrypt")
	}
}

func TestLongKeyEncryptReturnsError(t *testing.T) {
	_, err := Encrypt([]byte("dream"), make([]byte, 33))
	if err == nil {
		t.Error("expected error for key > 32 bytes on Encrypt")
	}
}

func TestShortKeyDecryptReturnsError(t *testing.T) {
	key := newKey(0x08)
	ct, _ := Encrypt([]byte("dream"), key)
	_, err := Decrypt(ct, []byte("tooshort"))
	if err == nil {
		t.Error("expected error for key < 32 bytes on Decrypt")
	}
}

func TestTamperedCiphertext(t *testing.T) {
	key := newKey(0x09)
	ct, _ := Encrypt([]byte("very secret dream"), key)

	// Corrupt the last character of the base64 string.
	corrupted := []byte(ct)
	corrupted[len(corrupted)-3] ^= 0xff
	_, err := Decrypt(string(corrupted), key)
	if err == nil {
		t.Error("expected decryption to fail on tampered ciphertext")
	}
}

func TestInvalidBase64(t *testing.T) {
	key := newKey(0x0a)
	_, err := Decrypt("not!!valid%%base64", key)
	if err == nil {
		t.Error("expected error on invalid base64 input")
	}
}

func TestCiphertextTooShort(t *testing.T) {
	key := newKey(0x0b)
	// base64("hello") is only 5 bytes decoded — shorter than nonce(12)+tag(16)
	_, err := Decrypt("aGVsbG8=", key)
	if err == nil {
		t.Error("expected error for ciphertext shorter than nonce+tag minimum")
	}
}

func TestOutputIsValidBase64(t *testing.T) {
	key := newKey(0x0c)
	ct, err := Encrypt([]byte("dream text"), key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if strings.ContainsAny(ct, " \n\r\t") {
		t.Error("ciphertext must not contain whitespace")
	}
}

// TestKeyRotation simulates a full key rotation:
// encrypt with key1 → decrypt with key1 → re-encrypt with key2 → verify key2 only.
func TestKeyRotation(t *testing.T) {
	key1, _ := hex.DecodeString("0101010101010101010101010101010101010101010101010101010101010101")
	key2, _ := hex.DecodeString("0202020202020202020202020202020202020202020202020202020202020202")

	dreams := []string{
		"I was flying over a mountain range",
		"A dark forest surrounded me with strange sounds",
		"I met an old friend I had completely forgotten",
	}

	// Phase 1 — encrypt all rows with key1 (simulates current state in DB).
	encrypted := make([]string, len(dreams))
	for i, d := range dreams {
		ct, err := Encrypt([]byte(d), key1)
		if err != nil {
			t.Fatalf("phase1 Encrypt[%d]: %v", i, err)
		}
		encrypted[i] = ct
	}

	// Phase 2 — decrypt with key1, re-encrypt with key2 (the migration step).
	rotated := make([]string, len(dreams))
	for i, ct := range encrypted {
		pt, err := Decrypt(ct, key1)
		if err != nil {
			t.Fatalf("phase2 Decrypt[%d] with key1: %v", i, err)
		}
		ct2, err := Encrypt(pt, key2)
		if err != nil {
			t.Fatalf("phase2 re-Encrypt[%d] with key2: %v", i, err)
		}
		rotated[i] = ct2
	}

	// Phase 3 — verify new ciphertexts: readable with key2, rejected with key1.
	for i, ct2 := range rotated {
		pt, err := Decrypt(ct2, key2)
		if err != nil {
			t.Fatalf("phase3 Decrypt[%d] with key2: %v", i, err)
		}
		if string(pt) != dreams[i] {
			t.Errorf("dream[%d] mismatch after rotation: got %q want %q", i, pt, dreams[i])
		}

		_, err = Decrypt(ct2, key1)
		if err == nil {
			t.Errorf("dream[%d] must not decrypt with old key after rotation", i)
		}
	}
}

// TestKeyRotationOldCiphertextsAreInert confirms original key1 ciphertexts are
// no longer used once rotation is complete (old ciphertexts can't decrypt with key2).
func TestKeyRotationOldCiphertextsAreInert(t *testing.T) {
	key1, _ := hex.DecodeString("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"[:64])
	key2, _ := hex.DecodeString("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	ct, _ := Encrypt([]byte("old dream"), key1)

	_, err := Decrypt(ct, key2)
	if err == nil {
		t.Error("old ciphertext (key1) should not be decryptable with new key2")
	}
}
