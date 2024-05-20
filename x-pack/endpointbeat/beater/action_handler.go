// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"

	"github.com/elastic/elastic-agent-libs/logp"
)

type actionHandler struct {
	log       *logp.Logger
	inputType string
}

func (a *actionHandler) Name() string {
	return a.inputType
}

// Execute handles the action request.
func (a *actionHandler) Execute(ctx context.Context, req map[string]interface{}) (map[string]interface{}, error) {

	return nil, nil
}
