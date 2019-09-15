package atomic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStoreAndLoad(t *testing.T) {
	var h Bool

	h.Store(false)
	assert.Equal(t, false, h.Load())
	h.Store(true)
	assert.Equal(t, true, h.Load())
}

func TestLoadDefault(t *testing.T) {
	var h Bool

	assert.Equal(t, false, h.Load())
}
