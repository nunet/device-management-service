package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsLiteral(t *testing.T) {
	intValue := int(2)
	float32Value := float32(3.456)
	float64Value := float64(45.59736)

	stringValue := "some string"

	// negative assertions
	assert.False(t, IsLiteral(intValue))
	assert.False(t, IsLiteral(float32Value))
	assert.False(t, IsLiteral(float64Value))

	// negative assertions
	assert.True(t, IsLiteral(stringValue))
}

func TestIsBlank(t *testing.T) {
	// positive assertions
	assert.True(t, IsBlank("   "))
	assert.True(t, IsBlank(""))
	assert.True(t, IsBlank(" "))

	// negative assertions
	assert.False(t, IsBlank("  a  "))
	assert.False(t, IsBlank("a"))
}

func TestIsNotBlank(t *testing.T) {
	// positive assertions
	assert.True(t, IsNotBlank("  a  "))
	assert.True(t, IsNotBlank("a"))

	// negative assertions
	assert.False(t, IsNotBlank("   "))
	assert.False(t, IsNotBlank(""))
	assert.False(t, IsNotBlank(" "))
}
