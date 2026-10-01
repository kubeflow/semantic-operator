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
// Package starrocks implements the StarRocks SQL dialect.
package starrocks

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kubeflow/semantic-operator/internal/emitter"
)

// Dialect emits StarRocks (MySQL-family) SQL.
type Dialect struct{}

func init() { emitter.Register(Dialect{}) }

func (Dialect) Name() string { return "starrocks" }

func (Dialect) QuoteIdent(ident string) string {
	return "`" + strings.ReplaceAll(ident, "`", "``") + "`"
}

func (d Dialect) QualifyTable(catalog, database, table string) string {
	return d.QuoteIdent(catalog) + "." + d.QuoteIdent(database) + "." + d.QuoteIdent(table)
}

func (Dialect) Literal(v any) (string, error) {
	switch x := v.(type) {
	case nil:
		return "NULL", nil
	case string:
		return "'" + strings.ReplaceAll(strings.ReplaceAll(x, `\`, `\\`), "'", "''") + "'", nil
	case bool:
		if x {
			return "TRUE", nil
		}
		return "FALSE", nil
	case int:
		return strconv.Itoa(x), nil
	case int32:
		return strconv.FormatInt(int64(x), 10), nil
	case int64:
		return strconv.FormatInt(x, 10), nil
	case float32:
		return strconv.FormatFloat(float64(x), 'g', -1, 32), nil
	case float64:
		// JSON numbers decode as float64; render integral values as integers.
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10), nil
		}
		return strconv.FormatFloat(x, 'g', -1, 64), nil
	case time.Time:
		return "'" + x.UTC().Format("2006-01-02 15:04:05") + "'", nil
	default:
		return "", fmt.Errorf("unsupported literal type %T", v)
	}
}

func (Dialect) DateTrunc(grain, scalar string) string {
	return "DATE_TRUNC('" + grain + "', " + scalar + ")"
}

func (Dialect) NullSafeEq(a, b string) string {
	return a + " <=> " + b
}

func (Dialect) CreateSchema(quotedSchema string) string {
	return "CREATE DATABASE IF NOT EXISTS " + quotedSchema
}
