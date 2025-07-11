//
// Copyright 2025 Signal Messenger, LLC
// SPDX-License-Identifier: AGPL-3.0-only
//

package db

import (
	"testing"
)

var testNewCachedTransparencyStore = []struct {
	cachesToEnable    Bitmask
	expectPrefixCache bool
	expectLogCache    bool
	expectTopCache    bool
	expectHeadCache   bool
}{
	{PrefixCache, true, false, false, false},
	{TransparencyCache | PrefixCache | LogCache | HeadCache, true, true, true, true},
	{TransparencyCache | PrefixCache | LogCache, true, true, true, false},
}

func TestNewCachedTransparencyStore(t *testing.T) {
	mockDb := NewMemoryTransparencyStore()

	for _, p := range testNewCachedTransparencyStore {
		cachedStore := NewCachedTransparencyStore(mockDb, p.cachesToEnable, 2000, 2000, 20000).(*cachedTransparencyStore)

		if p.expectPrefixCache != (cachedStore.prefixCache != nil) {
			t.Fatalf("Expect prefix cache: %v, cachedStore.prefixCache != nil is %v", p.expectPrefixCache, cachedStore.prefixCache != nil)
		}

		if p.expectLogCache != (cachedStore.logCache != nil) {
			t.Fatalf("Expect log cache: %v, cachedStore.logCache != nil is %v", p.expectLogCache, cachedStore.logCache != nil)
		}

		if p.expectTopCache != (cachedStore.topCache != nil) {
			t.Fatalf("Expect top cache: %v, cachedStore.topCache != nil is %v", p.expectTopCache, cachedStore.topCache != nil)
		}

		if p.expectHeadCache != (cachedStore.head != nil) {
			t.Fatalf("Expect head cache: %v, cachedStore.head != nil is %v", p.expectHeadCache, cachedStore.head != nil)
		}

	}
}
