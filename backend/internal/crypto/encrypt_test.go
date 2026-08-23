package crypto

import (
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	key := "01234567890123456789012345678901" // 32 bit (AES-256)
	plaintext := "postgres://user:pass@host:5432/db"

	enc, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if enc == "" {
		t.Error("Encrypt returned empty string")
	}

	dec, err := Decrypt(key, enc)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if dec != plaintext {
		t.Errorf("Decrypt() = %v, want %v", dec, plaintext)
	}

	// wrong key
	wrongKey := "00000000000000000000000000000000"
	_, err = Decrypt(wrongKey, enc)
	if err == nil {
		t.Error("Decrypt with wrong key should fail")
	}

	// empty key
	_, err = Encrypt("", plaintext)
	if err == nil {
		t.Error("Encrypt with empty key should fail")
	}

	// invalid base64
	_, err = Decrypt(key, "not_base64_data")
	if err == nil {
		t.Error("Decrypt with invalid base64 should fail")
	}
}
