// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/endpointbeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/endpointbeat/internal/pub"
	conf "github.com/elastic/elastic-agent-libs/config"
)

var (
	ErrInvalidQueryConfig = errors.New("invalid query configuration")
	ErrAlreadyRunning     = errors.New("already running")
	ErrQueryExecution     = errors.New("failed query execution")
	ErrActionRequest      = errors.New("invalid action request")
)

type endpointbeat struct {
	b      *beat.Beat
	config config.Config

	pub *pub.Publisher

	log *logp.Logger

	// Beat lifecycle context, cancelled on Stop
	cancel context.CancelFunc
	mx     sync.Mutex
}

// New creates an instance of endpointbeat.
func New(b *beat.Beat, cfg *conf.C) (beat.Beater, error) {
	log := logp.NewLogger("endpointbeat")

	c := config.DefaultConfig
	if err := cfg.Unpack(&c); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	bt := &endpointbeat{
		b:      b,
		config: c,
		log:    log,
		pub:    pub.New(b, log),
	}

	return bt, nil
}

func (bt *endpointbeat) init() (context.Context, error) {
	bt.mx.Lock()
	defer bt.mx.Unlock()
	if bt.cancel != nil {
		return nil, ErrAlreadyRunning
	}
	var ctx context.Context
	ctx, bt.cancel = context.WithCancel(context.Background())

	return ctx, nil
}

func (bt *endpointbeat) close() {
	bt.mx.Lock()
	defer bt.mx.Unlock()
	if bt.pub != nil {
		bt.pub.Close()
	}
	if bt.cancel != nil {
		bt.cancel()
		bt.cancel = nil
	}
}

// Run starts endpointbeat.
func (bt *endpointbeat) Run(b *beat.Beat) error {
	ctx, err := bt.init()
	if err != nil {
		return err
	}
	defer bt.close()

	// Watch input configuration updates
	inputConfigCh := config.WatchInputs(ctx, bt.log)

	g, ctx := errgroup.WithContext(ctx)

	if err := b.Manager.Start(); err != nil {
		return err
	}
	defer b.Manager.Stop()

	// Run main loop
	g.Go(func() error {
		// Configure publisher from initial input
		err := bt.pub.Configure(bt.config.Inputs)
		if err != nil {
			return err
		}

		for {
			select {
			case <-ctx.Done():
				bt.log.Info("endpointbeat context cancelled, exiting")
				return ctx.Err()
			case inputConfigs := <-inputConfigCh:
				err = bt.pub.Configure(inputConfigs)
				if err != nil {
					bt.log.Errorf("Failed to connect beat publisher client, err: %v", err)
					return err
				}
				// TODO: propage the configuration change to Endpoint library
			}
		}
	})

	// Wait for clean exit
	err = g.Wait()
	if err != nil {
		if errors.Is(err, context.Canceled) {
			bt.log.Debugf("endpointbeat Run exited, context cancelled")
		} else {
			bt.log.Errorf("endpointbeat Run exited with error: %v", err)
		}
	} else {
		bt.log.Debugf("endpointbeat Run exited")
	}
	return err
}

// Stop stops endpointbeat.
func (bt *endpointbeat) Stop() {
	bt.close()
}
