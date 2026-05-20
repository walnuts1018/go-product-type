//go:build adtgen_generate

package passthrough

import "fmt"

const InlinePrefix = "inline"

type Inline struct {
	Label string
}

var DefaultInline = Inline{Label: InlinePrefix}

func FormatInline(x Inline) string {
	return fmt.Sprintf("%s:%s", InlinePrefix, x.Label)
}

// +adtgen:product=Inline,Shared
type Combined struct{}
