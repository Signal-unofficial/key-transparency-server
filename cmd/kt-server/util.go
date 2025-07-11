//
// Copyright 2025 Signal Messenger, LLC
// SPDX-License-Identifier: AGPL-3.0-only
//

package main

import (
	"bytes"
	"context"
	"crypto/subtle"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/signalapp/keytransparency/cmd/internal/config"
	"github.com/signalapp/keytransparency/tree/transparency/pb"
)

const (
	AuditorNameContextKey = "auditor-name"
	HeaderValueContextKey = "header-value"
)

func verifyMappedValueConstantTime(mappedValue, expectedValue []byte) error {
	if 1 != subtle.ConstantTimeCompare(expectedValue, mappedValue) {
		return status.Error(codes.PermissionDenied, "provided value does not match expected value")
	}
	return nil
}

// createDistinctValue returns a value that is different from the given []byte
func createDistinctValue(value []byte) []byte {
	if len(value) < 1 {
		// This should only ever happen in the case of a programmer error.
		return []byte{0}
	}
	distinctValue := make([]byte, len(value))
	copy(distinctValue, value)
	distinctValue[0] = distinctValue[0] + 1
	return distinctValue
}

func getServerOptions(config *config.ServiceConfig, additionalInterceptors []grpc.UnaryServerInterceptor) []grpc.ServerOption {
	if config.AuthorizedHeaders == nil || len(config.AuthorizedHeaders) == 0 {
		return nil
	}

	interceptors := []grpc.UnaryServerInterceptor{func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unavailable, "metadata read error")
		}

		matchedHeaderValue, err := validateAuthorizedHeaders(config.AuthorizedHeaders, md)
		if err != nil {
			return nil, err
		}

		// Store the matched header value in the context for downstream interceptors
		ctx = context.WithValue(ctx, HeaderValueContextKey, matchedHeaderValue)

		return handler(ctx, req)
	}}

	if len(additionalInterceptors) > 0 {
		interceptors = append(interceptors, additionalInterceptors...)
	}

	return []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(interceptors...),
	}
}

func storeAuditorNameInterceptor(config *config.ServiceConfig) func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		headerValue, ok := ctx.Value(HeaderValueContextKey).(string)
		if !ok {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid type for header value. expected string, got %T", headerValue))
		}

		if len(headerValue) == 0 {
			return nil, status.Error(codes.InvalidArgument, "no matched header value in context")
		}

		auditorName := config.HeaderValueToAuditorName[headerValue]
		if len(auditorName) == 0 {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("auditor name not specified for header value: %s", headerValue))
		}

		// Store the auditor name in the context
		ctx = context.WithValue(ctx, AuditorNameContextKey, auditorName)
		return handler(ctx, req)
	}
}

// validateAuthorizedHeaders ensures that at least one of the specified header to value mappings is present on the request
// Returns the last header value that matched.
func validateAuthorizedHeaders(authorizedHeaders map[string][]string, md metadata.MD) (string, error) {
	if authorizedHeaders == nil || len(authorizedHeaders) == 0 {
		return "", nil
	}

	passedValidation := false
	matchedValue := ""
	for header, authorizedHeaderValues := range authorizedHeaders {
		requestHeaderValues := md.Get(header)
		if len(requestHeaderValues) == 0 {
			continue
		}
		for _, requestHeaderValue := range requestHeaderValues {
			for _, authorizedValue := range authorizedHeaderValues {
				if subtle.ConstantTimeCompare([]byte(authorizedValue), []byte(requestHeaderValue)) == 1 {
					matchedValue = requestHeaderValue
					passedValidation = true
				}
			}
		}
	}

	if !passedValidation {
		return "", status.Error(codes.PermissionDenied, fmt.Sprintf("invalid header values"))
	}

	return matchedValue, nil
}

func isTombstoneUpdate(updateRequest *pb.UpdateRequest) bool {
	return bytes.Equal(updateRequest.GetValue(), tombstoneBytes)
}
