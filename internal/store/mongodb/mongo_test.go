package mongodb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewMongoStoreInvalidURI(t *testing.T) {
	store, err := NewMongoStore("invalid://localhost")
	require.Error(t, err)
	require.Nil(t, store)
}
