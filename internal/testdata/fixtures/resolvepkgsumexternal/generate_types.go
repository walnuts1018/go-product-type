//go:build adtgen_generate

package resolvepkgsumexternal

import "time"

var _ time.Time

// +adtgen:sum Hoge time.Time
type HogeOrTime struct{}
