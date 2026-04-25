package typex

import (
	"slices"
	"strings"

	"github.com/grafvonb/c8volt/toolx"
)

type Keys []string

func (k Keys) Contains(key string) bool {
	return slices.Contains(k, key)
}

func (k Keys) String() string {
	return strings.Join(k, ",")
}

func (k Keys) Unique() Keys {
	return toolx.UniqueSlice(k)
}
