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
	"log/slog"
	"testing"

	"github.com/kubeflow/semantic-operator/internal/serving/auth"
)

func TestToEngineConfigSharesMaxResultBytes(t *testing.T) {
	cfg := Config{
		Engine: EngineConfig{
			Connection: ConnectionConfig{Host: "h", Port: 8443, User: "u", TLSEnabled: true},
		},
		Query: QueryConfig{MaxResultBytes: 4096},
	}
	got := toEngineConfig(cfg.Engine, cfg.Query)
	if got.Host != "h" || got.Port != 8443 || got.User != "u" || !got.TLSEnabled {
		t.Errorf("engine config = %+v", got)
	}
	// The engine client and the service must share one result ceiling.
	if got.MaxResultBytes != 4096 {
		t.Errorf("MaxResultBytes = %d, want 4096 (from query limits)", got.MaxResultBytes)
	}
}

func TestToLimits(t *testing.T) {
	q := QueryConfig{
		DefaultRowLimit: 1, MaxRowLimit: 2, MaxMetrics: 3, MaxDimensions: 4,
		MaxFilters: 5, MaxFilterValues: 6, MaxResultBytes: 7, MaxCacheEntryBytes: 8,
		MaxRequestBytes: 9, MaxConcurrent: 10,
	}
	l := toLimits(q)
	if l.DefaultRowLimit != 1 || l.MaxRowLimit != 2 || l.MaxMetrics != 3 || l.MaxDimensions != 4 ||
		l.MaxFilters != 5 || l.MaxFilterValues != 6 || l.MaxResultBytes != 7 || l.MaxCacheEntryBytes != 8 ||
		l.MaxRequestBytes != 9 || l.MaxConcurrentQueries != 10 {
		t.Errorf("limits = %+v", l)
	}
}

func TestToAuthOptions(t *testing.T) {
	a := AuthConfig{
		Mode: "jwt", JWKSURL: "j", Issuer: "i", Audience: "a",
		PrincipalClaim: "p", RoleClaim: "r", GroupsClaim: "g",
		ClaimsToCopy: []string{"tenant"}, EngineUserClaim: "preferred_username",
	}
	o := toAuthOptions(a)
	if o.Mode != auth.Mode("jwt") || o.JWKSURL != "j" || o.Issuer != "i" || o.Audience != "a" ||
		o.PrincipalClaim != "p" || o.RoleClaim != "r" || o.GroupsClaim != "g" ||
		len(o.ClaimsToCopy) != 1 || o.EngineUserClaim != "preferred_username" {
		t.Errorf("auth options = %+v", o)
	}
}

func TestToExchangeOptionsCarriesInsecureAndLogger(t *testing.T) {
	log := slog.Default()
	o := toExchangeOptions(ExchangeConfig{
		TokenURL: "https://idp/token", ClientID: "c", ClientSecret: "s", AllowInsecureHTTP: true,
	}, log)
	if o.TokenURL != "https://idp/token" || o.ClientID != "c" || o.ClientSecret != "s" || !o.AllowInsecureHTTP {
		t.Errorf("exchange options = %+v", o)
	}
	if o.Logger != log {
		t.Error("exchange options must carry the logger")
	}
}
