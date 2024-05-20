package beater

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/x-pack/endpointbeat/internal/config"
)

func handleEndpointConfiguration(inputConfigs []config.InputConfig) {
	// TODO: pass configuration to endpoint library
	fmt.Println("ENDPOINT CONFIGURATION HANDLING")
}

func handleEndpointAction(ctx context.Context, req map[string]interface{}) (map[string]interface{}, error) {
	// TODO: pass the action to endpoint library
	fmt.Println("ENDPOINT ACTION HANDLING")
	return map[string]interface{}{}, nil
}
