//
// Copyright 2025 Signal Messenger, LLC
// SPDX-License-Identifier: AGPL-3.0-only
//

package prefix

import (
	"fmt"
	mrand "math/rand"
	"testing"

	"github.com/signalapp/keytransparency/db"
)

// prefixStore allows tests to track a search's lookup pattern and explicitly
// decide which entries are cached.
type prefixStore struct {
	inner   db.PrefixStore
	lookups [][]uint64
	cache   map[uint64][]byte
}

// Reset clears the store's lookup history and cache.
func (ps *prefixStore) Reset() {
	ps.lookups = nil
	ps.cache = make(map[uint64][]byte)
}

// Cache adds an entry to cache.
func (ps *prefixStore) Cache(key uint64) {
	raw, err := ps.inner.BatchGet([]uint64{key})
	if err != nil {
		panic(err)
	}
	ps.cache[key] = raw[key]
}

func (ps *prefixStore) BatchGet(keys []uint64) (map[uint64][]byte, error) {
	ps.lookups = append(ps.lookups, keys)
	return ps.inner.BatchGet(keys)
}

func (ps *prefixStore) GetCached(key uint64) []byte { return ps.cache[key] }

func (ps *prefixStore) Put(key uint64, data []byte) {
	delete(ps.cache, key)
	ps.inner.Put(key, data)
}

func first(lookups [][]uint64) (out []uint64) {
	for _, ids := range lookups {
		if len(ids) != 1 {
			panic("more than one id in lookup step")
		}
		out = append(out, ids[0])
	}
	return
}

// mergePath takes two lookup paths as input and returns the expected lookup
// path of a batch search for the same keys.
func mergePaths(path1, path2 []uint64, cached map[uint64][]byte) (out [][]uint64) {
	dedup := make(map[uint64]struct{})
	for id := range cached {
		dedup[id] = struct{}{}
	}

	i, j := 0, 0
	for i < len(path1) && j < len(path2) {
		x, y := path1[i], path2[j]
		_, ok1 := dedup[x]
		_, ok2 := dedup[y]

		if ok1 {
			i++
		}
		if ok2 {
			j++
		}
		if !ok1 && !ok2 {
			if x == y {
				out = append(out, []uint64{x})
			} else if x < y {
				out = append(out, []uint64{x, y})
			} else {
				out = append(out, []uint64{y, x})
			}
			dedup[x], dedup[y] = struct{}{}, struct{}{}
			i++
			j++
		}
	}
	for i < len(path1) {
		x := path1[i]
		if _, ok := dedup[x]; !ok {
			out = append(out, []uint64{x})
			dedup[x] = struct{}{}
		}
		i++
	}
	for j < len(path2) {
		y := path2[j]
		if _, ok := dedup[y]; !ok {
			out = append(out, []uint64{y})
			dedup[y] = struct{}{}
		}
		j++
	}

	return
}

func TestLookupPattern(t *testing.T) {
	store := &prefixStore{inner: db.NewMemoryTransparencyStore().PrefixStore()}
	store.Reset()
	tree := NewTree(make([]byte, 16), store)

	// Insert a bunch of random keys.
	var key []byte
	for version := uint64(0); version < 300; version++ {
		temp := random()
		_, _, err := tree.Insert(version, temp, make([]byte, 32), false)
		if err != nil {
			t.Fatal(err)
		} else if version == 9 {
			key = temp
		}
	}

	for i := 0; i < 100; i++ {
		store.cache = make(map[uint64][]byte)

		// Choose two random versions to do lookups in.
		ver1 := uint64(mrand.Intn(290) + 10)
		ver2 := uint64(mrand.Intn(290) + 10)
		if ver1 == ver2 {
			continue
		}

		// Find the uncached, unbatched lookup path for each lookup.
		store.Reset()
		if _, err := tree.Search(ver1, key); err != nil {
			t.Fatal(err)
		}
		path1 := first(store.lookups)

		store.Reset()
		if _, err := tree.Search(ver2, key); err != nil {
			t.Fatal(err)
		}
		path2 := first(store.lookups)

		// Cache some random entries.
		store.Reset()
		dedup := make(map[uint64]struct{})
		for _, x := range append(path1, path2...) {
			dedup[x] = struct{}{}
		}
		for x := range dedup {
			if mrand.Intn(3) == 0 {
				store.Cache(x)
			}
		}

		// Calculate expected lookup path.
		expected := mergePaths(path1, path2, store.cache)

		// Do the actual lookup and see if that's what happens.
		search1, err := tree.BatchSearch(ver1, key)
		if err != nil {
			t.Fatal(err)
		}
		search2, err := tree.BatchSearch(ver2, key)
		if err != nil {
			t.Fatal(err)
		}
		_, err = tree.BatchExec([]*Search{search1, search2})
		if err != nil {
			t.Fatal(err)
		} else if fmt.Sprint(store.lookups) != fmt.Sprint(expected) {
			t.Fatal("unexpected lookup pattern")
		}
	}
}
