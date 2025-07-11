//
// Copyright 2025 Signal Messenger, LLC
// SPDX-License-Identifier: AGPL-3.0-only
//

package db

import (
	"crypto/rand"
	"encoding/json"
	"testing"

	"github.com/kinbiko/jsonassert"
)

func TestSerializeNewStoredTreeHead(t *testing.T) {
	// The new stored tree head no longer populates its AuditorHead or TreeHead.Signature fields
	newStoredTreeHead := &storedTreeHead{
		TreeHead: &TransparencyTreeHead{
			TreeSize:  123,
			Timestamp: 123456,
			//Signature: random(32),
			Signatures: []*Signature{
				{random(32), random(32)},
			},
		},
		//AuditorHead: &AuditorTreeHead{},
		AuditorHeads: map[string]*AuditorTreeHead{
			"example-auditor": {
				AuditorTransparencyTreeHead: AuditorTransparencyTreeHead{
					TreeSize:  234,
					Timestamp: 234567,
					Signature: random(32),
				},
				RootValue:   random(32),
				Consistency: [][]byte{random(32), random(32)},
			},
		},
	}

	bytes, err := json.Marshal(newStoredTreeHead)
	if err != nil {
		t.Fatal("expected no error marshaling new stored tree head")
	}

	actualJsonString := string(bytes)
	expectedJsonString := `{
	"head": {"n": 123, "ts": 123456, "sigs": "<<PRESENCE>>"},
	"auditor-heads": {
		"example-auditor": {"n": 234, "ts": 234567, "sig": "<<PRESENCE>>", "root": "<<PRESENCE>>", "consistency": "<<PRESENCE>>"}
	}}`

	// go 1.24 requires a constant format string to Printf-like functions
	jsonassert.New(t).Assertf(actualJsonString, "%s", expectedJsonString)
}

func random(numBytes int) []byte {
	out := make([]byte, numBytes)
	if _, err := rand.Read(out); err != nil {
		panic(err)
	}
	return out
}
