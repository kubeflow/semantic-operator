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
//go:build e2e

package e2e

// profile is the single source of truth for one engine's e2e shape: what the
// harness queries and asserts, and how the orchestrator prepares the engine.
// Deploy details (values files, model manifests, credentials) live in the bash
// primitives the orchestrator calls, not here, so this stays about the test.
type profile struct {
	engine    string
	modelName string // ossie model name, used in the query URL

	metric      string // a certified flat metric on the identity model
	ratioMetric string // a certified ratio metric that requires split planning
	allowDim    string // a declared dimension not on the denied table
	denyDim     string // a declared dimension on the denied table
	maskDim     string // the masked declared dimension, or "" when the engine cannot mask
	nonDim      string // a readable field with no Ossie dimension declaration

	// setup is the make target (and args) that prepares the engine, data, and
	// per-user grants before the identity-mode releases are deployed. Trino's
	// masking dataset is the built-in tpch connector, so it needs no load;
	// StarRocks needs the retail data and grants from models-deploy.
	setup []string
}

var profiles = map[string]profile{
	"trino": {
		engine:      "trino",
		modelName:   "tpch_orders_model",
		metric:      "total_price",
		ratioMetric: "price_per_customer_balance",
		allowDim:    "orders.clerk",
		denyDim:     "customer.mktsegment",
		maskDim:     "orders.clerk",
		nonDim:      "orders.totalprice",
		setup:       []string{"trino-deploy"},
	},
	"starrocks": {
		engine:      "starrocks",
		modelName:   "retail_identity_model",
		metric:      "total_sales",
		ratioMetric: "store_productivity",
		allowDim:    "store.s_store_name",
		denyDim:     "customer.c_birth_year",
		maskDim:     "", // StarRocks has no column masking; denial only.
		nonDim:      "store_sales.ss_ext_sales_price",
		setup:       []string{"models-deploy", "KIND_ENGINE_TYPE=starrocks"},
	},
}
