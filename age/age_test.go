package age

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"
	"filippo.io/age/armor"
)

func TestEncryptDecryptRoundtrip(t *testing.T) {
	// Generate test identity and recipient
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate identity: %v", err)
	}
	recipient := identity.Recipient()

	t.Run("encrypt and decrypt plaintext without armor", func(t *testing.T) {
		plaintext := []byte("Hello, AGE!")

		ciphertext, err := EncryptToMemory(plaintext, []age.Recipient{recipient}, false)
		if err != nil {
			t.Fatalf("encryption failed: %v", err)
		}

		// Decrypt the ciphertext
		r, err := age.Decrypt(bytes.NewReader(ciphertext), identity)
		if err != nil {
			t.Fatalf("decryption failed: %v", err)
		}

		var decrypted bytes.Buffer
		if _, err := decrypted.ReadFrom(r); err != nil {
			t.Fatalf("reading decrypted data failed: %v", err)
		}

		if !bytes.Equal(plaintext, decrypted.Bytes()) {
			t.Errorf("decrypted text does not match original: got %q, want %q", decrypted.String(), string(plaintext))
		}
	})

	t.Run("encrypt and decrypt plaintext with armor", func(t *testing.T) {
		plaintext := []byte("Hello, AGE with armor!")

		ciphertext, err := EncryptToMemory(plaintext, []age.Recipient{recipient}, true)
		if err != nil {
			t.Fatalf("encryption with armor failed: %v", err)
		}

		// Verify armor header is present
		if !bytes.Contains(ciphertext, []byte("-----BEGIN AGE ENCRYPTED FILE-----")) {
			t.Error("armored output missing BEGIN header")
		}

		// Decrypt the armored ciphertext - need to use armor reader
		armorReader := armor.NewReader(bytes.NewReader(ciphertext))
		r, err := age.Decrypt(armorReader, identity)
		if err != nil {
			t.Fatalf("decryption of armored data failed: %v", err)
		}

		var decrypted bytes.Buffer
		if _, err := decrypted.ReadFrom(r); err != nil {
			t.Fatalf("reading decrypted armored data failed: %v", err)
		}

		if !bytes.Equal(plaintext, decrypted.Bytes()) {
			t.Errorf("decrypted armored text does not match original: got %q, want %q", decrypted.String(), string(plaintext))
		}
	})

	t.Run("encrypt with multiple recipients", func(t *testing.T) {
		identity2, err := age.GenerateX25519Identity()
		if err != nil {
			t.Fatalf("failed to generate second identity: %v", err)
		}
		recipient2 := identity2.Recipient()

		plaintext := []byte("Multi-recipient message")

		ciphertext, err := EncryptToMemory(plaintext, []age.Recipient{recipient, recipient2}, false)
		if err != nil {
			t.Fatalf("encryption with multiple recipients failed: %v", err)
		}

		// Decrypt with first identity
		r1, err := age.Decrypt(bytes.NewReader(ciphertext), identity)
		if err != nil {
			t.Fatalf("decryption with first identity failed: %v", err)
		}
		var decrypted1 bytes.Buffer
		if _, err := decrypted1.ReadFrom(r1); err != nil {
			t.Fatalf("reading decrypted data with first identity failed: %v", err)
		}

		// Decrypt with second identity
		r2, err := age.Decrypt(bytes.NewReader(ciphertext), identity2)
		if err != nil {
			t.Fatalf("decryption with second identity failed: %v", err)
		}
		var decrypted2 bytes.Buffer
		if _, err := decrypted2.ReadFrom(r2); err != nil {
			t.Fatalf("reading decrypted data with second identity failed: %v", err)
		}

		if !bytes.Equal(plaintext, decrypted1.Bytes()) {
			t.Errorf("first identity decryption failed: got %q, want %q", decrypted1.String(), string(plaintext))
		}
		if !bytes.Equal(plaintext, decrypted2.Bytes()) {
			t.Errorf("second identity decryption failed: got %q, want %q", decrypted2.String(), string(plaintext))
		}
	})
}

