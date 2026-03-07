package admin

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
)

type Encryptor interface {
	Encrypt([]byte) (string, error)
	Decrypt(string) ([]byte, error)
}

type aesGCMEncryptor struct {
	gcm cipher.AEAD
}

func NewEncryptorFromSecret(secret string) Encryptor {
	key := sha256.Sum256([]byte(secret))
	block, _ := aes.NewCipher(key[:])
	gcm, _ := cipher.NewGCM(block)
	return &aesGCMEncryptor{gcm: gcm}
}

func (e *aesGCMEncryptor) Encrypt(plain []byte) (string, error) {
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := e.gcm.Seal(nil, nonce, plain, nil)
	buf := append(nonce, ct...)
	return base64.StdEncoding.EncodeToString(buf), nil
}

func (e *aesGCMEncryptor) Decrypt(s string) ([]byte, error) {
	buf, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	ns := e.gcm.NonceSize()
	if len(buf) < ns {
		return nil, io.ErrUnexpectedEOF
	}
	nonce := buf[:ns]
	ct := buf[ns:]
	pt, err := e.gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, err
	}
	return pt, nil
}
