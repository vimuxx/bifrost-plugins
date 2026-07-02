package main

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strings"

	schemas "github.com/maximhq/bifrost/core/schemas"
)

type RoutingTarget struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

type SessionRoutingPlugin struct {
	targets []RoutingTarget
}

var pluginInstance *SessionRoutingPlugin

func GetName() string {
	return "session-routing"
}

func Init(config any) error {
	var targets []RoutingTarget

	if cfg, ok := config.(map[string]any); ok {
		if rawTargets, ok := cfg["targets"].([]any); ok {
			for _, item := range rawTargets {
				t := item.(map[string]any)
				p, _ := t["provider"].(string)
				m, _ := t["model"].(string)
				if p != "" && m != "" {
					targets = append(targets, RoutingTarget{Provider: p, Model: m})
				}
			}
		}
	}

	if len(targets) == 0 {
		return fmt.Errorf("plugin 'session-routing' requires at least one target in configuration")
	}

	pluginInstance = &SessionRoutingPlugin{targets: targets}
	return nil
}

func Cleanup() error {
	return nil
}

func HTTPTransportPreHook(_ *schemas.BifrostContext, req *schemas.HTTPRequest) (*schemas.HTTPResponse, error) {
	if pluginInstance == nil {
		return nil, nil
	}

	sid, _ := req.Headers["X-Claude-Code-Session-Id"]
	if sid == "" {
		return nil, nil
	}

	var payload map[string]any
	if err := json.Unmarshal(req.Body, &payload); err != nil {
		fmt.Printf("[SessionRouting] Failed to parse body: %v\n", err)
		return nil, nil
	}

	modelStr, _ := payload["model"].(string)
	if strings.Contains(modelStr, "/") {
		parts := strings.SplitN(modelStr, "/", 2)
		if schemas.IsKnownProvider(parts[0]) {
			return nil, nil
		}
	}

	h := fnv.New32a()
	h.Write([]byte(sid))
	target := pluginInstance.targets[h.Sum32()%uint32(len(pluginInstance.targets))]
	newModel := fmt.Sprintf("%s/%s", target.Provider, target.Model)
	payload["model"] = newModel

	newBody, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("[SessionRouting] Failed to marshal body: %v\n", err)
		return nil, nil
	}
	req.Body = newBody

	return nil, nil
}
