//
// Copyright 2025 Signal Messenger, LLC
// SPDX-License-Identifier: AGPL-3.0-only
//

package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/signalapp/keytransparency/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type QueryArgs struct {
	Aci                   []byte
	AciIdentityKey        []byte
	E164                  string
	UnidentifiedAccessKey []byte
	UsernameHash          []byte
}

func extractQueryArgs(command string) QueryArgs {
	if flag.Arg(1) == "" {
		log.Fatalf("No ACI given. Usage: kt-client [-e164 e164] [-uak base64_encoded_uak] [-username-hash base64url_encoded_username_hash] %s <UUID> <base64_encoded_aci_identity_key>", command)
	} else if flag.Arg(2) == "" {
		log.Fatalf("No ACI identity key given. Usage: kt-client [-e164 e164] [-uak base64_encoded_uak] [-username-hash base64url_encoded_username_hash] %s <UUID> <base64_encoded_aci_identity_key>", command)
	}

	aci, err := uuid.Parse(flag.Arg(1))
	checkErr("invalid UUID string for ACI", err)

	aciBytes, err := aci.MarshalBinary()
	checkErr("getting UUID bytes", err)

	aciIdentityKeyBytes, err := base64.StdEncoding.DecodeString(flag.Arg(2))
	checkErr("decoding base64 encoding for ACI identity key", err)

	var unidentifiedAccessKeyBytes []byte
	if *uak == "" {
		unidentifiedAccessKeyBytes = db.UnidentifiedAccessKey
	} else {
		unidentifiedAccessKeyBytes, err = base64.StdEncoding.DecodeString(*uak)
		checkErr("decoding base64 encoding for unidentified access key", err)
	}

	var usernameHashBytes []byte
	if *usernameHash != "" {
		usernameHashBytes, err = base64.URLEncoding.DecodeString(*usernameHash)
		checkErr("decoding base64url encoding for username hash", err)
	}

	return QueryArgs{
		aciBytes,
		aciIdentityKeyBytes,
		*e164,
		unidentifiedAccessKeyBytes,
		usernameHashBytes,
	}
}

type SamplingArgs struct {
	SampleSize int
	NumSamples int
}

func extractSamplingArgs() SamplingArgs {
	var sampleSize int
	if *timingSampleSize == 0 {
		log.Fatal("sample size cannot be 0")
	}

	var numSamples int
	if *timingNumSamples == 0 {
		log.Fatal("number of samples cannot be 0")

	}
	return SamplingArgs{
		SampleSize: sampleSize,
		NumSamples: numSamples,
	}
}

func timeRequest(grpcCall func() error, samplingArgs SamplingArgs) {
	for j := 0; j < samplingArgs.NumSamples; j++ {
		fmt.Printf("\n\nRound %d of %d:\n", j, samplingArgs.NumSamples)
		var totalDuration time.Duration
		var minDuration time.Duration
		var maxDuration time.Duration

		minDuration = time.Hour
		for i := 0; i < samplingArgs.SampleSize; i++ {
			start := time.Now()
			err := grpcCall()
			duration := time.Since(start)

			gprcError, ok := status.FromError(err)
			if !ok {
				fmt.Printf("Could not parse err: %v", err)
			}

			if gprcError.Code() == codes.NotFound {
				if i%10 == 0 {
					fmt.Printf("Request %d returned not found: %.5f seconds\n", i, duration.Seconds())
				}
			} else if gprcError.Code() == codes.OK {
				if i%10 == 0 {
					fmt.Printf("Request %d succeeded: %.5f seconds\n", i, duration.Seconds())
				}
			} else {
				log.Fatalf("Unexpected error: %v", err)
			}

			totalDuration += duration

			if duration < minDuration {
				minDuration = duration
			}

			if duration > maxDuration {
				maxDuration = duration
			}
		}

		avgDuration := totalDuration / time.Duration(samplingArgs.SampleSize)
		fmt.Println("\nResults:")
		fmt.Printf("  Min latency: %.5f seconds\n", minDuration.Seconds())
		fmt.Printf("  Max latency: %.5f seconds\n", maxDuration.Seconds())
		fmt.Printf("  Avg latency: %.5f seconds\n", avgDuration.Seconds())
	}
}

func checkErr(context string, err error) {
	if err != nil {
		_, _ = os.Stderr.WriteString(fmt.Sprintf("%s: %v\n", context, err))
		os.Exit(1)
	}
}
