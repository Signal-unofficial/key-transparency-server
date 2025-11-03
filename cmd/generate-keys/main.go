//
// Copyright 2025 Signal Messenger, LLC
// SPDX-License-Identifier: AGPL-3.0-only
//

// Command generate-keys outputs fresh cryptographic keys for a Key Transparency Server.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"log"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Generating signing private key
	signingPriv := make([]byte, ed25519.SeedSize)
	if _, err := rand.Read(signingPriv); err != nil {
		log.Fatal(err)
	}

	// Generating VRF private Key
	vrfPriv := make([]byte, ed25519.SeedSize)
	if _, err := rand.Read(vrfPriv); err != nil {
		log.Fatal(err)
	}

	// Generating prefix AES Key
	prefixAesKey := make([]byte, 32)
	if _, err := rand.Read(prefixAesKey); err != nil {
		log.Fatal(err)
	}

	// Generating opening Key
	openingKey := make([]byte, 32)
	if _, err := rand.Read(openingKey); err != nil {
		log.Fatal(err)
	}

	// Encoding signing public key for Java
	signingPubTmp, err := x509.MarshalPKIXPublicKey(ed25519.NewKeyFromSeed(signingPriv).Public())
	if err != nil {
		log.Fatal(err)
	}
	signingPub := string(base64.StdEncoding.EncodeToString(signingPubTmp))

	// Encoding vrf public key for Java
	vrfPubTmp, err := x509.MarshalPKIXPublicKey(ed25519.NewKeyFromSeed(vrfPriv).Public())
	if err != nil {
		log.Fatal(err)
	}
	vrfPub := string(base64.StdEncoding.EncodeToString(vrfPubTmp))

	fmt.Printf("# kt-server\n")
	fmt.Printf("signing-key: %x\n", signingPriv)
	fmt.Printf("vrf-key: %x\n", vrfPriv)
	fmt.Printf("prefix-key: %x\n", prefixAesKey)
	fmt.Printf("opening-key: %x\n", openingKey)
	fmt.Printf("\n# kt-auditor\n")
	fmt.Printf("key-transparency-service-signing-public-key: %v\n", signingPub)
	fmt.Printf("key-transparency-service-vrf-public-key: %v\n", vrfPub)
}
