//
// Copyright 2025 Signal Messenger, LLC
// SPDX-License-Identifier: AGPL-3.0-only
//

package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	kinesistypes "github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	consumer "github.com/harlow/kinesis-consumer"
	metrics "github.com/hashicorp/go-metrics"
	"golang.org/x/sync/errgroup"

	"github.com/signalapp/keytransparency/cmd/internal/config"
	"github.com/signalapp/keytransparency/cmd/internal/util"
	"github.com/signalapp/keytransparency/db"
	"github.com/signalapp/keytransparency/tree/transparency/pb"
)

const (
	backfillScanShards = 1000
	backfillWorkers    = 96
	withinBackfill     = "backfill"
	withinStream       = "stream"
	tombstoneString    = "tombstone"
)

var tombstoneBytes = marshalValue([]byte(tombstoneString))
var logUpdater = &LogUpdater{}

// metricsCounter implements the consumer.Counter interface for exporting
// Kinesis metrics.
type metricsCounter struct{}

func (pc metricsCounter) Add(name string, val int64) {
	metrics.IncrCounterWithLabels([]string{withinStream, "kinesis"}, float32(val), []metrics.Label{{Name: "type", Value: name}})
}

// kinesisLogger implements the consumer.Logger interface for printing Kinesis
// logs to stdout.
type kinesisLogger struct{}

func (kl kinesisLogger) Log(v ...any) { util.Log().Infof("%s", fmt.Sprintln(v...)) }

// shardState is used to keep track of the scan state of each shard.
type shardState struct {
	// sinceLast is the number of entries from this shard that have been
	// sequenced since its last checkpoint.
	sinceLast int
	// wg is a WaitGroup that all goroutines sequencing entries from this shard
	// are given.
	wg *sync.WaitGroup
}

type Streamer struct {
	config *config.APIConfig
	tx     db.TransparencyStore
}

