//
// Copyright 2025 Signal Messenger, LLC
// SPDX-License-Identifier: AGPL-3.0-only
//

package db

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createTestSignature(sig, pubKey []byte) *Signature {
	return &Signature{
		Signature:        sig,
		AuditorPublicKey: pubKey,
	}
}

func TestCloneTransparencyTreeHeadSignatures(t *testing.T) {
	original := []*Signature{
		createTestSignature([]byte("sig1"), []byte("key1")),
		createTestSignature([]byte("sig2"), []byte("key2")),
	}

	cloned := cloneTransparencyTreeHeadSignatures(original)

	// Check if the signatures are correctly cloned
	for i, orig := range original {
		// Verify Signature and AuditorPublicKey have same content as original
		if !bytes.Equal(cloned[i].Signature, orig.Signature) {
			t.Fatalf("cloned Signature doesn't match original: got %x, want %x",
				cloned[i].Signature, orig.Signature)
		}

		if !bytes.Equal(cloned[i].AuditorPublicKey, orig.AuditorPublicKey) {
			t.Fatalf("cloned AuditorPublicKey doesn't match original: got %x, want %x",
				cloned[i].AuditorPublicKey, orig.AuditorPublicKey)
		}

		// Verify that the clone is a deep copy
		if &cloned[i].Signature[0] == &original[i].Signature[0] {
			t.Fatalf("Signature %d was not deep copied", i)
		}

		if &cloned[i].AuditorPublicKey[0] == &original[i].AuditorPublicKey[0] {
			t.Fatalf("AuditorPublicKey %d was not deep copied", i)
		}
	}
}

func TestCloneAuditorTreeHeadMap(t *testing.T) {
	original := map[string]*AuditorTreeHead{
		"auditor1": createTestAuditorTreeHead(1, 100, []byte("sig1")),
		"auditor2": createTestAuditorTreeHead(2, 200, []byte("sig2")),
	}

	cloned := cloneAuditorTreeHeadMap(original)

	for auditor, origAuditorTreeHead := range original {
		assert.True(t, isAuditorTreeHeadDeepCopy(*origAuditorTreeHead, *cloned[auditor]),
			fmt.Sprintf("cloned auditor head %+v is not a deep copy of %+v", cloned[auditor], origAuditorTreeHead))
	}
}

func createTestAuditorTreeHead(treeSize uint64, timestamp int64, sig []byte) *AuditorTreeHead {
	return &AuditorTreeHead{
		AuditorTransparencyTreeHead: AuditorTransparencyTreeHead{
			TreeSize:  treeSize,
			Timestamp: timestamp,
			Signature: sig,
		},
		RootValue:   []byte("root"),
		Consistency: [][]byte{[]byte("cons1"), []byte("cons2")},
	}
}

// isDeepCopy checks that the contents of a and b are equal, but that they do not point to the same underlying array
func isDeepCopy(a, b []byte) bool {
	if !bytes.Equal(a, b) {
		return false
	}

	if len(a) == 0 || len(b) == 0 {
		return true
	}

	return &a[0] != &b[0]
}

func isAuditorTreeHeadDeepCopy(a, b AuditorTreeHead) bool {
	if a.TreeSize != b.TreeSize || a.Timestamp != b.Timestamp {
		return false
	}

	if !isDeepCopy(a.Signature, b.Signature) {
		return false
	}

	if !isDeepCopy(a.RootValue, b.RootValue) {
		return false
	}

	if len(a.Consistency) != len(b.Consistency) {
		return false
	}

	for i := range a.Consistency {
		if !isDeepCopy(a.Consistency[i], b.Consistency[i]) {
			return false
		}
	}

	return true
}
