package crypto

import "sync"

var (
	encryptionKey   []byte
	encryptionKeyMu sync.RWMutex
)

// SetEncryptionKey stores the 32-byte key used for encrypting secrets at rest.
func SetEncryptionKey(key []byte) {
	encryptionKeyMu.Lock()
	encryptionKey = make([]byte, len(key))
	copy(encryptionKey, key)
	encryptionKeyMu.Unlock()
}

// GetEncryptionKey returns the current encryption key.
func GetEncryptionKey() []byte {
	encryptionKeyMu.RLock()
	defer encryptionKeyMu.RUnlock()
	return encryptionKey
}
