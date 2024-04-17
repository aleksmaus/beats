package cmd

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const rawInLegacyJSON = `{
    "source": {
        "data_stream": {
            "namespace": "default"
        },
        "id": "74c7d0a8-ce04-4663-95da-ff6d537c268c",
        "meta": {
            "package": {
                "name": "osquery_manager",
                "version": "1.12.1"
            }
        },
        "name": "osquery_manager-1",
        "package_policy_id": "74c7d0a8-ce04-4663-95da-ff6d537c268c",
        "policy": {
            "revision": 2
        },
        "revision": 1,
        "streams": [
        ],
        "type": "osquery"
    },
    "id": "74c7d0a8-ce04-4663-95da-ff6d537c268c",
    "type": "osquery",
    "name": "osquery_manager-1",
    "revision": 1,
    "meta": {
        "source": {
            "package": {
                "name": "osquery_manager",
                "version": "1.12.1"
            }
        },
        "package": {
            "source": {
                "name": "osquery_manager",
                "version": "1.12.1"
            },
            "name": "osquery_manager",
            "version": "1.12.1"
        }
    },
    "data_stream": {
        "source": {
            "namespace": "default"
        },
        "namespace": "default"
    },
    "streams": [
    ]
}`

const rawInJSON = `{
    "source": {
        "data_stream": {
            "namespace": "default"
        },
        "id": "74c7d0a8-ce04-4663-95da-ff6d537c268c",
        "meta": {
            "package": {
                "name": "osquery_manager",
                "version": "1.12.1"
            }
        },
        "name": "osquery_manager-1",
        "package_policy_id": "74c7d0a8-ce04-4663-95da-ff6d537c268c",
        "policy": {
            "revision": 2
        },
        "revision": 1,
        "streams": [
            {
                "data_stream": {
                    "dataset": "osquery_manager.action.results",
                    "type": "logs"
                },
                "id": "osquery-osquery_manager.action.results-74c7d0a8-ce04-4663-95da-ff6d537c268c",
                "query": null
            },
            {
                "data_stream": {
                    "dataset": "osquery_manager.result",
                    "type": "logs"
                },
                "id": null,
                "query": null
            }
        ],
        "type": "osquery"
    },
    "id": "74c7d0a8-ce04-4663-95da-ff6d537c268c",
    "type": "osquery",
    "name": "osquery_manager-1",
    "revision": 1,
    "meta": {
        "source": {
            "package": {
                "name": "osquery_manager",
                "version": "1.12.1"
            }
        },
        "package": {
            "source": {
                "name": "osquery_manager",
                "version": "1.12.1"
            },
            "name": "osquery_manager",
            "version": "1.12.1"
        }
    },
    "data_stream": {
        "source": {
            "namespace": "default"
        },
        "namespace": "default"
    },
    "streams": [
        {
            "source": {
                "data_stream": {
                    "dataset": "osquery_manager.action.results",
                    "type": "logs"
                },
                "id": "osquery-osquery_manager.action.results-74c7d0a8-ce04-4663-95da-ff6d537c268c",
                "query": null
            },
            "id": "osquery-osquery_manager.action.results-74c7d0a8-ce04-4663-95da-ff6d537c268c",
            "data_stream": {
                "source": {
                    "dataset": "osquery_manager.action.results",
                    "type": "logs"
                },
                "dataset": "osquery_manager.action.results",
                "type": "logs"
            }
        },
        {
            "source": {
                "data_stream": {
                    "dataset": "osquery_manager.result",
                    "type": "logs"
                },
                "id": null,
                "query": null
            },
            "data_stream": {
                "source": {
                    "dataset": "osquery_manager.result",
                    "type": "logs"
                },
                "dataset": "osquery_manager.result",
                "type": "logs"
            }
        }
    ]
}`

func TestOsquerybeatCfg(t *testing.T) {
	var rawIn proto.UnitExpectedConfig
	err := json.Unmarshal([]byte(rawInJSON), &rawIn)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := osquerybeatCfg(&rawIn, &client.AgentInfo{ID: "abc7d0a8-ce04-4663-95da-ff6d537c268f", Version: "8.13.1"})
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range cfg {
		m := mapstr.M{}
		err = c.Config.Unpack(&m)
		if err != nil {
			t.Fatal(err)
		}

		b, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}

		fmt.Println(string(b))
	}
}
