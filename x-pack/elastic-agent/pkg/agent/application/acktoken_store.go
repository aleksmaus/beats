// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"io"
	"sync"

	yaml "gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

var (
	atsCached AckTokenSerializer
	mx        sync.RWMutex
)

// Quick&dirty storage for ack token for the agent action prototype
type ackTokenStore struct {
	log   *logger.Logger
	store storeLoad
}

func newAckTokenStore(log *logger.Logger, store storeLoad) (*ackTokenStore, error) {
	reader, err := store.Load()
	if err != nil {
		return &ackTokenStore{log: log, store: store}, nil
	}
	defer reader.Close()

	var ats AckTokenSerializer

	dec := yaml.NewDecoder(reader)
	err = dec.Decode(&ats)
	if err == io.EOF {
		return &ackTokenStore{
			log:   log,
			store: store,
		}, nil
	}
	if err != nil {
		return nil, err
	}

	mx.Lock()
	atsCached = ats
	mx.Unlock()

	return &ackTokenStore{
		log:   log,
		store: store,
	}, nil
}

func (s *ackTokenStore) Save(ackToken string) error {
	var reader io.Reader
	var ats AckTokenSerializer
	ats.AckToken = ackToken
	reader, err := yamlToReader(&ats)
	if err != nil {
		return err
	}

	mx.Lock()
	atsCached = ats
	mx.Unlock()

	if err := s.store.Save(reader); err != nil {
		return err
	}
	return nil
}

func (s *ackTokenStore) GetToken() string {
	mx.RLock()
	ats := atsCached
	mx.RUnlock()
	return ats.AckToken
}

type AckTokenSerializer struct {
	AckToken string `yaml:"ack_token"`
}
