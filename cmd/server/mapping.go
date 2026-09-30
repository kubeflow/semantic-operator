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

	"github.com/kubeflow/semantic-operator/internal/cache"
	"github.com/kubeflow/semantic-operator/internal/dbclient"
	"github.com/kubeflow/semantic-operator/internal/serving"
	"github.com/kubeflow/semantic-operator/internal/serving/auth"
	"github.com/kubeflow/semantic-operator/internal/serving/exchange"
)

// This file maps the config catalog onto the runtime option types. The mapping
// is pure so it can be unit-tested without constructing real clients.

func toLimits(q QueryConfig) serving.Limits {
	return serving.Limits{
		DefaultRowLimit:      q.DefaultRowLimit,
		MaxRowLimit:          q.MaxRowLimit,
		MaxMetrics:           q.MaxMetrics,
		MaxDimensions:        q.MaxDimensions,
		MaxFilters:           q.MaxFilters,
		MaxFilterValues:      q.MaxFilterValues,
		MaxResultBytes:       q.MaxResultBytes,
		MaxCacheEntryBytes:   q.MaxCacheEntryBytes,
		MaxRequestBytes:      q.MaxRequestBytes,
		MaxConcurrentQueries: q.MaxConcurrent,
	}
}

// toEngineConfig builds the engine connection config. MaxResultBytes is sourced
// from the query limits so the client and the service share one ceiling.
func toEngineConfig(e EngineConfig, q QueryConfig) dbclient.Config {
	return dbclient.Config{
		Host:                  e.Connection.Host,
		Port:                  e.Connection.Port,
		User:                  e.Connection.User,
		Password:              e.Connection.Password,
		QueryTimeout:          e.Connection.QueryTimeout,
		MaxResultBytes:        q.MaxResultBytes,
		TLSEnabled:            e.Connection.TLSEnabled,
		TLSInsecureSkipVerify: e.Connection.TLSInsecureSkipVerify,
	}
}

func toAuthOptions(a AuthConfig) auth.Options {
	return auth.Options{
		Mode:            auth.Mode(a.Mode),
		JWKSURL:         a.JWKSURL,
		Issuer:          a.Issuer,
		Audience:        a.Audience,
		PrincipalClaim:  a.PrincipalClaim,
		RoleClaim:       a.RoleClaim,
		GroupsClaim:     a.GroupsClaim,
		ClaimsToCopy:    a.ClaimsToCopy,
		EngineUserClaim: a.EngineUserClaim,
	}
}

func toCacheOptions(c CacheConfig) cache.Options {
	return cache.Options{
		Addr:      c.Addr,
		Password:  c.Password,
		DB:        c.DB,
		PlanTTL:   c.PlanTTL,
		ResultTTL: c.ResultTTL,
	}
}

func toExchangeOptions(e ExchangeConfig, log *slog.Logger) exchange.Options {
	return exchange.Options{
		TokenURL:          e.TokenURL,
		ClientID:          e.ClientID,
		ClientSecret:      e.ClientSecret,
		AllowInsecureHTTP: e.AllowInsecureHTTP,
		Logger:            log,
	}
}
