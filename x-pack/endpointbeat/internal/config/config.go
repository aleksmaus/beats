// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import (
	"github.com/elastic/beats/v7/libbeat/processors"
)

type StreamConfig struct {
	ID       string `config:"id"`
	Query    string `config:"query"`    // the SQL query to run
	Interval int    `config:"interval"` // an interval in seconds to run the query (subject to splay/smoothing). It has a maximum value of 604,800 (1 week).
	Platform string `config:"platform"` // restrict this query to a given platform, default is 'all' platforms; you may use commas to set multiple platforms
}

type DatastreamConfig struct {
	Namespace string `config:"namespace"`
	Dataset   string `config:"dataset"`
	Type      string `config:"type"`
}

type InputConfig struct {
	Name       string                  `config:"name"`
	Type       string                  `config:"type"`
	Datastream DatastreamConfig        `config:"data_stream"` // Datastream configuration
	Processors processors.PluginConfig `config:"processors"`

	// Full Endpoint configuration
	// Endpoint *EndpointConfig `config:"endpoint"`
}

type Config struct {
	Inputs []InputConfig `config:"inputs"`
}

var DefaultConfig = Config{}
