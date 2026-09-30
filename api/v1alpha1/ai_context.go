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
package v1alpha1

import (
	"encoding/json"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// AIContext is the decoded form of an Ossie ai_context value, which the spec
// allows to be either a plain string or a structured object.
type AIContext struct {
	Instructions string   `json:"instructions,omitempty"`
	Synonyms     []string `json:"synonyms,omitempty"`
	Examples     []string `json:"examples,omitempty"`
}

// DecodeAIContext tolerantly decodes an Ossie ai_context JSON value. A plain
// string becomes Instructions. Unknown object keys are ignored. A nil input
// or undecodable value yields an empty context, never an error: ai_context is
// advisory metadata and must not fail validation.
func DecodeAIContext(raw *apiextensionsv1.JSON) AIContext {
	var out AIContext
	if raw == nil || len(raw.Raw) == 0 {
		return out
	}
	var s string
	if err := json.Unmarshal(raw.Raw, &s); err == nil {
		out.Instructions = s
		return out
	}
	_ = json.Unmarshal(raw.Raw, &out)
	return out
}
