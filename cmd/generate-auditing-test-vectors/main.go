//
// Copyright 2025 Signal Messenger, LLC
// SPDX-License-Identifier: AGPL-3.0-only
//

//go:generate protoc -I ./pb -I ../../tree/transparency/pb --go_out=pb --go_opt=paths=source_relative --go-grpc_out=pb --go-grpc_opt=paths=source_relative vectors.proto

// Command generate-auditing-test-vectors outputs a file in the current working
// directory that contains test vectors for an auditor implementation.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"fmt"
	"log"
	"os"

	"google.golang.org/protobuf/proto"

	"github.com/signalapp/keytransparency/cmd/generate-auditing-test-vectors/pb"
	"github.com/signalapp/keytransparency/tree/transparency"
	tpb "github.com/signalapp/keytransparency/tree/transparency/pb"
	transparency_test "github.com/signalapp/keytransparency/tree/transparency/test"
)

func random() []byte {
	out := make([]byte, 16)
	if _, err := rand.Read(out); err != nil {
		panic(err)
	}
	return out
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var output []*pb.TestVectors_ShouldFailTestVector

	// first proof type must be newTree
	{
		tree, _, _, _ := transparency_test.NewTree(nil, transparency.ContactMonitoring)

		tree.UpdateSimple(&tpb.UpdateRequest{SearchKey: random(), Value: random()})
		tree.UpdateSimple(&tpb.UpdateRequest{SearchKey: random(), Value: random()})

		updates, _, _ := tree.Audit(0, 1000)
		updates = updates[1:]

		output = append(output, &pb.TestVectors_ShouldFailTestVector{
			Description: "first proof type must be newTree",
			Updates:     updates,
		})
	}

	// first proof type must be a real update
	{
		tree, _, _, _ := transparency_test.NewTree(nil, transparency.ContactMonitoring)

		tree.UpdateSimple(&tpb.UpdateRequest{SearchKey: random(), Value: random()})

		updates, _, _ := tree.Audit(0, 1000)
		updates[0].Real = false

		output = append(output, &pb.TestVectors_ShouldFailTestVector{
			Description: "first proof type must be real update",
			Updates:     updates,
		})
	}

	// newTree proof cannot be given for a non-empty tree
	{
		tree, _, _, _ := transparency_test.NewTree(nil, transparency.ContactMonitoring)

		tree.UpdateSimple(&tpb.UpdateRequest{SearchKey: random(), Value: random()})
		tree.UpdateSimple(&tpb.UpdateRequest{SearchKey: random(), Value: random()})

		updates, _, _ := tree.Audit(0, 1000)
		updates[1] = updates[0]

		output = append(output, &pb.TestVectors_ShouldFailTestVector{
			Description: "newTree proof cannot be given for a non-empty tree",
			Updates:     updates,
		})
	}

	// differentKey must match old root
	{
		tree, _, _, _ := transparency_test.NewTree(nil, transparency.ContactMonitoring)

		tree.UpdateSimple(&tpb.UpdateRequest{SearchKey: random(), Value: random()})
		tree.UpdateSimple(&tpb.UpdateRequest{SearchKey: random(), Value: random()})
		tree.UpdateSimple(&tpb.UpdateRequest{SearchKey: random(), Value: random()})

		updates, _, _ := tree.Audit(0, 1000)
		updates[2].Proof.Proof.(*tpb.AuditorProof_DifferentKey_).DifferentKey.OldSeed[0] ^= 1

		output = append(output, &pb.TestVectors_ShouldFailTestVector{
			Description: "differentKey must match old root",
			Updates:     updates,
		})
	}

	// sameKey must match old root
	{
		tree, _, _, _ := transparency_test.NewTree(nil, transparency.ContactMonitoring)

		tree.UpdateSimple(&tpb.UpdateRequest{SearchKey: random(), Value: random()})
		newKey := random()
		tree.UpdateSimple(&tpb.UpdateRequest{SearchKey: newKey, Value: random()})
		tree.UpdateSimple(&tpb.UpdateRequest{SearchKey: newKey, Value: random()})

		updates, _, _ := tree.Audit(0, 1000)
		updates[2].Proof.Proof.(*tpb.AuditorProof_SameKey_).SameKey.Position += 1

		output = append(output, &pb.TestVectors_ShouldFailTestVector{
			Description: "sameKey must match old root",
			Updates:     updates,
		})
	}

	// proof may not be sameKey if update type is fake
	{
		tree, _, _, _ := transparency_test.NewTree(nil, transparency.ContactMonitoring)

		tree.UpdateSimple(&tpb.UpdateRequest{SearchKey: random(), Value: random()})
		newKey := random()
		tree.UpdateSimple(&tpb.UpdateRequest{SearchKey: newKey, Value: random()})
		tree.UpdateSimple(&tpb.UpdateRequest{SearchKey: newKey, Value: random()})

		updates, _, _ := tree.Audit(0, 1000)
		updates[2].Real = false

		output = append(output, &pb.TestVectors_ShouldFailTestVector{
			Description: "proof may not be sameKey if update type is fake",
			Updates:     updates,
		})
	}

	// happy path
	var successVector *pb.TestVectors_ShouldSucceedTestVector
	{
		tree, store, _, _ := transparency_test.NewTree(nil, transparency.ContactMonitoring)
		transparency_test.RandomTree(tree, store, 10, []int{3}, []int{4, 9})
		updates, _, _ := tree.Audit(0, 1000)

		vector := &pb.TestVectors_ShouldSucceedTestVector{}
		for i, update := range updates {
			root, err := tree.GetLogTree().GetRoot(uint64(i + 1))
			if err != nil {
				log.Fatal(err)
			}
			vector.Updates = append(vector.Updates, &pb.TestVectors_ShouldSucceedTestVector_UpdateAndHash{
				Update:  update,
				LogRoot: root,
			})
		}
		successVector = vector
	}

	// signature
	var signatureVector *pb.TestVectors_SignatureTestVector
	{
		auditorName := "example-auditor"
		_, _, config, _ := transparency_test.NewTree(nil, transparency.ThirdPartyAuditing)
		auditorPub, auditorPriv, err := ed25519.GenerateKey(nil)
		if err != nil {
			log.Fatal(err)
		}
		config.AuditorKeys = map[string]ed25519.PublicKey{
			auditorName: auditorPub,
		}

		head, signatureInput, _ := transparency.SignNewAuditorHead(auditorPriv, config.Public(), 1337, make([]byte, 32), auditorName)

		// Encode auditor public and private key for Java.
		auditorPub, err = x509.MarshalPKIXPublicKey(auditorPub)
		if err != nil {
			log.Fatal(err)
		}
		auditorPriv, err = x509.MarshalPKCS8PrivateKey(auditorPriv)
		if err != nil {
			log.Fatal(err)
		}

		sigKey, err := x509.MarshalPKIXPublicKey(config.SigKey.Public())
		if err != nil {
			log.Fatal(err)
		}

		vrfKey, err := x509.MarshalPKIXPublicKey(ed25519.PublicKey(config.VrfKey.Public().([]byte)))
		if err != nil {
			log.Fatal(err)
		}

		signatureVector = &pb.TestVectors_SignatureTestVector{
			AuditorPrivKey: auditorPriv,

			DeploymentMode: uint32(config.Mode),
			SigPubKey:      sigKey,
			AuditorPubKey:  auditorPub,
			VrfPubKey:      vrfKey,

			TreeSize:  1337,
			Timestamp: head.Timestamp,
			Root:      make([]byte, 32),

			Signature:      head.Signature,
			SignatureInput: signatureInput,
		}
	}

	fmtProof := func(proof *tpb.AuditorProof) string {
		switch p := proof.Proof.(type) {
		case *tpb.AuditorProof_NewTree_:
			return "newTree{}"
		case *tpb.AuditorProof_DifferentKey_:
			return fmt.Sprintf("differentKey{copath: %v, old_seed: %x}", len(p.DifferentKey.Copath), p.DifferentKey.OldSeed)
		case *tpb.AuditorProof_SameKey_:
			return fmt.Sprintf("sameKey{copath: %v, ctr: %v, pos: %v}", len(p.SameKey.Copath), p.SameKey.Counter, p.SameKey.Position)
		default:
			panic("unreachable")
		}
	}
	for _, updates := range output {
		for _, update := range updates.Updates {
			fmt.Printf("real=%v index=%x seed=%x commitment = %x proof = %v\n", update.Real, update.Index, update.Seed, update.Commitment, fmtProof(update.Proof))
		}
		fmt.Println("----")
	}

	raw, err := proto.Marshal(&pb.TestVectors{
		ShouldFail:    output,
		ShouldSucceed: successVector,
		Signature:     signatureVector,
	})
	if err != nil {
		log.Fatal(err)
	} else if err := os.WriteFile("./kt_test_vectors.pb", raw, 0777); err != nil {
		log.Fatal(err)
	}
}
