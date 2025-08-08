//
// Copyright 2025 Signal Messenger, LLC
// SPDX-License-Identifier: AGPL-3.0-only
//

// Command kt-client is a test/example client used for interacting with a
// key transparency server.
package main

import (
	"context"
	"flag"
	"log"
	"time"

	"golang.org/x/text/message"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/signalapp/keytransparency/cmd/internal/config"
	"github.com/signalapp/keytransparency/cmd/kt-server/pb"
	tpb "github.com/signalapp/keytransparency/tree/transparency/pb"
)

var (
	p = message.NewPrinter(message.MatchLanguage("en"))

	ktQueryServerAddr = flag.String("query-addr", "localhost:8080", "Address of read-only server.")
	ktServerAddr      = flag.String("kt-addr", "localhost:8082", "Address of read-write server.")
	testServerAddr    = flag.String("test-addr", "localhost:8081", "Address of test server.")
	configFile        = flag.String("config", "", "(Optional) Location of server config file.")

	usernameHash     = flag.String("username-hash", "", "Base64url encoded username hash")
	e164             = flag.String("e164", "", "E164-formatted phone number. Must be preceded with a '+'. E.g. +14155550101")
	uak              = flag.String("uak", "", "Standard base64 encoded unidentified access key")
	timingNumSamples = flag.Int("num-samples", 5, "Number of samples to use for measuring timing of a query request")
	timingSampleSize = flag.Int("sample-size", 100, "Number of requests per sample to use for measuring timing of a query request")

	last = flag.Int("last", -1, "(Optional) Size of tree when last observed, or -1 for none.")
)

func consistency(x *int) *tpb.Consistency {
	if *x == -1 {
		return &tpb.Consistency{}
	} else if *x < -1 {
		log.Fatal("Flag value may not be less than -1.")
	}
	y := uint64(*x)
	return &tpb.Consistency{Last: &y}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.LUTC)
	flag.Parse()

	configs, err := readConfig(*configFile)
	checkErr("parsing config", err)

	ktQueryConn, err := grpc.NewClient(*ktQueryServerAddr, getClientOptions(configs.KtQueryServiceConfig)...)
	checkErr("instantiating kt query client", err)

	defer ktQueryConn.Close()
	ktQueryClient := pb.NewKeyTransparencyQueryServiceClient(ktQueryConn)

	testConn, err := grpc.NewClient(*testServerAddr, getClientOptions(configs.KtTestServiceConfig)...)
	checkErr("instantiating kt test client", err)

	defer testConn.Close()
	testClient := pb.NewKeyTransparencyTestServiceClient(testConn)

	ktConn, err := grpc.NewClient(*ktServerAddr, getClientOptions(configs.KtServiceConfig)...)
	checkErr("instantiating kt client", err)

	defer ktConn.Close()

	ktAuditorClient := pb.NewKeyTransparencyAuditorServiceClient(ktConn)

	switch flag.Arg(0) {
	case "distinguished":
		handleDistinguished(ktQueryClient)
	case "search":
		handleSearch(ktQueryClient)
	case "search-timing":
		handleSearchTiming(ktQueryClient)
	case "update":
		handleUpdate(testClient)
	case "monitor":
		handleMonitor(ktQueryClient)
	case "monitor-timing":
		handleMonitorTiming(ktQueryClient)
	case "audit":
		handleAudit(ktAuditorClient)
	case "tree-size":
		res, err := ktAuditorClient.TreeSize(context.Background(), &emptypb.Empty{})
		checkErr("tree size request", err)
		p.Println("Tree Size: ", res.TreeSize)
	default:
		log.Fatal("Unexpected operation requested. Allowed arguments: \n- distinguished" +
			"\n- search\n- search-timing\n- update\n- monitor\n- monitor-timing\n- audit\n- tree-size")
	}
}

func readConfig(configFile string) (*config.Config, error) {
	if configFile == "" {
		return &config.Config{}, nil
	}

	cfg, err := config.Read(configFile)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func getClientOptions(config *config.ServiceConfig) []grpc.DialOption {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	if config == nil || len(config.AuthorizedHeaders) == 0 {
		return opts
	}

	opts = append(opts,
		grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			headers := []string{}
			for header, values := range config.AuthorizedHeaders {
				for _, value := range values {
					headers = append(append(headers, header), value)
				}
			}
			ctx = metadata.AppendToOutgoingContext(ctx, headers...)
			return invoker(ctx, method, req, reply, cc, opts...)
		}))

	return opts
}

func round(d time.Duration) time.Duration {
	if d > time.Second {
		return d.Round(time.Second)
	}
	return d.Round(time.Millisecond)
}

func printFullTreeHead(fth *tpb.FullTreeHead) {
	p.Printf("Full Tree Head:\n")
	p.Printf("  Tree Head:\n")
	p.Printf("    Tree Size: %v\n", fth.TreeHead.TreeSize)
	ts := time.UnixMilli(fth.TreeHead.Timestamp)
	p.Printf("    Timestamp: %v (%v ago)\n", ts, round(time.Now().Sub(ts)))
	p.Printf("    Signature: %x\n", fth.TreeHead.Signatures)

	if n := len(fth.Last); n > 0 {
		p.Printf("  Consistency Proof: Given (%v entries)\n", n)
	} else {
		p.Printf("  Consistency Proof: None\n")
	}
	p.Println()
}

func printSearchProof(proof *tpb.SearchProof) {
	p.Printf("Search Proof:\n")
	p.Printf("  Pos: %v\n", proof.Pos)
	p.Printf("  Inclusion Proof: %v entries\n", len(proof.Inclusion))
	for _, step := range proof.Steps {
		p.Printf("  - counter=%v commitment=%x\n", step.Prefix.Counter, step.Commitment)
	}
	p.Println()
}
