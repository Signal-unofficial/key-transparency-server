//
// Copyright 2025 Signal Messenger, LLC
// SPDX-License-Identifier: AGPL-3.0-only
//

package transparency

import (
	"errors"
	"sort"

	"github.com/signalapp/keytransparency/tree/sharedmath"
	"github.com/signalapp/keytransparency/tree/transparency/math"
)

type idCtrPair struct {
	id  uint64
	ctr uint32
}

type idCtrSlice []idCtrPair

func (s idCtrSlice) Len() int           { return len(s) }
func (s idCtrSlice) Less(i, j int) bool { return s[i].id < s[j].id }
func (s idCtrSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// proofGuide contains the data necessary to carry out a search of the implicit binary tree
// for a given index.
// The implicit binary tree is formed by arranging the log entries into a left-balanced binary search formation,
// where each node contains an entry ID and a counter value that denotes which "version" of a given index exists in that entry.
// Unlike a typical binary search that would move left or right based on the entry ID alone, this search moves left and right
// based on two criteria:
//  1. The ctr value for each entry ID in the tree which is stored in the sorted field alongside the entry ID.
//     If the ctr value at the current node is less than targetCtr, the search moves left; otherwise it moves right.
//  2. However, the search doesn't necessarily move to the immediate left or right child. Instead, it moves
//     to the first node within its left or right subtree that satisfies the upper and lower bound,
//     which are determined by firstUpdatePosition and treeSize.
//
// The search terminates when it hits a leaf node or when it lands on the lower or upper bound entry ID.
type proofGuide struct {
	firstUpdatePosition, treeSize uint64   // Position of the index's first occurrence in the log, and size of log.
	targetCtr                     uint32   // The counter of the index we're searching for
	ids                           []uint64 // List of entry IDs to traverse in the search.

	// List of entry IDs that have been traversed so far in the search,
	// along with the counter of the target index found at each entry.
	sorted idCtrSlice

	// Whether the search is at the frontier of the log.
	// This always starts out as true and is set to false once the search steps into a left or right subtree.
	frontier bool
}

// mostRecentProofGuide initializes the data structure used to search for the most recent occurrence of an index.
// Specifically, it adds all frontier nodes that fall within the search space to the list of initial nodes to search.
func mostRecentProofGuide(firstUpdatePosition, treeSize uint64) *proofGuide {
	frontier := []uint64{math.Root(firstUpdatePosition, treeSize)}
	for frontier[len(frontier)-1] != treeSize-1 {
		frontier = append(frontier, math.Right(frontier[len(frontier)-1], firstUpdatePosition, treeSize))
	}
	return &proofGuide{firstUpdatePosition: firstUpdatePosition, treeSize: treeSize, ids: frontier, frontier: true}
}

// done returns true if the search proof is finished.
// Otherwise, it looks up and stores the next entry ID to search.
func (pg *proofGuide) done() (bool, error) {
	if len(pg.ids) > len(pg.sorted) {
		return false, nil
	}
	sort.Sort(pg.sorted)

	// Check that the list of counters is monotonic.
	for i := 1; i < len(pg.sorted); i++ {
		if pg.sorted[i-1].ctr > pg.sorted[i].ctr {
			return false, errors.New("set of counters given is not monotonic")
		}
	}

	// Determine the "last" id looked up. Generally this is actually just the
	// last id that was looked up, but if we just fetched the frontier then we
	// start searching at the first element of the frontier with the greatest ctr.
	var last uint64
	if pg.frontier {
		// The ctr of the index that we're searching for is determined by the most recent entry in the log.
		pg.targetCtr = pg.sorted[len(pg.sorted)-1].ctr

		for i := 0; i < len(pg.sorted); i++ {
			if pg.sorted[i].ctr == pg.targetCtr {
				last = pg.sorted[i].id

				break
			}
		}

		pg.frontier = false
	} else {
		last = pg.ids[len(pg.ids)-1]
	}
	if sharedmath.IsLeaf(last) {
		return true, nil
	}

	// Find the counter associated with the last id looked up.
	var ctr uint32
	for _, pair := range pg.sorted {
		if pair.id == last {
			ctr = pair.ctr
			break
		}
	}

	// Find the next id to lookup by moving left or right depending on ctr.
	if ctr < pg.targetCtr {
		if last == pg.treeSize-1 {
			return true, nil
		}
		pg.ids = append(pg.ids, math.Right(last, pg.firstUpdatePosition, pg.treeSize))
	} else {
		if last == pg.firstUpdatePosition {
			return true, nil
		}
		pg.ids = append(pg.ids, math.Left(last, pg.firstUpdatePosition, pg.treeSize))
	}
	return false, nil
}

// next returns the next id to fetch from the database.
func (pg *proofGuide) next() uint64 { return pg.ids[len(pg.sorted)] }

// insert adds an id-counter pair to the guide.
func (pg *proofGuide) insert(id uint64, ctr uint32) {
	pg.sorted = append(pg.sorted, idCtrPair{id, ctr})
}

// final returns the index that represents the final search result.
func (pg *proofGuide) final() int {
	if pg.frontier {
		panic("unexpected error")
	}

	smallest := 0
	for pg.sorted[smallest].ctr < pg.targetCtr {
		smallest++
	}
	if pg.sorted[smallest].ctr != pg.targetCtr {
		return -1
	}

	// Return the index of the search that contains the result we want.
	for i := 0; i < len(pg.ids); i++ {
		if pg.ids[i] == pg.sorted[smallest].id {
			return i
		}
	}
	panic("unexpected error")
}
