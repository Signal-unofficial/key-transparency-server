// Copyright 2016 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commitments

import (
	"bytes"
	"encoding/hex"
	"errors"
	"testing"
)

// Use a constant key of zero to obtain consistent test vectors.
var zeroKey = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

func TestCommit(t *testing.T) {
	for _, tc := range []struct {
		userID, data   string
		muserID, mdata string
		mutate         bool
		want           error
	}{
		{"foo", "bar", "foo", "bar", false, nil},
		{"foo", "bar", "fo", "obar", false, ErrInvalidCommitment},
		{"foo", "bar", "foob", "ar", false, ErrInvalidCommitment},
		{"foo", "bar", "foo", "bar", true, ErrInvalidCommitment},
	} {
		data := []byte(tc.data)
		c, err := Commit([]byte(tc.userID), data, zeroKey)
		if err != nil {
			t.Errorf("Unexpected error calculating commitment: %v", err)
		}
		if tc.mutate {
			c[0] ^= 1
		}
		if got := Verify([]byte(tc.muserID), c, data, zeroKey); !errors.Is(got, tc.want) {
			t.Errorf("Verify(%v, %x, %x, %x): %v, want %v", tc.userID, c, data, zeroKey, got, tc.want)
		}
	}
}

func TestInvalidNonces(t *testing.T) {
	nonce1 := []byte{}
	nonce2 := []byte{0x00, 0x02}

	commit1, err := Commit([]byte{0x00, 0x00}, []byte{0x00, 0x00, 0x00, 0x00}, nonce1)
	if err == nil || err != ErrInvalidNonce {
		t.Errorf("Expected %v in calculating commitment, got no error instead", ErrInvalidNonce)
	}

	commit2, err := Commit([]byte{0x00, 0x00}, []byte{0x00, 0x00, 0x00, 0x00}, nonce2)
	if err == nil || err != ErrInvalidNonce {
		t.Errorf("Expected %v in calculating commitment, got no error instead", ErrInvalidNonce)
	}

	if commit1 != nil || commit2 != nil {
		t.Errorf("Expected no commitments, got: \ncommit1=%x\ncommit2=%x", commit1, commit2)
	}
}

func TestVectors(t *testing.T) {
	for _, tc := range []struct {
		userID, data string
		want         []byte
	}{
		{"", "", dh("edc3f59798cd87f2f48ec8836e2b6ef425cde9ab121ffdefc93d769db7cebabf")},
		{"foo", "bar", dh("25df431e884358826fe66f96d65702580104240abd63fa741d9ea3f32914bbf5")},
		{"foo1", "bar", dh("6c31a163a7660d1467fc1c997bd78b0a70b8921ca76b7eb0c6ca077f1e5e121e")},
		{"foo", "bar1", dh("5de6c6c9ed4bf48122f6c851c80e6eacbf885947f02f974cdc794b14c8e975f1")},
	} {
		userID := []byte(tc.userID)
		data := []byte(tc.data)
		got, err := Commit(userID, data, zeroKey)
		if err != nil {
			t.Errorf("Unexpected error calculating commitment: %v", err)
		}
		if !bytes.Equal(got, tc.want) {
			t.Errorf("Commit(%v, %v): %x ,want %x", tc.userID, tc.data, got, tc.want)
		}
	}
}

func BenchmarkCommit(b *testing.B) {
	searchKey := make([]byte, 32)
	data := make([]byte, 500)
	nonce := make([]byte, 16)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Commit(searchKey, data, nonce)
	}
}

func dh(h string) []byte {
	result, err := hex.DecodeString(h)
	if err != nil {
		panic("DecodeString failed")
	}
	return result
}
