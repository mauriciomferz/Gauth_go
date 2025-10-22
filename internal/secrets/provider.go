package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const filesystemBackend = "filesystem"

// SecretProvider defines minimal storage contract for secret governance.
type SecretProvider interface {
	Store(key string, value []byte) error
	Get(key string) ([]byte, error)
	Delete(key string) error
	Rotate(newMasterKey []byte) error
	Backend() string
}

// FilesystemProvider stores encrypted secrets as individual files on disk using AES-256-GCM.
// File format: hex(nonce):hex(ciphertext)
type FilesystemProvider struct {
	root      string
	masterKey []byte // 32 bytes
}

// NewFilesystemProvider initializes a provider at path with masterKey (32 bytes). If masterKey nil, attempts to load or generate one.
func NewFilesystemProvider(root string, masterKey []byte) (*FilesystemProvider, error) {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	keyPath := filepath.Join(root, "master.key")
	if masterKey == nil {
		if b, err := os.ReadFile(keyPath); err == nil {
			mk, err2 := hex.DecodeString(strings.TrimSpace(string(b)))
			if err2 != nil {
				return nil, err2
			}
			masterKey = mk
		} else {
			mk := make([]byte, 32)
			if _, err := rand.Read(mk); err != nil {
				return nil, err
			}
			if err := os.WriteFile(keyPath, []byte(hex.EncodeToString(mk)), 0o600); err != nil {
				return nil, err
			}
			masterKey = mk
		}
	}
	if len(masterKey) != 32 {
		return nil, errors.New("master key must be 32 bytes")
	}
	return &FilesystemProvider{root: root, masterKey: masterKey}, nil
}

func (p *FilesystemProvider) Backend() string { return filesystemBackend }

// sanitize converts key into safe filename segment.
func sanitize(key string) string {
	key = strings.TrimSpace(key)
	key = strings.ReplaceAll(key, string(os.PathSeparator), "_")
	return key
}

func (p *FilesystemProvider) pathFor(key string) string {
	return filepath.Join(p.root, sanitize(key)+".secret")
}

func (p *FilesystemProvider) Store(key string, value []byte) error {
	if len(value) == 0 {
		return errors.New("empty secret value")
	}
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	block, err := aes.NewCipher(p.masterKey)
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	ciphertext := gcm.Seal(nil, nonce, value, nil)
	enc := hex.EncodeToString(nonce) + ":" + hex.EncodeToString(ciphertext)
	return os.WriteFile(p.pathFor(key), []byte(enc), 0o600)
}

func (p *FilesystemProvider) Get(key string) ([]byte, error) {
	raw, err := os.ReadFile(p.pathFor(key))
	if err != nil {
		return nil, err
	}
	parts := strings.SplitN(string(raw), ":", 2)
	if len(parts) != 2 {
		return nil, errors.New("corrupt secret file")
	}
	nonce, err := hex.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	ciphertext, err := hex.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(p.masterKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

func (p *FilesystemProvider) Delete(key string) error {
	if err := os.Remove(p.pathFor(key)); err != nil {
		return err
	}
	return nil
}

// Rotate replaces master key and re-encrypts existing secrets atomically (best effort).
func (p *FilesystemProvider) Rotate(newMasterKey []byte) error {
	if len(newMasterKey) != 32 {
		return errors.New("new master key must be 32 bytes")
	}
	entries, err := os.ReadDir(p.root)
	if err != nil {
		return err
	}
	oldKey := p.masterKey
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".secret") {
			continue
		}
		path := filepath.Join(p.root, e.Name())
		// read & decrypt with old key
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		parts := strings.SplitN(string(raw), ":", 2)
		if len(parts) != 2 {
			return errors.New("corrupt secret file during rotation")
		}
		nonceOld, err := hex.DecodeString(parts[0])
		if err != nil {
			return err
		}
		ciphertextOld, err := hex.DecodeString(parts[1])
		if err != nil {
			return err
		}
		blockOld, err := aes.NewCipher(oldKey)
		if err != nil {
			return err
		}
		gcmOld, err := cipher.NewGCM(blockOld)
		if err != nil {
			return err
		}
		plaintext, err := gcmOld.Open(nil, nonceOld, ciphertextOld, nil)
		if err != nil {
			return err
		}
		// encrypt with new key
		nonceNew := make([]byte, 12)
		if _, err := io.ReadFull(rand.Reader, nonceNew); err != nil {
			return err
		}
		blockNew, err := aes.NewCipher(newMasterKey)
		if err != nil {
			return err
		}
		gcmNew, err := cipher.NewGCM(blockNew)
		if err != nil {
			return err
		}
		ciphertextNew := gcmNew.Seal(nil, nonceNew, plaintext, nil)
		enc := hex.EncodeToString(nonceNew) + ":" + hex.EncodeToString(ciphertextNew)
		if err := os.WriteFile(path, []byte(enc), 0o600); err != nil {
			return err
		}
	}
	// persist new master key
	if err := os.WriteFile(filepath.Join(p.root, "master.key"), []byte(hex.EncodeToString(newMasterKey)), 0o600); err != nil {
		return err
	}
	p.masterKey = newMasterKey
	return nil
}
