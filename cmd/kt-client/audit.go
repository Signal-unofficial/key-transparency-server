//
// Copyright 2025 Signal Messenger, LLC
// SPDX-License-Identifier: AGPL-3.0-only
//

package main

import (
	"context"
	"flag"
	"log"
	"strconv"

	"github.com/signalapp/keytransparency/cmd/kt-server/pb"
)

func handleAudit(client pb.KeyTransparencyAuditorServiceClient) {
	if flag.Arg(1) == "" {
		log.Fatal("No starting position given. Usage: kt-client audit <start> <limit>")
	} else if flag.Arg(2) == "" {
		log.Fatal("No entry limit given. Usage: kt-client audit <start> <limit>")
	}
	start, err := strconv.ParseUint(flag.Arg(1), 10, 64)
	checkErr("parsing starting position for audit request", err)

	limit, err := strconv.ParseUint(flag.Arg(2), 10, 64)
	checkErr("parsing entry limit for audit request", err)

	res, err := client.Audit(context.Background(), &pb.AuditRequest{Start: start, Limit: limit})
	checkErr("audit request", err)

	p.Println("Updates:")
	for _, update := range res.Updates {
		p.Printf("  - real=%-5v index=%-64x seed=%x commitment=%x\n", update.Real, update.Index, update.Seed, update.Commitment)
	}
	if len(res.Updates) == 0 {
		p.Println("  (None)")
	}
	p.Println()
	if res.More {
		p.Printf("More available, starting at: %v\n", start+limit)
	} else {
		p.Println("Reached end of log.")
	}
}
