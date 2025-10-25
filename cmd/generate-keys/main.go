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

	// Signing key
	signingKey := make([]byte, ed25519.SeedSize)
	if _, err := rand.Read(signingKey); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("signing-key: %x\n", signingKey)

	// VRF Private Key
	vrfPriv := make([]byte, ed25519.SeedSize)
	if _, err := rand.Read(vrfPriv); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("vrf-key: %x\n", vrfPriv)

	// Prefix Aes Key
	prefixAesKey := make([]byte, 32)
	if _, err := rand.Read(prefixAesKey); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("prefix-key: %x\n", prefixAesKey)

	// Opening Key
	openingKey := make([]byte, 32)
	if _, err := rand.Read(openingKey); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("opening-key: %x\n", openingKey)
}
