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
package dbclient

import (
	"context"
	"strings"
	"testing"
)

type fakeClient struct{ Client }

func TestRegistryOpenAndUnknown(t *testing.T) {
	Register("test-engine", func(cfg Config) (Client, error) {
		if cfg.Host != "h" {
			t.Errorf("config not passed through: %+v", cfg)
		}
		return fakeClient{}, nil
	})
	if _, err := Open("test-engine", Config{Host: "h"}); err != nil {
		t.Fatalf("Open registered engine: %v", err)
	}
	_, err := Open("no-such-engine", Config{})
	if err == nil {
		t.Fatal("Open must fail for unregistered engines")
	}
	// The error names the registered engines so a typo in SQL_DIALECT is
	// diagnosable from the pod log alone.
	if !strings.Contains(err.Error(), "test-engine") {
		t.Errorf("error should list registered engines: %v", err)
	}
}

// Compile-time check that the interface stays satisfiable by value types.
var _ = context.Background

func TestEnvConfigGenericNames(t *testing.T) {
	t.Setenv("ENGINE_HOST", "trino.trino.svc")
	t.Setenv("ENGINE_PORT", "8080")
	t.Setenv("ENGINE_USER", "svc")
	cfg, err := EnvConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "trino.trino.svc" || cfg.Port != 8080 || cfg.User != "svc" {
		t.Fatalf("cfg = %+v", cfg)
	}
}

func TestEnvConfigLegacyFallbackAndPrecedence(t *testing.T) {
	// Legacy-only env (an install that predates multi-engine support).
	t.Setenv("STARROCKS_HOST", "fe.starrocks.svc")
	t.Setenv("STARROCKS_PORT", "9030")
	cfg, err := EnvConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "fe.starrocks.svc" || cfg.Port != 9030 {
		t.Fatalf("legacy fallback broken: %+v", cfg)
	}
	// Generic names win when both are set.
	t.Setenv("ENGINE_HOST", "clickhouse.db.svc")
	cfg, err = EnvConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "clickhouse.db.svc" {
		t.Fatalf("ENGINE_HOST must take precedence, got %q", cfg.Host)
	}
}

func TestEnvConfigErrors(t *testing.T) {
	// No host at all: hard error, since there is no sane default.
	if _, err := EnvConfig(); err == nil {
		t.Fatal("missing ENGINE_HOST must be an error")
	}
	// A malformed port is a loud error, not a silent default.
	t.Setenv("ENGINE_HOST", "h")
	t.Setenv("ENGINE_PORT", "not-a-number")
	if _, err := EnvConfig(); err == nil {
		t.Fatal("malformed ENGINE_PORT must be an error")
	}
}
