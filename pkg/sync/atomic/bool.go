package atomic

import (
	base "sync/atomic"
)

// Bool provide atomic way to read/write a boolean value.
type Bool int32

// Load atomically loads the value.
func (b *Bool) Load() bool {
	return base.LoadInt32((*int32)(b)) != 0
}

// Store atomically store a value.
func (b *Bool) Store(value bool) {
	var i int32
	if value {
		i = 1
	} else {
		i = 0
	}
	base.StoreInt32((*int32)(b), i)
}
