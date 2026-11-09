package gitx

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseVersion(t *testing.T) {
	maj, min, err := ParseVersion("git version 2.54.0 (Apple Git-157)\n")
	require.NoError(t, err)
	assert.Equal(t, 2, maj)
	assert.Equal(t, 54, min)

	_, _, err = ParseVersion("")
	assert.Error(t, err)
}

func TestAvailable(t *testing.T) {
	f := NewFakeRunner().On("--version", Resp{Stdout: "git version 2.19.1\n"})
	_, err := Available(context.Background(), f)
	assert.ErrorContains(t, err, "too old")

	f = NewFakeRunner().On("--version", Resp{Stdout: "git version 2.20.0\n"})
	v, err := Available(context.Background(), f)
	require.NoError(t, err)
	assert.Equal(t, "2.20", v)

	f = NewFakeRunner() // unscripted → error
	_, err = Available(context.Background(), f)
	assert.ErrorContains(t, err, "not installed")
}