// run runs the streamer, blocking forever.
func (s *Streamer) run(ctx context.Context, name string, startAtTimestamp time.Time, updateHandler *KtUpdateHandler) {
	i := 0
	for {
		var updatesWg sync.WaitGroup
		var shardsMu sync.Mutex

		// Reset the shards map on consumer restart.
		shards := make(map[string]*shardState)

		// Create a new context for each run.
		runCtx, cancel := context.WithCancel(ctx)

		// Note on thread safety: The Kinesis consumer library will use one
		// goroutine per shard to scan. As such, a mutex is required to lookup shard
		// state from the `shards` map because many shards may be read/written to
		// the map in parallel. But the returned shardState struct can then used
		// without a mutex because there is only one goroutine working with it.

		c, err := consumer.New(
			name,
			consumer.WithLogger(kinesisLogger{}),
			consumer.WithCounter(metricsCounter{}),
			consumer.WithStore(s.tx.StreamStore()),
			consumer.WithShardIteratorType(string(kinesistypes.ShardIteratorTypeAtTimestamp)),
			consumer.WithTimestamp(startAtTimestamp),
		)
		if err != nil {
			util.Log().Errorf("stream consumer initialization error: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}
		err = c.Scan(runCtx, func(r *consumer.Record) error {
			// Get the existing shardState struct or initialize a new one if needed.
			shardsMu.Lock()
			if _, ok := shards[r.ShardID]; !ok {
				shards[r.ShardID] = &shardState{0, &sync.WaitGroup{}}
			}
			state := shards[r.ShardID]
			shardsMu.Unlock()

			// Start a dedicated goroutine to process this update. We don't call
			// wg.Done until the update is successfully executed, retrying
			// infinitely if necessary. This is required to prevent us from
			// checkpointing past an update that we might've failed to sequence.
			state.sinceLast += 1
			state.wg.Add(1)
			go func(ctx context.Context, data []byte, wg *sync.WaitGroup) {
				updatesWg.Add(1)
				defer updatesWg.Done()
				for {
					select {
					case <-ctx.Done():
						return
					default:
						err := updateFromStream(ctx, data, updateHandler, logUpdater)
						if err != nil {
							util.Log().Infof("failed to update entry from stream: %v", err)
							metrics.IncrCounter([]string{withinStream, "errors"}, 1)
							time.Sleep(3 * time.Second)
						} else {
							wg.Done()
							return
						}
					}
				}
			}(runCtx, dup(r.Data), state.wg)

			// If only a few entries have been sequenced from this shard, move on.
			if state.sinceLast < 100 {
				return consumer.ErrSkipCheckpoint
			}
			// If many entries have been sequenced, we need to checkpoint. First
			// wait for all processing updates to complete.
			state.wg.Wait()
			state.sinceLast = 0
			return nil
		})
		util.Log().Errorf("stream consumer error: %v", err)

		// We only reach this point if c.Scan returns an error.
		// In this case, clean up the current context, sleep with an exponential backoff,
		// and wait for all spawned goroutines to exit.
		cancel()

		// Cap the backoff at 60 seconds
		delay := time.Duration(math.Min(60, math.Pow(2, float64(i)))) * time.Second
		util.Log().Infof("iteration %d of stream consumer, sleeping %s", i, delay)
		time.Sleep(delay)

		// Ensure that all update goroutines have exited before restarting the consumer
		updatesWg.Wait()
		i++
	}
}

func backfill(ctx context.Context, table string, updateHandler *KtUpdateHandler) error {
	eg, ctx := errgroup.WithContext(ctx)
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRetryer(func() aws.Retryer {
		// Max attempts set to 0 indicates that the attempt should be retried until it succeeds
		// https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/aws/retry#AdaptiveMode.MaxAttempts
		return retry.AddWithMaxAttempts(retry.NewAdaptiveMode(), 0)
	}))
	if err != nil {
		return fmt.Errorf("loading aws sdk config: %w", err)
	}
	ddb := dynamodb.NewFromConfig(cfg)
	totalShards := int32(backfillScanShards)
	eg.SetLimit(backfillWorkers)

	for shard := int32(0); shard < backfillScanShards; shard++ {
		shard := shard
		eg.Go(func() (returnedErr error) {
			util.Log().Infof("Starting processing of backfill shard %d", shard)
			defer func() {
				metrics.IncrCounter([]string{withinBackfill, "shards_processed"}, 1)
				util.Log().Infof("Finished processing of backfill shard %d: err=%v", shard, returnedErr)
			}()
			var exclusiveStartKey map[string]types.AttributeValue
			for i := 0; ; i++ {
				if err := ctx.Err(); err != nil {
					return err
				}
				out, err := ddb.Scan(ctx, &dynamodb.ScanInput{
					TableName:         &table,
					Segment:           &shard,
					TotalSegments:     &totalShards,
					ExclusiveStartKey: exclusiveStartKey,
				})
				if err != nil {
					return fmt.Errorf("scan %d of shard %d failed: %w", i, shard, err)
				} else if err := backfillScanOutput(ctx, out, updateHandler, logUpdater); err != nil {
					return fmt.Errorf("scan %d of shard %d backfill processing failed: %w", i, shard, err)
				} else if exclusiveStartKey = out.LastEvaluatedKey; len(exclusiveStartKey) == 0 {
					return nil
				}
			}
		})
	}
	return eg.Wait()
}

func backfillScanOutput(ctx context.Context, scan *dynamodb.ScanOutput, updateHandler *KtUpdateHandler, updater Updater) error {
	for i, item := range scan.Items {
		if err := updateFromBackfill(ctx, item, updateHandler, updater); err != nil {
			return fmt.Errorf("processing item %d: %w", i, err)
		}
	}
	return nil
}

type accountPair struct {
	Prev *account `json:"prev"`
	Next *account `json:"next"`
}

type account struct {
	ACI            []byte `json:"aci"`
	ACIIdentityKey []byte `json:"aciIdentityKey"`

	Number       string `json:"number"`
	UsernameHash []byte `json:"usernameHash"`
}

func updateFromBackfill(ctx context.Context, item map[string]types.AttributeValue, updateHandler *KtUpdateHandler, updater Updater) error {
	type backfillAccount struct {
		Number         string `json:"number"`
		ACIIdentityKey []byte `json:"identityKey"`
		UsernameHash   string `json:"usernameHash"` // URL-encoded base64
	}
	u := item["U"]
	if u == nil {
		return fmt.Errorf("no account ID")
	}
	ub, ok := u.(*types.AttributeValueMemberB)
	if !ok {
		return fmt.Errorf("account ID not bytes")
	} else if len(ub.Value) != 16 {
		return fmt.Errorf("account ID not valid")
	}
	accountID := ub.Value
	d := item["D"]
	if d == nil {
		return fmt.Errorf("account %x no data", accountID)
	}
	db, ok := d.(*types.AttributeValueMemberB)
	if !ok {
		return fmt.Errorf("account %x data not bytes", accountID)
	}
	var account backfillAccount
	if err := json.Unmarshal(db.Value, &account); err != nil {
		return fmt.Errorf("parsing account %x data: %w", accountID, err)
	} else if len(account.Number) == 0 {
		return fmt.Errorf("account %x data has empty number", accountID)
	}
	if len(account.ACIIdentityKey) > 0 {
		if err := updater.update(ctx, withinBackfill,
			append([]byte{util.AciPrefix}, accountID...),
			marshalValue(account.ACIIdentityKey), updateHandler, nil); err != nil {
			return fmt.Errorf("updating %x ACI: %w", accountID, err)
		}
	}
	if len(account.Number) > 0 {
		if err := updater.update(ctx, withinBackfill,
			append([]byte{util.NumberPrefix}, []byte(account.Number)...),
			marshalValue(accountID), updateHandler, nil); err != nil {
			return fmt.Errorf("updating %x Number: %w", accountID, err)
		}
	}
	if len(account.UsernameHash) > 0 {
		usernameHash, err := base64.RawURLEncoding.DecodeString(account.UsernameHash)
		if err != nil {
			return fmt.Errorf("updating %x username hash: failed to base64 decode hash: %w", accountID, err)
		}
		if err := updater.update(ctx, withinBackfill,
			append([]byte{util.UsernameHashPrefix}, usernameHash...),
			marshalValue(accountID), updateHandler, nil); err != nil {
			return fmt.Errorf("updating %x username hash: %w", accountID, err)
		}
	}
	return nil
}

func updateFromStream(ctx context.Context, data []byte, updateHandler *KtUpdateHandler, updater Updater) error {
	pair := &accountPair{}
	if err := json.Unmarshal(data, pair); err != nil {
		// Note: This is not a temporary error and will intentionally cause the
		// scanner to get stuck until new code is deployed that can handle
		// whatever is in the stream.
		return fmt.Errorf("unmarshaling from stream: %w", err)
	} else if pair.Prev == nil && pair.Next == nil {
		// This should never happen, but we want to know about it if it does
		metrics.IncrCounter([]string{"stream_empty_pair"}, 1)
		return nil
	} else if pair.Prev == nil {
		// New registration. ACI and number should always be present on these updates.
		if err := updater.update(ctx, withinStream,
			append([]byte{util.AciPrefix}, pair.Next.ACI...),
			marshalValue(pair.Next.ACIIdentityKey), updateHandler, nil); err != nil {
			return fmt.Errorf("updating ACI: %w", err)
		}

		if err := updater.update(ctx, withinStream,
			append([]byte{util.NumberPrefix}, []byte(pair.Next.Number)...),
			marshalValue(pair.Next.ACI), updateHandler, nil); err != nil {
			return fmt.Errorf("updating number: %w", err)
		}

		if len(pair.Next.UsernameHash) > 0 {
			if err := updater.update(ctx, withinStream,
				append([]byte{util.UsernameHashPrefix}, pair.Next.UsernameHash...),
				marshalValue(pair.Next.ACI), updateHandler, nil); err != nil {
				return fmt.Errorf("updating username hash: %w", err)
			}
		}
	} else if pair.Next == nil {
		// Account deletion. Overwrite all associated mappings to point to a tombstone value.
		if err := updater.update(ctx, withinStream,
			append([]byte{util.AciPrefix}, pair.Prev.ACI...),
			tombstoneBytes, updateHandler, marshalValue(pair.Prev.ACIIdentityKey)); err != nil {
			return fmt.Errorf("updating ACI: %w", err)
		}

		if err := updater.update(ctx, withinStream,
			append([]byte{util.NumberPrefix}, []byte(pair.Prev.Number)...),
			tombstoneBytes, updateHandler, marshalValue(pair.Prev.ACI)); err != nil {
			return fmt.Errorf("updating number: %w", err)
		}

		if len(pair.Prev.UsernameHash) > 0 {
			if err := updater.update(ctx, withinStream,
				append([]byte{util.UsernameHashPrefix}, pair.Prev.UsernameHash...),
				tombstoneBytes, updateHandler, marshalValue(pair.Prev.ACI)); err != nil {
				return fmt.Errorf("updating username hash: %w", err)
			}
		}
	} else {
		if !bytes.Equal(pair.Prev.ACIIdentityKey, pair.Next.ACIIdentityKey) {
			if err := updater.update(ctx, withinStream,
				append([]byte{util.AciPrefix}, pair.Next.ACI...),
				marshalValue(pair.Next.ACIIdentityKey), updateHandler, nil); err != nil {
				return fmt.Errorf("updating ACI: %w", err)
			}
		}

		if !bytes.Equal(pair.Prev.UsernameHash, pair.Next.UsernameHash) {
			if len(pair.Prev.UsernameHash) > 0 {
				// Tombstone the old username hash
				if err := updater.update(ctx, withinStream,
					append([]byte{util.UsernameHashPrefix}, pair.Prev.UsernameHash...),
					tombstoneBytes, updateHandler, marshalValue(pair.Prev.ACI)); err != nil {
					return fmt.Errorf("updating username hash: %w", err)
				}
			}

			if len(pair.Next.UsernameHash) > 0 {
				if err := updater.update(ctx, withinStream,
					append([]byte{util.UsernameHashPrefix}, pair.Next.UsernameHash...),
					marshalValue(pair.Next.ACI), updateHandler, nil); err != nil {
					return fmt.Errorf("updating username hash: %w", err)
				}
			}
		}

		if pair.Prev.Number != pair.Next.Number {
			if len(pair.Prev.Number) > 0 {
				// Tombstone the old phone number
				if err := updater.update(ctx, withinStream,
					append([]byte{util.NumberPrefix}, pair.Prev.Number...),
					tombstoneBytes, updateHandler, marshalValue(pair.Prev.ACI)); err != nil {
					return fmt.Errorf("updating number: %w", err)
				}
			}

			if len(pair.Next.Number) > 0 {
				if err := updater.update(ctx, withinStream,
					append([]byte{util.NumberPrefix}, []byte(pair.Next.Number)...),
					marshalValue(pair.Next.ACI), updateHandler, nil); err != nil {
					return fmt.Errorf("updating number: %w", err)
				}
			}
		}
	}

	return nil
}

// Updater interface supports mocking in tests
type Updater interface {
	update(ctx context.Context, within string, key, value []byte, handler *KtUpdateHandler, expectedPreUpdateValue []byte) error
}

type LogUpdater struct{}

func (s *LogUpdater) update(ctx context.Context, within string, key, value []byte, updateHandler *KtUpdateHandler, expectedPreUpdateValue []byte) (returnedErr error) {
	defer func() {
		success := returnedErr == nil
		metrics.IncrCounterWithLabels([]string{within, "items_processed"}, 1, []metrics.Label{{Name: "success", Value: strconv.FormatBool(success)}})
	}()
	updateReq := &pb.UpdateRequest{
		SearchKey:   key,
		Value:       value,
		Consistency: &pb.Consistency{}}

	if expectedPreUpdateValue != nil {
		updateReq.ExpectedPreUpdateValue = expectedPreUpdateValue
	}

	_, err := updateHandler.update(ctx, updateReq, 30*time.Minute)
	return err
}

func marshalValue(bytes []byte) []byte {
	// It's not clear to me if we'll want to store more information in the log
	// later. For now, prefix with a 0 as a format version identifier.
	return append([]byte{0}, bytes...)
}

func dup(in []byte) []byte {
	if in == nil {
		return nil
	}
	out := make([]byte, len(in))
	copy(out, in)
	return out
}
