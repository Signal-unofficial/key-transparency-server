//
// Copyright 2025 Signal Messenger, LLC
// SPDX-License-Identifier: AGPL-3.0-only
//

// Command kt-stress is a tool for stress testing a key transparency server.
package main

import (
	"context"
	"crypto/rand"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/text/message"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/signalapp/keytransparency/cmd/kt-server/pb"
	tpb "github.com/signalapp/keytransparency/tree/transparency/pb"
)

var (
	p = message.NewPrinter(message.MatchLanguage("en"))

	serverAddr = flag.String("addr", "localhost:8080", "Address of test server.")
	threads    = flag.Int("threads", 1, "Number of threads to use.")
)

var (
	TotalTime    int64
	TotalUpdates int64
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.LUTC)
	flag.Parse()

	start := time.Now()
	for i := 0; i < *threads; i++ {
		go stress()
	}

	p.Println("Testing started. Press Ctrl^C to stop...")
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	select {
	case <-ch:
	case <-time.After(10 * time.Second):
	}

	dur := time.Since(start)
	latency := atomic.LoadInt64(&TotalTime)
	updates := atomic.LoadInt64(&TotalUpdates)

	p.Println()
	p.Println("Report:")
	p.Printf("  Duration: %v\n", dur.Round(time.Second))
	p.Printf("  Threads: %v\n", *threads)
	p.Printf("  Total updates: %d\n", updates)
	p.Printf("    Throughput: %.0f op/s\n", float64(updates)/dur.Seconds())
	p.Printf("    Latency: %.0f us/op\n", float64(latency)/float64(updates))
}

func random() []byte {
	out := make([]byte, 16)
	if _, err := rand.Read(out); err != nil {
		panic(err)
	}
	return out
}

func stress() {
	conn, err := grpc.Dial(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := pb.NewKeyTransparencyTestServiceClient(conn)

	for {
		start := time.Now()
		req := &tpb.UpdateRequest{SearchKey: random(), Value: random()}
		_, err := client.Update(context.Background(), req)
		if err != nil {
			log.Fatalf("Error executing request: %v", err)
		}
		atomic.AddInt64(&TotalTime, time.Since(start).Microseconds())
		atomic.AddInt64(&TotalUpdates, 1)
	}
}
