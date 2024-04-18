// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"encoding/json"
	"fmt"

	cmd "github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/beats/v7/libbeat/common/cli"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/ecs"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/publisher/processing"
	"github.com/elastic/beats/v7/x-pack/libbeat/management"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/spf13/cobra"

	_ "github.com/elastic/beats/v7/x-pack/libbeat/include"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/beater"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/install"
)

// Name of this beat
const (
	Name = "osquerybeat"
)

// withECSVersion is a modifier that adds ecs.version to events.
var withECSVersion = processing.WithFields(mapstr.M{
	"ecs": mapstr.M{
		"version": ecs.Version,
	},
})

var RootCmd = Osquerybeat()

func Osquerybeat() *cmd.BeatsRootCmd {
	management.ConfigTransform.SetTransform(osquerybeatCfg)
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

	// Add verify command
	command.AddCommand(genVerifyCmd(settings))

	return command
}

func genVerifyCmd(_ instance.Settings) *cobra.Command {
	return &cobra.Command{
		Use:   "verify",
		Short: "Verify installation",
		Run: cli.RunWith(
			func(_ *cobra.Command, args []string) error {
				log := logp.NewLogger("osquerybeat")
				err := install.VerifyWithExecutableDirectory(log)
				if err != nil {
					return err
				}
				return nil
			}),
	}
}

func osquerybeatCfg(rawIn *proto.UnitExpectedConfig, agentInfo *client.AgentInfo) ([]*reload.ConfigWithMeta, error) {
	// For the older stack there were no streams, creating one
	if len(rawIn.GetStreams()) == 0 {
		return osquerybeatCfgNoStreams(rawIn, agentInfo)
	}
	return osquerybeatCfgFromStreams(rawIn, agentInfo)
}

func osquerybeatCfgFromStreams(rawIn *proto.UnitExpectedConfig, agentInfo *client.AgentInfo) ([]*reload.ConfigWithMeta, error) {

	b, _ := json.Marshal(rawIn)
	fmt.Println("OSQUERY RAWIN:", string(b))

	streams := make([]*proto.Stream, 0, len(rawIn.Streams))

	// Attach osquery configuration to the osquery_manager.result stream and set it as a first stream
	for _, stream := range rawIn.Streams {
		if stream.DataStream != nil && stream.DataStream.Dataset == config.DefaultDataset {
			if stream.Source == nil {
				// If for any reason the stream source is missing completely, use datastream source as before
				stream.Source = rawIn.Source
			} else {
				// Set osquery configuration value
				fieldsSrc := rawIn.Source.Fields
				fieldsDst := stream.Source.Fields
				var osqVal *structpb.Value
				if fieldsSrc != nil {
					osqVal = fieldsSrc["osquery"]
				}
				if osqVal != nil {
					fieldsDst["osquery"] = osqVal
				}
				// Setting id to the source because it is being picked up from there in shared management.CreateInputsFromStreams
				vId, ok := fieldsDst["id"]
				shouldSet := false
				if !ok || vId == nil {
					shouldSet = true
				} else {
					if _, ok := vId.GetKind().(*structpb.Value_NullValue); ok {
						shouldSet = true
					}
				}
				if shouldSet {
					fieldsDst["id"] = structpb.NewStringValue(rawIn.Id)
				}
			}
			streams = append([]*proto.Stream{stream}, streams...)
			continue
		}
		streams = append(streams, stream)
	}
	rawIn.Streams = streams

	streamList, err := management.CreateInputsFromStreams(rawIn, "logs", agentInfo)
	if err != nil {
		return nil, fmt.Errorf("error creating input list from raw expected config: %w", err)
	}

	var ns string
	if rawIn.DataStream != nil {
		ns = rawIn.DataStream.Namespace
		if ns == "" {
			ns = config.DefaultNamespace
		}
	}

	for iter := range streamList {
		if _, ok := streamList[iter]["type"]; !ok {
			streamList[iter]["type"] = rawIn.Type
		}
		if v, ok := streamList[iter]["data_stream"]; ok {
			if m, ok := v.(map[string]interface{}); ok {
				if _, ok := m["namespace"]; !ok {
					m["namespace"] = ns
				}
			}
		}
	}

	// format for the reloadable list needed bythe cm.Reload() method
	configList, err := management.CreateReloadConfigFromInputs(streamList)
	if err != nil {
		return nil, fmt.Errorf("error creating config for reloader: %w", err)
	}

	return configList, nil
}

