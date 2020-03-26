package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tennuem/tbot/pkg/provider"
)

func TestFindLinks(t *testing.T) {
	testData := []struct {
		in  *Message
		out *Message
	}{
		{
			&Message{URL: "http://p1"},
			&Message{
				URL:   "http://p1",
				Title: "test_title",
			},
		},
	}
	svc := NewService(
		NewStoreMock(),
		map[string]provider.Provider{
			"p1": provider.NewMockProvider(),
		},
		log.NewNopLogger(),
	)
	for k, c := range testData {
		res, err := svc.FindLinks(context.Background(), c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res, fmt.Sprintf("case-%d", k))
	}
}
