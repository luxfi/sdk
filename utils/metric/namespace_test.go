// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utilmetric

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppendNamespace(t *testing.T) {
	tests := []struct {
		prefix   string
		suffix   string
		expected string
	}{
		{
			prefix:   "node",
			suffix:   "isgreat",
			expected: "node_isgreat",
		},
		{
			prefix:   "",
			suffix:   "sucks",
			expected: "sucks",
		},
		{
			prefix:   "sucks",
			suffix:   "",
			expected: "sucks",
		},
		{
			prefix:   "",
			suffix:   "",
			expected: "",
		},
	}
	for _, test := range tests {
		t.Run(strings.Join([]string{test.prefix, test.suffix}, "_"), func(t *testing.T) {
			namespace := AppendNamespace(test.prefix, test.suffix)
			require.Equal(t, test.expected, namespace)
		})
	}
}
