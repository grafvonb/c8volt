package typex

import (
	"strings"

	"github.com/grafvonb/c8volt/toolx"
)

type Keys []string

func (k Keys) Contains(key string) bool {
	for _, v := range k {
		if v == key {
			return true
		}
	}
	return false
}

func (k Keys) String() string {
	return strings.Join(k, ",")
}

func (k Keys) Unique() Keys {
	return toolx.UniqueSlice(k)
}