func TestAtomicEncryptWrite(t *testing.T) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate identity: %v", err)
	}
	recipient := identity.Recipient()

	t.Run("writes encrypted file atomically", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.age")
		plaintext := []byte("Atomic write test")

		err := AtomicEncryptWrite(filePath, plaintext, []age.Recipient{recipient}, false)
		if err != nil {
			t.Fatalf("atomic encrypt write failed: %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Fatal("encrypted file was not created")
		}

		// Decrypt and verify content
		decrypted, err := DecryptToMemory(filePath, []age.Identity{identity})
		if err != nil {
			t.Fatalf("failed to decrypt written file: %v", err)
		}

		if decrypted != string(plaintext) {
			t.Errorf("decrypted content does not match: got %q, want %q", decrypted, string(plaintext))
		}
	})

	t.Run("writes armored encrypted file atomically", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test-armored.age")
		plaintext := []byte("Atomic armored write test")

		err := AtomicEncryptWrite(filePath, plaintext, []age.Recipient{recipient}, true)
		if err != nil {
			t.Fatalf("atomic encrypt write with armor failed: %v", err)
		}

		// Read file and verify armor headers
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("failed to read written file: %v", err)
		}

		if !bytes.Contains(content, []byte("-----BEGIN AGE ENCRYPTED FILE-----")) {
			t.Error("written file missing armor BEGIN header")
		}

		// Decrypt and verify content
		decrypted, err := DecryptToMemory(filePath, []age.Identity{identity})
		if err != nil {
			t.Fatalf("failed to decrypt written armored file: %v", err)
		}

		if decrypted != string(plaintext) {
			t.Errorf("decrypted armored content does not match: got %q, want %q", decrypted, string(plaintext))
		}
	})
}

func TestLoadIdentities(t *testing.T) {
	t.Run("loads valid identity file", func(t *testing.T) {
		tmpDir := t.TempDir()
		keyPath := filepath.Join(tmpDir, "key.txt")

		// Generate and save identity
		identity, err := age.GenerateX25519Identity()
		if err != nil {
			t.Fatalf("failed to generate identity: %v", err)
		}

		err = os.WriteFile(keyPath, []byte(identity.String()+"\n"), 0600)
		if err != nil {
			t.Fatalf("failed to write identity file: %v", err)
		}

		// Load identities
		identities, err := LoadIdentities(keyPath)
		if err != nil {
			t.Fatalf("failed to load identities: %v", err)
		}

		if len(identities) != 1 {
			t.Errorf("expected 1 identity, got %d", len(identities))
		}
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		_, err := LoadIdentities("/nonexistent/path/key.txt")
		if err == nil {
			t.Error("expected error for missing identity file")
		}
	})
}

func TestLoadRecipients(t *testing.T) {
	t.Run("loads valid recipients file", func(t *testing.T) {
		tmpDir := t.TempDir()
		recipientsPath := filepath.Join(tmpDir, "recipients.txt")

		// Generate recipient
		identity, err := age.GenerateX25519Identity()
		if err != nil {
			t.Fatalf("failed to generate identity: %v", err)
		}

		err = os.WriteFile(recipientsPath, []byte(identity.Recipient().String()+"\n"), 0644)
		if err != nil {
			t.Fatalf("failed to write recipients file: %v", err)
		}

		// Load recipients
		recipients, err := LoadRecipients(recipientsPath)
		if err != nil {
			t.Fatalf("failed to load recipients: %v", err)
		}

		if len(recipients) != 1 {
			t.Errorf("expected 1 recipient, got %d", len(recipients))
		}
	})

	t.Run("returns error for empty recipients file", func(t *testing.T) {
		tmpDir := t.TempDir()
		recipientsPath := filepath.Join(tmpDir, "empty.txt")

		err := os.WriteFile(recipientsPath, []byte(""), 0644)
		if err != nil {
			t.Fatalf("failed to write empty recipients file: %v", err)
		}

		_, err = LoadRecipients(recipientsPath)
		if err == nil {
			t.Error("expected error for empty recipients file")
		}
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		_, err := LoadRecipients("/nonexistent/path/recipients.txt")
		if err == nil {
			t.Error("expected error for missing recipients file")
		}
	})
}
