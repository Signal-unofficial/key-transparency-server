//
// Copyright 2025 Signal Messenger, LLC
// SPDX-License-Identifier: AGPL-3.0-only
//

package test

import (
	"crypto/ed25519"
	"crypto/rand"
	mrand "math/rand"
	"slices"
	"testing"

	edvrf "github.com/signalapp/keytransparency/crypto/vrf/ed25519"
	"github.com/signalapp/keytransparency/db"
	"github.com/signalapp/keytransparency/tree/transparency"
	"github.com/signalapp/keytransparency/tree/transparency/pb"
)

const (
	exampleAuditorName1 = "example-auditor1"
	exampleAuditorName2 = "example-auditor2"
)

func random() []byte {
	out := make([]byte, 16)
	if _, err := rand.Read(out); err != nil {
		panic(err)
	}
	return out
}

// MemoryClientStorage implements the ClientStorage interface in-memory.
type MemoryClientStorage struct {
	config *transparency.PublicConfig
	head   *db.TransparencyTreeHead
	root   []byte
	data   map[string]*transparency.MonitoringData
}

func (m *MemoryClientStorage) PublicConfig() *transparency.PublicConfig { return m.config }

func (m *MemoryClientStorage) GetLastTreeHead() (*db.TransparencyTreeHead, []byte, error) {
	return m.head, m.root, nil
}

func (m *MemoryClientStorage) SetLastTreeHead(head *db.TransparencyTreeHead, root []byte) error {
	m.head, m.root = head, root
	return nil
}

func (m *MemoryClientStorage) GetData(key []byte) (*transparency.MonitoringData, error) {
	return m.data[string(key)], nil
}

func (m *MemoryClientStorage) SetData(key []byte, data *transparency.MonitoringData) error {
	m.data[string(key)] = data
	return nil
}

// Last returns the correct "last" parameter for a request to the transparency
// tree, according to the provided client storage.
func Last(store transparency.ClientStorage) *pb.Consistency {
	head, _, err := store.GetLastTreeHead()
	if err != nil {
		panic(err)
	} else if head == nil {
		return &pb.Consistency{}
	}
	return &pb.Consistency{Last: &head.TreeSize}
}

func NewTree(t testing.TB, deploymentMode transparency.DeploymentMode) (*transparency.Tree, *MemoryClientStorage, *transparency.PrivateConfig, []ed25519.PrivateKey) {
	_, sigKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	vrfPriv, _ := edvrf.GenerateKey()
	prefixAesKey := make([]byte, 32)
	if _, err := rand.Read(prefixAesKey); err != nil {
		t.Fatal(err)
	}
	openingKey := make([]byte, 32)
	if _, err := rand.Read(openingKey); err != nil {
		t.Fatal(err)
	}

	config := &transparency.PrivateConfig{
		Mode:         deploymentMode,
		SigKey:       sigKey,
		VrfKey:       vrfPriv,
		PrefixAesKey: prefixAesKey,
		OpeningKey:   openingKey,
	}

	var auditorPrivateKeys []ed25519.PrivateKey
	if deploymentMode == transparency.ThirdPartyAuditing {
		auditor1PublicKey, auditor1PrivateKey, err := ed25519.GenerateKey(nil)
		auditor2PublicKey, auditor2PrivateKey, err := ed25519.GenerateKey(nil)
		if err != nil {
			t.Fatal(err)
		}
		config.AuditorKeys = map[string]ed25519.PublicKey{
			exampleAuditorName1: auditor1PublicKey,
			exampleAuditorName2: auditor2PublicKey,
		}
		auditorPrivateKeys = []ed25519.PrivateKey{auditor1PrivateKey, auditor2PrivateKey}
	}

	tree, err := transparency.NewTree(config, db.NewMemoryTransparencyStore())
	if err != nil {
		t.Fatal(err)
	}

	store := &MemoryClientStorage{
		config: config.Public(),
		data:   make(map[string]*transparency.MonitoringData),
	}

	return tree, store, config, auditorPrivateKeys
}

func RandomTree(tree *transparency.Tree, store transparency.ClientStorage, total int, keys, repeats []int) ([][]byte, error) {
	var chosen [][]byte

	for i := 0; i < total; i++ {
		keep := slices.Contains(keys, i)
		repeat := slices.Contains(repeats, i)

		if i == 0 || keep || repeat || mrand.Intn(2) == 0 {
			var newKey []byte
			if repeat {
				newKey = chosen[0]
			} else {
				newKey = random()
			}
			if keep {
				chosen = append(chosen, newKey)
			}

			req := &pb.UpdateRequest{
				SearchKey:   newKey,
				Value:       random(),
				Consistency: Last(store),
			}
			res, err := tree.UpdateSimple(req)
			if err != nil {
				return nil, err
			} else if err := transparency.VerifyUpdate(store, req, res); err != nil {
				return nil, err
			}
		} else {
			if err := tree.BatchUpdateFake(1); err != nil {
				return nil, err
			}
		}
	}

	return chosen, nil
}
