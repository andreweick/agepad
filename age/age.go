package age

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"filippo.io/age"
	"filippo.io/age/armor"
)

// LoadIdentities loads AGE identities from the specified file path.
func LoadIdentities(path string) ([]age.Identity, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("\nCould not read AGE key at %s\n"+
			"- If you don't have one:   age-keygen --output %s\n"+
			"- Or point to another key: --identities /path/to/key.txt\nOriginal error: %w",
			path, path, err)
	}
	ids, err := age.ParseIdentities(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("failed to parse identities in %s: %w", path, err)
	}
	return ids, nil
}

// LoadRecipients loads AGE recipients from the specified file path.
func LoadRecipients(path string) ([]age.Recipient, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("\nRecipients file not found: %s\n"+
			"- Create one and commit it to your repo (recommended).\n"+
			"- Example (one public key per line): age1xxxx...\nOriginal error: %w", path, err)
	}
	rs, err := age.ParseRecipients(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("failed to parse recipients in %s: %w", path, err)
	}
	if len(rs) == 0 {
		return nil, fmt.Errorf("no recipients in %s; add at least one age public key", path)
	}
	return rs, nil
}

// DecryptToMemory decrypts an AGE-encrypted file to memory.
func DecryptToMemory(cipherPath string, ids []age.Identity) (string, error) {
	f, err := os.Open(cipherPath)
	if err != nil {
		return "", fmt.Errorf("open ciphertext: %w", err)
	}
	defer f.Close()

	// Try to unwrap armor if present
	reader := io.Reader(f)
	armoredReader := armor.NewReader(f)
	if _, err := armoredReader.Read(make([]byte, 1)); err == nil {
		// Reset file pointer and use armored reader
		f.Seek(0, 0)
		reader = armor.NewReader(f)
	} else {
		// Not armored, reset and use plain reader
		f.Seek(0, 0)
	}

	r, err := age.Decrypt(reader, ids...)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	plain, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read plaintext: %w", err)
	}
	return string(plain), nil
}

// EncryptToMemory encrypts plaintext to memory using AGE.
func EncryptToMemory(plaintext []byte, recips []age.Recipient, useArmor bool) ([]byte, error) {
	var buf bytes.Buffer
	if useArmor {
		aw := armor.NewWriter(&buf)
		w, err := age.Encrypt(aw, recips...)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(plaintext); err != nil {
			return nil, err
		}
		if err := w.Close(); err != nil { // closes the encryption stream
			return nil, err
		}
		if err := aw.Close(); err != nil { // closes the armor wrapper
			return nil, err
		}
		return buf.Bytes(), nil
	}
	w, err := age.Encrypt(&buf, recips...)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(plaintext); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// AtomicEncryptWrite encrypts and writes data to a file atomically.
func AtomicEncryptWrite(dstPath string, b []byte, recips []age.Recipient, useArmor bool) error {
	dir := filepath.Dir(dstPath)
	tmp, err := os.CreateTemp(dir, ".agepad-tmp-*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if useArmor {
		aw := armor.NewWriter(tmp)
		w, err := age.Encrypt(aw, recips...)
		if err != nil {
			return fmt.Errorf("encrypt: %w", err)
		}
		if _, err := w.Write(b); err != nil {
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
		if err := aw.Close(); err != nil {
			return err
		}
	} else {
		w, err := age.Encrypt(tmp, recips...)
		if err != nil {
			return fmt.Errorf("encrypt: %w", err)
		}
		if _, err := w.Write(b); err != nil {
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}
	return os.Rename(tmpPath, dstPath) // atomic replace on same filesystem
}
