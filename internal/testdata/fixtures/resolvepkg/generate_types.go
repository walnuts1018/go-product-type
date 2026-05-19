//go:build adtgen_generate

package resolvepkg

import timex "time"

type LocalTime = timex.Time

// +adtgen:product Customer Address
type CustomerAddress struct{}

// +adtgen:product Customer Envelope[string]
type CustomerEnvelope struct{}

// +adtgen:product Customer timex.Time
type CustomerTime struct{}

// +adtgen:product Customer LocalTime
type CustomerLocalTime struct{}

// +adtgen:product Customer Envelope[T]
type CustomerEnvelopeForTypeParam[T any] struct{}
