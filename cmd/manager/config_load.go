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

// defaults returns the built-in manager configuration so the binary is runnable
// with no config file; the file and env layers override only what they set.
//
// The leader-election durations are left at zero on purpose: zero means "use
// controller-runtime's default", and the mapping onto ctrl.Options must leave
// those nil rather than pass a literal zero.
func defaults() Config {
	return Config{
		Metrics: MetricsConfig{BindAddress: ":8080"},
		Health:  HealthConfig{HealthProbeBindAddress: ":8081"},
		LeaderElection: LeaderElectionConfig{
			LeaderElect:  false,
			ResourceName: "semantic-operator.semantic.ossie.io",
		},
		Engine: EngineConfig{Dialect: "starrocks"},
		Controller: ControllerConfig{
			ViewDatabase: "semantic_views",
			ResyncPeriod: 5 * time.Minute,
		},
		Logging: LoggingConfig{Level: "info"},
	}
}

// Load builds the manager configuration from defaults, an optional YAML config
// file at path, and SEMANTIC__ environment overrides. See the confload package.
// The controller-runtime flags remain the highest-precedence layer, folded in
// and mapped onto ctrl.Options when main wires the manager.
func Load(path string) (Config, error) {
	return confload.Load(defaults(), path)
}
