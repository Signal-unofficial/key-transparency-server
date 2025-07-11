//
// Copyright 2025 Signal Messenger, LLC
// SPDX-License-Identifier: AGPL-3.0-only
//

package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var testAuditEndpointsParameters = []struct {
	key          string
	value        string
	expectError  bool
	expectedName string
}{
	// No metadata
	{"", "", true, ""},
	// Wrong metadata key
	{"wrong-metadata-key", "", true, ""},
	{"wrong-metadata-key", "some-value", true, ""},
	// Right metadata key, no auditor name
	{AuditorNameContextKey, "", true, ""},
	{AuditorNameContextKey, "some-value", false, "some-value"},
}

func TestExtractAuditorName(t *testing.T) {
	for _, p := range testAuditEndpointsParameters {
		ctx := context.Background()
		if len(p.key) > 0 {
			ctx = context.WithValue(ctx, p.key, p.value)
		}

		name, err := extractAuditorName(ctx)
		if p.expectError {
			if grpcError, ok := status.FromError(err); grpcError.Code() != codes.InvalidArgument || !ok {
				t.Fatalf("Expected %v, got %v", codes.InvalidArgument, err)
			}
		} else {
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
		}
		assert.Equal(t, p.expectedName, name)
	}
}
