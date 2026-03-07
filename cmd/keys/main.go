package main

import (
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
)

func randBytes(n int) []byte {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return b
}

func genAccessSecret(n int) string {
	return base64.RawURLEncoding.EncodeToString(randBytes(n))
}

func genRefreshSecret(n int) string {
	return base64.RawURLEncoding.EncodeToString(randBytes(n))
}

func genEncryptionKey(n int) string {
	return base64.StdEncoding.EncodeToString(randBytes(n))
}

func main() {
	accessLen := flag.Int("access-bytes", 32, "")
	refreshLen := flag.Int("refresh-bytes", 64, "")
	keyLen := flag.Int("key-bytes", 32, "")
	yaml := flag.Bool("yaml", true, "")
	flag.Parse()

	access := genAccessSecret(*accessLen)
	refresh := genRefreshSecret(*refreshLen)
	key := genEncryptionKey(*keyLen)

	if *yaml {
		fmt.Printf("jwt:\n  access_secret: %s\n  refresh_secret: %s\n", access, refresh)
		fmt.Printf("twofa:\n  encryption_key: %s\n", key)
		return
	}

	fmt.Printf("access_secret: %s\n", access)
	fmt.Printf("refresh_secret: %s\n", refresh)
	fmt.Printf("encryption_key: %s\n", key)
}
