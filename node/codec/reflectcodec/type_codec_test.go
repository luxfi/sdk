// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package reflectcodec

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSizeWithNil(t *testing.T) {
	require := require.New(t)
	var x *int32
	y := int32(1)
	c := genericCodec{}
	// size method no longer supports nullable parameter
	// It will return error for nil pointers
	_, _, err := c.size(reflect.ValueOf(x), nil)
	require.Error(err)

	// For non-nil values
	x = &y
	len, _, err := c.size(reflect.ValueOf(y), nil)
	require.NoError(err)
	require.Equal(4, len) // int32 is 4 bytes

	len, _, err = c.size(reflect.ValueOf(x), nil)
	require.NoError(err)
	require.Equal(4, len) // pointer to int32 is also 4 bytes when dereferenced
}