// This is needed for compatibility with the legacy implementation where kibana set empty streams array [] into the policy
func osquerybeatCfgNoStreams(rawIn *proto.UnitExpectedConfig, agentInfo *client.AgentInfo) ([]*reload.ConfigWithMeta, error) {
	// Convert to streams, osquerybeat doesn't use streams
	streams := make([]*proto.Stream, 1)

	// Enforce the datastream dataset and type because the libbeat call to CreateInputsFromStreams
	// provides it's own defaults that are breaking the osquery with logstash
	// The target datastream for the publisher is expected to be logs-osquery_manager.result-<namespace>
	// while the libebeat management.CreateInputsFromStreams defaults to osquery-generic-default
	var datastream *proto.DataStream
	if rawIn.GetDataStream() != nil {
		// Copy by value and modify dataset and type
		ds := *rawIn.GetDataStream()
		ds.Dataset = config.DefaultDataset
		ds.Type = config.DefaultType
		datastream = &ds
	}

	streams[0] = &proto.Stream{
		Source:     rawIn.GetSource(),
		Id:         rawIn.GetId(),
		DataStream: datastream,
	}

	rawIn.Streams = streams

	modules, err := management.CreateInputsFromStreams(rawIn, "osquery", agentInfo)
	if err != nil {
		return nil, fmt.Errorf("error creating input list from raw expected config: %w", err)
	}
	for iter := range modules {
		modules[iter]["type"] = "log"
	}

	// format for the reloadable list needed bythe cm.Reload() method
	configList, err := management.CreateReloadConfigFromInputs(modules)
	if err != nil {
		return nil, fmt.Errorf("error creating config for reloader: %w", err)
	}
	return configList, nil
}

