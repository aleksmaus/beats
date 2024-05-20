// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	"github.com/elastic/elastic-agent-libs/mapstr"

	cmd "github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/ecs"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/publisher/processing"
	"github.com/elastic/beats/v7/x-pack/endpointbeat/beater"
	_ "github.com/elastic/beats/v7/x-pack/endpointbeat/include"
	_ "github.com/elastic/beats/v7/x-pack/libbeat/include"
	"github.com/elastic/beats/v7/x-pack/libbeat/management"
)

// Name of this beat
const (
	Name = "endpointbeat"
)

// withECSVersion is a modifier that adds ecs.version to events.
var withECSVersion = processing.WithFields(mapstr.M{
	"ecs": mapstr.M{
		"version": ecs.Version,
	},
})

var RootCmd = Endpointbeat()

func Endpointbeat() *cmd.BeatsRootCmd {
	globalProcs, err := processors.NewPluginConfigFromList(defaultProcessors())
	if err != nil { // these are hard-coded, shouldn't fail
		panic(fmt.Errorf("error creating global processors: %w", err))
	}
	settings := instance.Settings{
		Name:            Name,
		Processing:      processing.MakeDefaultSupport(true, globalProcs, withECSVersion, processing.WithHost, processing.WithAgentMeta()),
		ElasticLicensed: true,
	}
	command := cmd.GenRootCmdWithSettings(beater.New, settings)
	command.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		management.ConfigTransform.SetTransform(endpointbeatCfg)
	}

	return command
}

func endpointbeatCfg(rawIn *proto.UnitExpectedConfig, agentInfo *client.AgentInfo) ([]*reload.ConfigWithMeta, error) {
	return endpointbeatCfgNoStreams(rawIn, agentInfo)
}

// This is needed for compatibility with the legacy implementation where kibana set empty streams array [] into the policy
func endpointbeatCfgNoStreams(rawIn *proto.UnitExpectedConfig, agentInfo *client.AgentInfo) ([]*reload.ConfigWithMeta, error) {
	modules, err := management.CreateInputsFromStreams(rawIn, "endpoint", agentInfo)
	if err != nil {
		return nil, fmt.Errorf("error creating input list from raw expected config: %w", err)
	}
	for iter := range modules {
		modules[iter]["type"] = "log"
	}

	// format for the reloadable list needed by the cm.Reload() method
	configList, err := management.CreateReloadConfigFromInputs(modules)
	if err != nil {
		return nil, fmt.Errorf("error creating config for reloader: %w", err)
	}
	return configList, nil
}

func defaultProcessors() []mapstr.M {
	// 	processors:
	//   - add_host_metadata: ~
	//   - add_cloud_metadata: ~
	return []mapstr.M{
		{"add_host_metadata": nil},
		{"add_cloud_metadata": nil},
	}
}
