package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUtils_DeduplicateStringSlice(t *testing.T) {
	data := []string{"animal", "bird", "animal", "crane", "crane", "fish"}
	result := []string{"animal", "bird", "crane", "fish"}

	assert.Equal(t, result, DeduplicateStringSlice(data))
}

func TestUtils_Max(t *testing.T) {
	assert.Equal(t, int64(2), Max(1, 2))
	assert.Equal(t, int64(2), Max(2, 1))
}

func TestUtils_Min(t *testing.T) {
	assert.Equal(t, int64(1), Min(1, 2))
	assert.Equal(t, int64(1), Min(2, 1))
}
