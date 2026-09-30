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
	"github.com/kubeflow/semantic-operator/internal/dbclient"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

// This file maps the config catalog onto controller-runtime's option types.

func toEngineConfig(e EngineConfig) dbclient.Config {
	return dbclient.Config{
		Host:                  e.Connection.Host,
		Port:                  e.Connection.Port,
		User:                  e.Connection.User,
		Password:              e.Connection.Password,
		QueryTimeout:          e.Connection.QueryTimeout,
		TLSEnabled:            e.Connection.TLSEnabled,
		TLSInsecureSkipVerify: e.Connection.TLSInsecureSkipVerify,
	}
}

// toManagerOptions builds ctrl.Options from the config. Zero leader-election
// durations are left as nil pointers so controller-runtime applies its own
// defaults rather than a literal zero. An empty WatchNamespaces means
// cluster-wide.
func toManagerOptions(cfg Config, scheme *runtime.Scheme) ctrl.Options {
	opts := ctrl.Options{
		Scheme:                  scheme,
		Metrics:                 metricsserver.Options{BindAddress: cfg.Metrics.BindAddress},
		HealthProbeBindAddress:  cfg.Health.HealthProbeBindAddress,
		LeaderElection:          cfg.LeaderElection.LeaderElect,
		LeaderElectionID:        cfg.LeaderElection.ResourceName,
		LeaderElectionNamespace: cfg.LeaderElection.ResourceNamespace,
	}
	if d := cfg.LeaderElection.LeaseDuration; d > 0 {
		opts.LeaseDuration = &d
	}
	if d := cfg.LeaderElection.RenewDeadline; d > 0 {
		opts.RenewDeadline = &d
	}
	if d := cfg.LeaderElection.RetryPeriod; d > 0 {
		opts.RetryPeriod = &d
	}
	if len(cfg.Cache.WatchNamespaces) > 0 {
		byNamespace := make(map[string]cache.Config, len(cfg.Cache.WatchNamespaces))
		for _, ns := range cfg.Cache.WatchNamespaces {
			byNamespace[ns] = cache.Config{}
		}
		opts.Cache = cache.Options{DefaultNamespaces: byNamespace}
	}
	return opts
}
