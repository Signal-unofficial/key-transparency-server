//
// Copyright 2025 Signal Messenger, LLC
// SPDX-License-Identifier: AGPL-3.0-only
//

package prefix

import (
	"bytes"
	"crypto/rand"
	mrand "math/rand"
	"slices"
	"testing"

	"github.com/signalapp/keytransparency/db"
)

func random() []byte {
	out := make([]byte, IndexLength)
	if _, err := rand.Read(out); err != nil {
		panic(err)
	}
	return out
}

func TestPrefixTree(t *testing.T) {
	var (
		tree = NewTree(make([]byte, 16), db.NewMemoryTransparencyStore().PrefixStore())

		keys                 [][]byte
		ctr                  []uint32
		firstUpdatePosition  []uint64
		latestUpdatePosition []uint64

		version uint64
		root    []byte
	)

	for i := 0; i < 1000; i++ {
		dice := mrand.Intn(4)

		if dice == 0 && len(keys) > 0 { // Search for an existing index.
			k := mrand.Intn(len(keys))

			res, err := tree.Search(version, keys[k])
			if err != nil {
				t.Fatal(err)
			} else if err = Verify(root, keys[k], firstUpdatePosition[k], res); err != nil {
				t.Fatal(err)
			} else if res.Counter != ctr[k] {
				t.Fatal("unexpected value for version counter")
			} else if res.FirstUpdatePosition != firstUpdatePosition[k] {
				t.Fatal("unexpected value for first update position")
			} else if res.LatestUpdatePosition != latestUpdatePosition[k] {
				t.Fatal("unexpected value for latest update position")
			}
		} else if dice == 1 && len(keys) > 0 { // Insert a fake entry.
			temp, err := tree.InsertFake(version)
			if err != nil {
				t.Fatal(err)
			}
			version, root = version+1, temp
		} else if dice == 2 { // Insert a new index.
			key := random()
			temp, _, err := tree.Insert(version, key, make([]byte, 32), false)
			if err != nil {
				t.Fatal(err)
			}
			keys = append(keys, key)
			ctr = append(ctr, 0)
			firstUpdatePosition = append(firstUpdatePosition, version)
			latestUpdatePosition = append(latestUpdatePosition, version)
			version, root = version+1, temp
		} else if dice == 3 && len(keys) > 0 { // Insert an existing index.
			k := mrand.Intn(len(keys))
			temp, _, err := tree.Insert(version, keys[k], make([]byte, 32), false)
			if err != nil {
				t.Fatal(err)
			}
			ctr[k] += 1
			latestUpdatePosition[k] = version
			version, root = version+1, temp
		}
	}
}

func TestBatchSearch(t *testing.T) {
	var (
		tree     = NewTree(make([]byte, 16), db.NewMemoryTransparencyStore().PrefixStore())
		versions = []uint64{20, 50, 100, 200, 250}

		key         []byte
		roots       [][]byte
		commitments [][]byte
	)
	for version := uint64(0); version < 300; version++ {
		temp := random()
		root, sr, err := tree.Insert(version, temp, make([]byte, 32), false)
		if err != nil {
			t.Fatal(err)
		}

		if version == 10 {
			key = temp
		} else if slices.Contains(versions, version+1) {
			roots = append(roots, root)
			commitments = append(commitments, sr.Commitment)
		}
	}

	searches := make([]*Search, 0)
	for _, version := range versions {
		search, err := tree.BatchSearch(version, key)
		if err != nil {
			t.Fatal(err)
		}
		searches = append(searches, search)
	}
	results, err := tree.BatchExec(searches)
	if err != nil {
		t.Fatal(err)
	}
	for i, res := range results {
		if err = Verify(roots[i], key, 10, res); err != nil {
			t.Fatal(err)
		} else if res.FirstUpdatePosition != 10 {
			t.Fatal("unexpected value for first update position")
		} else if res.LatestUpdatePosition != 10 {
			t.Fatal("unexpected value for version position")
		} else if !bytes.Equal(res.Commitment, commitments[i]) {
			t.Fatal("unexpected value for commitment")
		}
	}
}

func randomEntry() Entry {
	return Entry{Index: random(), Commitment: random()}
}

func TestBatchInsertFakeUpdatesEmptyTree(t *testing.T) {
	// Build a set of entries to add.
	entries := make([]Entry, 0)
	for i := 0; i < 10; i++ {
		entries = append(entries, randomEntry())
	}

	tree := NewTree(make([]byte, 16), db.NewMemoryTransparencyStore().PrefixStore())
	_, _, err := tree.BatchInsert(0, entries, true)
	if err == nil {
		t.Fatal("expected error inserting fake entries into an empty tree")
	}
}

func TestBatchInsert(t *testing.T) {
	// Build a set of entries to add.
	entries := make([]Entry, 0)
	for i := 0; i < 100; i++ {
		entries = append(entries, randomEntry())
	}
	for i := 75; i < 78; i++ {
		entries[i].Index = entries[70].Index
	}
	for i := 81; i < 84; i++ {
		entries[i].Index = entries[10].Index
	}

	// Insert into the tree in batches.
	tree1 := NewTree(make([]byte, 16), db.NewMemoryTransparencyStore().PrefixStore())
	_, _, err := tree1.BatchInsert(0, entries[:50], false)
	if err != nil {
		t.Fatal(err)
	}
	roots, _, err := tree1.BatchInsert(50, entries[50:], false)
	if err != nil {
		t.Fatal(err)
	}
	root1 := roots[49]

	// Insert into the tree one-by-one.
	tree2 := NewTree(make([]byte, 16), db.NewMemoryTransparencyStore().PrefixStore())
	var root2 []byte
	for i, entry := range entries {
		root2, _, err = tree2.Insert(uint64(i), entry.Index, entry.Commitment, false)
		if err != nil {
			t.Fatal(err)
		}
	}

	if !bytes.Equal(root1, root2) {
		t.Fatal("roots do not match")
	}
}

func BenchmarkInsert(b *testing.B) {
	tree := NewTree(make([]byte, 16), db.NewMemoryTransparencyStore().PrefixStore())
	for i := 0; i < 100; i++ {
		entry := randomEntry()
		_, _, err := tree.Insert(uint64(i), entry.Index, entry.Commitment, false)
		if err != nil {
			b.Fatal(err)
		}
	}
	entries := make([]Entry, b.N)
	for i := range entries {
		entries[i] = randomEntry()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := tree.Insert(100+uint64(i), entries[i].Index, entries[i].Commitment, false)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBatchInsert(b *testing.B) {
	const batchSize = 80

	tree := NewTree(make([]byte, 16), db.NewMemoryTransparencyStore().PrefixStore())
	for i := 0; i < 100; i++ {
		entry := randomEntry()
		_, _, err := tree.Insert(uint64(i), entry.Index, entry.Commitment, false)
		if err != nil {
			b.Fatal(err)
		}
	}
	entries := make([]Entry, batchSize*b.N)
	for i := range entries {
		entries[i] = randomEntry()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := batchSize * i
		end := batchSize * (i + 1)
		_, _, err := tree.BatchInsert(100+uint64(start), entries[start:end], false)
		if err != nil {
			b.Fatal(err)
		}
	}
}
