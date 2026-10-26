package serving

import (
	"errors"
	"fmt"

	"github.com/kubeflow/semantic-operator/internal/planner"
)

// ErrRequestTooLarge marks a request or result that exceeds a configured
// bound. Adapters map it to 400, which is the default status in writeErr.
var ErrRequestTooLarge = errors.New("request exceeds a configured limit")

// ErrResultIncomplete is retained for compatibility and no longer returned
// by Service.Query. Truncation is reported through QueryResult.HasMore
// instead of refused, so a caller can tell a prefix from a whole result
// while still receiving the rows. See issue 12.
var ErrResultIncomplete = errors.New("result is incomplete at the default row limit: narrow the request or set an explicit limit")

// Limits bound what one request may ask for and what one result may cost.
//
// The row limit matters most. Without it a single query against a
// high-cardinality dimension returns the whole grouping, and the server reads
// every row into memory, serializes it, and hands a copy to the cache. That is
// three copies of an unbounded result inside a pod with a fixed memory limit.
//
// A request over a bound is refused, except for row-count truncation which is
// reported with a hasMore flag. Silently returning the first N rows of a
// larger result is a confidently wrong answer: the caller cannot tell
// a complete answer from a truncated one. Reporting says so plainly while
// still returning the rows.
//
// Every statement fetches one row past its effective limit, so Query can
// distinguish a complete result from a larger one it must flag. apply fills
// in the effective limit, the caller's value or the default, and that is the
// request external authorization sees. Service.Plan adds the probe after the
// policy has decided, so the extra row never reaches a limit-bounding policy.
type Limits struct {
	// DefaultRowLimit applies when a request names no limit.
	DefaultRowLimit int
	// MaxRowLimit is the largest limit a caller may ask for.
	MaxRowLimit int

	MaxMetrics    int
	MaxDimensions int
	// MaxFilters bounds row filters and metric filters together.
	MaxFilters      int
	MaxFilterValues int

	// MaxResultBytes caps the encoded result. The row limit already bounds row
	// count, so this catches the other shape: few rows carrying large values.
	MaxResultBytes int
	// MaxCacheEntryBytes caps what may be written to the result cache. Over
	// this the result is still returned, just not cached, because caching is
	// an optimization and never a reason to fail a request.
	MaxCacheEntryBytes int

	// MaxRequestBytes caps a decoded JSON request body. A semantic query is a
	// short document, so this only has to stop an unbounded upload from being
	// buffered.
	MaxRequestBytes int

	// MaxConcurrentQueries bounds how many requests may be in flight at once.
	//
	// This is a memory budget, not a throughput setting. Each in-flight
	// request can hold a whole result until its response has been written, so
	// the worst case is roughly
	//
	//	MaxConcurrentQueries x MaxResultBytes
	//
	// plus the encoded copy made while serializing. That product has to stay
	// comfortably under the pod memory limit. The defaults are 4 x 32 MiB,
	// about 128 MiB of results against a default 512 MiB limit, which leaves
	// room for the encoding copy and everything else the process is doing.
	//
	// Raising either value means raising the pod limit to match.
	MaxConcurrentQueries int
}

// DefaultLimits are generous enough for interactive analytics and sized so the
// worst case stays well inside a default 512Mi server pod. See
// MaxConcurrentQueries for the budget those two numbers form together.
func DefaultLimits() Limits {
	return Limits{
		DefaultRowLimit:      1000,
		MaxRowLimit:          10000,
		MaxMetrics:           25,
		MaxDimensions:        25,
		MaxFilters:           50,
		MaxFilterValues:      1000,
		MaxResultBytes:       32 << 20, // 32 MiB
		MaxCacheEntryBytes:   8 << 20,  // 8 MiB
		MaxRequestBytes:      1 << 20,  // 1 MiB
		MaxConcurrentQueries: 4,
	}
}

// WithDefaults fills unset fields from DefaultLimits, so a caller outside this
// package can read the values actually in force rather than the ones that
// happen to be set.
func (l Limits) WithDefaults() Limits { return l.withDefaults() }

// withDefaults fills unset fields from DefaultLimits. A zero Limits therefore
// behaves as the defaults rather than as "no limit", so a Service constructed
// without configuring them is still bounded.
func (l Limits) withDefaults() Limits {
	d := DefaultLimits()
	if l.DefaultRowLimit <= 0 {
		l.DefaultRowLimit = d.DefaultRowLimit
	}
	if l.MaxRowLimit <= 0 {
		l.MaxRowLimit = d.MaxRowLimit
	}
	if l.MaxMetrics <= 0 {
		l.MaxMetrics = d.MaxMetrics
	}
	if l.MaxDimensions <= 0 {
		l.MaxDimensions = d.MaxDimensions
	}
	if l.MaxFilters <= 0 {
		l.MaxFilters = d.MaxFilters
	}
	if l.MaxFilterValues <= 0 {
		l.MaxFilterValues = d.MaxFilterValues
	}
	if l.MaxResultBytes <= 0 {
		l.MaxResultBytes = d.MaxResultBytes
	}
	if l.MaxCacheEntryBytes <= 0 {
		l.MaxCacheEntryBytes = d.MaxCacheEntryBytes
	}
	if l.MaxRequestBytes <= 0 {
		l.MaxRequestBytes = d.MaxRequestBytes
	}
	if l.MaxConcurrentQueries <= 0 {
		l.MaxConcurrentQueries = d.MaxConcurrentQueries
	}
	// A cache entry larger than the result ceiling can never occur, so fold it
	// down rather than leaving a bound that cannot be reached.
	if l.MaxCacheEntryBytes > l.MaxResultBytes {
		l.MaxCacheEntryBytes = l.MaxResultBytes
	}
	return l
}

// apply validates the request shape and returns it with the effective row
// limit filled in: the caller's value, or the default when none was named. It
// runs before the plan cache key is computed, so the key covers the effective
// limit. It is also the request external authorization sees: Service.Plan
// adds the one-row execution probe on top only after the policy has decided,
// so a policy bounding limits at N sees N, not N+1.
func (l Limits) apply(req planner.Request) (planner.Request, error) {
	l = l.withDefaults()

	if n := len(req.Metrics); n > l.MaxMetrics {
		return req, fmt.Errorf("%w: %d metrics requested, the maximum is %d", ErrRequestTooLarge, n, l.MaxMetrics)
	}
	if n := len(req.Dimensions); n > l.MaxDimensions {
		return req, fmt.Errorf("%w: %d dimensions requested, the maximum is %d", ErrRequestTooLarge, n, l.MaxDimensions)
	}
	if n := len(req.Filters) + len(req.MetricFilters); n > l.MaxFilters {
		return req, fmt.Errorf("%w: %d row and metric filters requested in total, the maximum is %d", ErrRequestTooLarge, n, l.MaxFilters)
	}
	for _, f := range req.Filters {
		if n := len(f.Values); n > l.MaxFilterValues {
			return req, fmt.Errorf("%w: filter on %q carries %d values, the maximum is %d",
				ErrRequestTooLarge, f.Field, n, l.MaxFilterValues)
		}
	}

	switch {
	case req.Limit < 0:
		return req, fmt.Errorf("%w: limit must not be negative", ErrRequestTooLarge)
	case req.Limit == 0:
		// The default is the effective limit. The probe row past it is the
		// execution detail Service.Plan adds when the statement is built.
		req.Limit = l.DefaultRowLimit
	case req.Limit > l.MaxRowLimit:
		return req, fmt.Errorf("%w: limit %d exceeds the maximum of %d", ErrRequestTooLarge, req.Limit, l.MaxRowLimit)
	}
	return req, nil
}
