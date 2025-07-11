//
// Copyright 2025 Signal Messenger, LLC
// SPDX-License-Identifier: AGPL-3.0-only
//

// Command generate-keys outputs fresh cryptographic keys for a Key Transparency Server.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"log"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	signingKey := make([]byte, ed25519.SeedSize)
	if _, err := rand.Read(signingKey); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Signing Key:     %x\n", signingKey)

	vrfPriv := make([]byte, ed25519.SeedSize)
	if _, err := rand.Read(vrfPriv); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("VRF Private Key: %x\n", vrfPriv)

	prefixAesKey := make([]byte, 32)
	if _, err := rand.Read(prefixAesKey); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Prefix Aes Key:     %x\n", prefixAesKey)

	openingKey := make([]byte, 32)
	if _, err := rand.Read(openingKey); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Opening Key:     %x\n", openingKey)
}
