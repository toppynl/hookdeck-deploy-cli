// Package schemas embeds the JSON Schema files for hookdeck-deploy manifests.
package schemas

import _ "embed"

//go:embed hookdeck-deploy.schema.json
var DeploySchema string
