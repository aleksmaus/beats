// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pub

import (
	"sync"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/endpointbeat/internal/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

type Publisher struct {
	b   *beat.Beat
	log *logp.Logger

	mx sync.Mutex
}

func New(b *beat.Beat, log *logp.Logger) *Publisher {
	return &Publisher{
		b:   b,
		log: log,
	}
}

func (p *Publisher) Configure(inputs []config.InputConfig) error {
	if len(inputs) == 0 {
		return nil
	}

	p.mx.Lock()
	defer p.mx.Unlock()

	// TODO: configure publisher
	return nil
}

func (p *Publisher) Close() {
	p.mx.Lock()
	defer p.mx.Unlock()

	// TODO: close publishers
}