func xxxosquerybeatCfg(rawIn *proto.UnitExpectedConfig, agentInfo *client.AgentInfo) ([]*reload.ConfigWithMeta, error) {

	/////////////////////////////
	// BEGIN: REMOVE!!!
	/////////////////////////////

	b, err := json.Marshal(rawIn)
	if err != nil {
		fmt.Println("OSQUERYBEATCFG FAILED MARSHALLING rawIn:", err)
	}
	fmt.Println("OSQUERYBEATCFG RAW IN: ", string(b))

	b, err = json.Marshal(rawIn.GetStreams())
	if err != nil {
		fmt.Println("OSQUERYBEATCFG FAILED MARSHALLING STREAMS:", err)
	}
	fmt.Println("OSQUERYBEATCFG STREAMS: ", string(b))

	b, err = json.Marshal(rawIn.GetDataStream())
	if err != nil {
		fmt.Println("OSQUERYBEATCFG FAILED MARSHALLING DATASTREAM:", err)
	}
	fmt.Println("OSQUERYBEATCFG DATASTREAM: ", string(b))

	b, err = rawIn.GetSource().MarshalJSON()
	if err != nil {
		fmt.Println("OSQUERYBEATCFG FAILED MARSHALLING GETSOURCE:", err)
	}
	fmt.Println("OSQUERYBEATCFG SOURCE: ", string(b))

	/////////////////////////////
	// END: REMOVE!!!
	/////////////////////////////

	// Legacy behavour where we could have had one no streams from the policy
	// This wonky behavour confirmed with at least 8.13.1
	streamCount := len(rawIn.Streams)
	if streamCount == 0 {
		streamCount = 1
	}

	// Convert to streams, osquerybeat doesn't use streams
	streams := make([]*proto.Stream, 0, streamCount)

	// Enforce the datastream dataset and type because the libbeat call to CreateInputsFromStreams
	// provides it's own defaults that are breaking the osquery with logstash
	// The target datastream for the publisher is expected to be logs-osquery_manager.result-<namespace>
	// while the libebeat management.CreateInputsFromStreams defaults to osquery-generic-default
	var datastream *proto.DataStream
	if rawIn.GetDataStream() != nil {
		// Copy by value and modify dataset and type
		ds := *rawIn.GetDataStream()
		ds.Dataset = config.DefaultDataset
		ds.Type = config.DefaultType
		datastream = &ds
	}

	dsSource := rawIn.GetSource()
	delete(dsSource.Fields, "streams")
	streams = append(streams, &proto.Stream{
		Source:     dsSource,
		Id:         rawIn.GetId(),
		DataStream: datastream,
	})

	// The action.results datastream is added
	if streamCount > 1 {
		// Iterate over streams that are not data_stream: dataset: osquery_manager.result
		for _, stream := range rawIn.Streams {
			if stream.DataStream.Dataset == config.DefaultDataset {
				continue
			}
			s := &proto.Stream{
				Source:     stream.GetSource(),
				Id:         stream.GetId(),
				DataStream: stream.DataStream,
			}
			if s.DataStream != nil && s.DataStream.Namespace == "" && datastream != nil {
				s.DataStream.Namespace = datastream.Namespace
			}
			streams = append(streams, s)
		}
	}

	/////////////////////////////
	// BEGIN: REMOVE!!!
	/////////////////////////////
	b, err = json.Marshal(streams)
	if err != nil {
		fmt.Println("OSQUERYBEATCFG FAILED MARSHALLING STREAMS:", err)
	}
	fmt.Println("OSQUERYBEATCFG STREAMS RESULT: ", string(b))
	/////////////////////////////
	// END: REMOVE!!!
	/////////////////////////////

	rawIn.Streams = streams

	modules, err := management.CreateInputsFromStreams(rawIn, "osquery", agentInfo)
	if err != nil {
		return nil, fmt.Errorf("error creating input list from raw expected config: %w", err)
	}
	for iter := range modules {
		modules[iter]["type"] = "log"
	}

	/////////////////////////////
	// BEGIN: REMOVE!!!
	/////////////////////////////
	b, err = json.Marshal(modules)
	if err != nil {
		fmt.Println("OSQUERYBEATCFG MODULES ERROR:", err)
	}
	fmt.Println("OSQUERYBEATCFG MODULES: ", string(b))
	/////////////////////////////
	// END: REMOVE!!!
	/////////////////////////////

	// format for the reloadable list needed bythe cm.Reload() method
	configList, err := management.CreateReloadConfigFromInputs(modules)
	if err != nil {
		return nil, fmt.Errorf("error creating config for reloader: %w", err)
	}

	/////////////////////////////
	// BEGIN: REMOVE!!!
	/////////////////////////////
	b, err = json.Marshal(configList)
	if err != nil {
		fmt.Println("OSQUERYBEATCFG CONFIG ERROR:", err)
	}
	fmt.Println("OSQUERYBEATCFG CONFIG: ", string(b))

	/////////////////////////////
	// END: REMOVE!!!
	/////////////////////////////

	return configList, nil
}

func updateLegacyCfg(rawIn *proto.UnitExpectedConfig) {
	// Convert to streams, osquerybeat doesn't use streams
	streams := make([]*proto.Stream, 1)

	// Enforce the datastream dataset and type because the libbeat call to CreateInputsFromStreams
	// provides it's own defaults that are breaking the osquery with logstash
	// The target datastream for the publisher is expected to be logs-osquery_manager.result-<namespace>
	// while the libebeat management.CreateInputsFromStreams defaults to osquery-generic-default
	var datastream *proto.DataStream
	if rawIn.GetDataStream() != nil {
		// Copy by value and modify dataset and type
		ds := *rawIn.GetDataStream()
		ds.Dataset = config.DefaultDataset
		ds.Type = config.DefaultType
		datastream = &ds
	}

	streams[0] = &proto.Stream{
		Source:     rawIn.GetSource(),
		Id:         rawIn.GetId(),
		DataStream: datastream,
	}

	rawIn.Streams = streams
}

func updateCfg(rawIn *proto.UnitExpectedConfig) {
	// for i, stream := range rawIn.Streams {
	// 	var datastream *proto.DataStream
	// 	if rawIn.GetDataStream() != nil {
	// 		ds := *rawIn.GetDataStream()
	// 		ds.Dataset = config.DefaultDataset
	// 		ds.Type = config.DefaultType
	// 		datastream = &ds
	// 	}
	// 	stream.
	// 		rawIn.Streams[i] = &proto.Stream{
	// 		// Source:     stream.GetSource(),
	// 		// Id:         rawIn.GetId(),
	// 		DataStream: datastream,
	// 	}
	// }
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
