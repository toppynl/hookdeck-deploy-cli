// Package wrangler provides utilities for reading and updating wrangler.jsonc
// configuration files used by Cloudflare Workers.
package wrangler

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/tailscale/hujson"
)

// SyncSourceURL writes the Hookdeck source URL into the given wrangler.jsonc
// file under env.<envName>.vars.HOOKDECK_SOURCE_URL.
//
// It returns true if the file was modified, or false if the existing value
// already matched sourceURL (no-op).
func SyncSourceURL(wranglerPath string, envName string, sourceURL string) (bool, error) {
	data, err := os.ReadFile(wranglerPath)
	if err != nil {
		return false, fmt.Errorf("reading wrangler file: %w", err)
	}

	// Standardize JSONC (strip comments, trailing commas) into valid JSON.
	standardized, err := hujson.Standardize(data)
	if err != nil {
		return false, fmt.Errorf("parsing JSONC: %w", err)
	}

	// Unmarshal into a generic map so we can navigate and modify the structure.
	var doc map[string]interface{}
	if err := json.Unmarshal(standardized, &doc); err != nil {
		return false, fmt.Errorf("unmarshaling wrangler JSON: %w", err)
	}

	// Navigate to env.<envName>.vars, creating intermediate maps as needed.
	envMap := ensureMap(doc, "env")
	envEntry := ensureMap(envMap, envName)
	vars := ensureMap(envEntry, "vars")

	// Check if the value is already set and matches.
	if existing, ok := vars["HOOKDECK_SOURCE_URL"].(string); ok && existing == sourceURL {
		return false, nil
	}

	// Set the new value.
	vars["HOOKDECK_SOURCE_URL"] = sourceURL

	// Write the maps back into the parent chain (ensureMap returns the child,
	// but we need to ensure the parent keys point to the right maps).
	envEntry["vars"] = vars
	envMap[envName] = envEntry
	doc["env"] = envMap

	// Marshal back to JSON with indentation to keep the file human-readable.
	output, err := json.MarshalIndent(doc, "", "\t")
	if err != nil {
		return false, fmt.Errorf("marshaling updated wrangler: %w", err)
	}

	// Append a trailing newline for POSIX compliance.
	output = append(output, '\n')

	if err := os.WriteFile(wranglerPath, output, 0644); err != nil {
		return false, fmt.Errorf("writing wrangler file: %w", err)
	}

	return true, nil
}

// ensureMap returns the map at parent[key], creating an empty map if the key
// is missing or not a map.
func ensureMap(parent map[string]interface{}, key string) map[string]interface{} {
	if child, ok := parent[key].(map[string]interface{}); ok {
		return child
	}
	child := make(map[string]interface{})
	parent[key] = child
	return child
}
