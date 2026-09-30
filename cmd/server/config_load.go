/*
Copyright The Kubeflow Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"time"

	"github.com/kubeflow/semantic-operator/internal/confload"
)

// defaults returns the built-in configuration so the binary is runnable with no
// config file; the file and env layers override only what they set. Query
// limits left at zero fall back to the serving package's built-in bounds.
func defaults() Config {
	return Config{
		Logging: LoggingConfig{Level: "info"},
		Server: ServerConfig{
			ListenAddr:        ":8090",
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			IdleTimeout:       120 * time.Second,
			RESTTimeout:       60 * time.Second,
			MaxHeaderBytes:    1 << 20,
		},
		Engine: EngineConfig{
			Dialect:  "starrocks",
			Identity: EngineIdentityConfig{Mode: "static"},
		},
		Auth: AuthConfig{Mode: "header"},
		Cache: CacheConfig{
			PlanTTL:   24 * time.Hour,
			ResultTTL: 60 * time.Second,
		},
		Query: QueryConfig{
			QueueWait: 5 * time.Second,
		},
		Store: StoreConfig{
			WatchNamespace: "semantic-system",
		},
	}
}

// Load builds the server configuration from defaults, an optional YAML config
// file at path, and SEMANTIC__ environment overrides. See the confload package.
func Load(path string) (Config, error) {
	return confload.Load(defaults(), path)
}
