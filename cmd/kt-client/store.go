//
// Copyright 2025 Signal Messenger, LLC
// SPDX-License-Identifier: AGPL-3.0-only
//

package main

import (
	"github.com/signalapp/keytransparency/cmd/internal/config"
	"github.com/signalapp/keytransparency/db"
	"github.com/signalapp/keytransparency/tree/transparency"
)

// clientStorage is a no-op implementation of the ClientStorage interface.
type clientStorage struct {
	config *transparency.PublicConfig
	head   *db.TransparencyTreeHead
	root   []byte
}

func newStore() *clientStorage {
	config, err := config.Read(*configFile)
	checkErr("loading config", err)

	return &clientStorage{config: config.APIConfig.TreeConfig().Public()}
}

func (cs *clientStorage) PublicConfig() *transparency.PublicConfig { return cs.config }

func (cs *clientStorage) GetLastTreeHead() (*db.TransparencyTreeHead, []byte, error) {
	return cs.head, cs.root, nil
}

func (cs *clientStorage) SetLastTreeHead(head *db.TransparencyTreeHead, root []byte) error {
	cs.head, cs.root = head, root
	return nil
}

func (cs *clientStorage) GetData(key []byte) (*transparency.MonitoringData, error) {
	return nil, nil
}

func (cs *clientStorage) SetData(key []byte, data *transparency.MonitoringData) error {
	return nil
}
