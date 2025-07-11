//
// Copyright 2025 Signal Messenger, LLC
// SPDX-License-Identifier: AGPL-3.0-only
//

package main

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	metrics "github.com/hashicorp/go-metrics"

	"github.com/signalapp/keytransparency/cmd/internal/config"
	"github.com/signalapp/keytransparency/cmd/internal/util"
	"github.com/signalapp/keytransparency/cmd/kt-server/pb"
	"github.com/signalapp/keytransparency/db"
	tpb "github.com/signalapp/keytransparency/tree/transparency/pb"
)

type KtUpdateHandler struct {
	config *config.APIConfig
	tx     db.TransparencyStore
	ch     chan<- updateRequest

	pb.UnimplementedKeyTransparencyTestServiceServer
}

func (h *KtUpdateHandler) Update(ctx context.Context, req *tpb.UpdateRequest) (*tpb.UpdateResponse, error) {
	start := time.Now()
	res, err := h.update(ctx, req, 5*time.Second)
	lbls := []metrics.Label{successLabel(err), grpcStatusLabel(err)}
	metrics.IncrCounterWithLabels([]string{"update_requests"}, 1, lbls)
	metrics.MeasureSinceWithLabels([]string{"update_duration"}, start, lbls)
	if err, _ := status.FromError(err); err.Code() == codes.Unknown {
		util.Log().Errorf("Unexpected update error in key transparency service: %v", err.Err())
	}
	return res, err
}

func (h *KtUpdateHandler) update(ctx context.Context, req *tpb.UpdateRequest, timeout time.Duration) (*tpb.UpdateResponse, error) {
	tree, err := h.config.NewTree(h.tx)
	if err != nil {
		return nil, err
	}
	pre, err := tree.PreUpdate(req)
	if err != nil {
		return nil, err
	}

	ch := make(chan updateResponse, 1)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	select {
	case h.ch <- updateRequest{req: pre, res: ch}:
	case <-ctx.Done():
		return nil, fmt.Errorf("submitting insertion request timed out: %w", ctx.Err())
	}
	select {
	case res := <-ch:
		if res.err != nil {
			return nil, res.err
		} else if res.res == nil {
			// In the case of tombstone updates, it is an expected case to get back
			// no update response and no error.
			return nil, nil
		}
		if req.ReturnUpdateResponse {
			return tree.PostUpdate(res.res)
		}
		return nil, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("waiting for insertion result timed out: %w", ctx.Err())
	}
}
