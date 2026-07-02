package main

import (
	"encoding/json"
	"strings"

	schemas "github.com/maximhq/bifrost/core/schemas"
)

type OpencodeDisguisePlugin struct{}

var pluginInstance *OpencodeDisguisePlugin

func GetName() string {
	return "opencode-disguise"
}

func Init(config any) error {
	pluginInstance = &OpencodeDisguisePlugin{}
	return nil
}

func Cleanup() error {
	return nil
}

func HTTPTransportPreHook(_ *schemas.BifrostContext, req *schemas.HTTPRequest) (*schemas.HTTPResponse, error) {
	if pluginInstance == nil {
		return nil, nil
	}

	// Only process requests targeting opencode provider
	modelStr, ok := getModel(req.Body)
	if !ok || !strings.Contains(modelStr, "opencode-zen") {
		return nil, nil
	}

	// Read source IDs before setting headers
	sessionId := req.CaseInsensitiveHeaderLookup("X-Claude-Code-Session-Id")
	requestId := req.CaseInsensitiveHeaderLookup("X-Request-Id")

	// Inject opencode disguise headers via x-bf-eh-* so ConvertToBifrostContext
	// collects them into ExtraHeaders and forwards them to the upstream provider.
	// Using req.Headers is more robust than ctx.SetValue(ExtraHeaders, ...):
	// ConvertToBifrostContext may overwrite the context if the client also sends
	// x-bf-eh-* headers, but writing to req.Headers ensures the plugin's
	// dynamically-generated values survive.
	req.Headers["x-bf-eh-x-opencode-client"]  = "cli"
	req.Headers["x-bf-eh-x-opencode-project"] = "global"
	req.Headers["x-bf-eh-user-agent"]         = "opencode/1.17.10"
	if sessionId != "" {
		req.Headers["x-bf-eh-x-opencode-session"] = "ses_" + strings.ReplaceAll(sessionId, "-", "")[:26]
	}
	if requestId != "" {
		req.Headers["x-bf-eh-x-opencode-request"] = "msg_" + strings.ReplaceAll(requestId, "-", "")[:26]
	}

	return nil, nil
}

func getModel(body []byte) (string, bool) {
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return "", false
	}
	model, ok := m["model"].(string)
	return model, ok
}
