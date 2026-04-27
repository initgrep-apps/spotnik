// Package panes exports unexported helpers for white-box unit tests.
package panes

// CacheAge is an exported shim for the unexported cacheAge helper so that
// panes_test can exercise all 4 age buckets directly without driving through View().
var CacheAge = cacheAge
