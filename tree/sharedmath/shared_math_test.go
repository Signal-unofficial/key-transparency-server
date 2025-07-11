//
// Copyright 2025 Signal Messenger, LLC
// SPDX-License-Identifier: AGPL-3.0-only
//

package sharedmath

import (
	"math"
	"testing"
)

func assert(ok bool) {
	if !ok {
		panic("Assertion failed.")
	}
}

func TestMath(t *testing.T) {
	assert(Log2(0) == 0)
	assert(Log2(8) == 3)
	assert(Log2(10000) == 13)

	assert(Level(1) == 1)
	assert(Level(2) == 0)
	assert(Level(3) == 2)

	assert(Left(7) == 3)
}

func TestLeft_LeafNode(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected a panic value")
		}
	}()
	Left(0)
}

func TestRightStep_LeafNode(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected a panic value")
		}
	}()
	RightStep(0)
}

func TestRightStep_MaxUint64(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected a panic value")
		}
	}()
	RightStep(math.MaxUint64)
}

func TestParent_MaxUint64(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected a panic value")
		}
	}()
	Parent(math.MaxUint64, math.MaxUint64)
}
