// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package admin

import (
	"context"

	"github.com/luxfi/database"
	sdkadmin "github.com/luxfi/sdk/api/admin"
)

var _ database.KeyValueReader = (*KeyValueReader)(nil)

type KeyValueReader struct {
	client *sdkadmin.Client
}

func NewKeyValueReader(client *sdkadmin.Client) *KeyValueReader {
	return &KeyValueReader{
		client: client,
	}
}

func (r *KeyValueReader) Has(key []byte) (bool, error) {
	_, err := r.client.DBGet(context.Background(), key)
	if err == database.ErrNotFound {
		return false, nil
	}
	return err == nil, err
}

func (r *KeyValueReader) Get(key []byte) ([]byte, error) {
	return r.client.DBGet(context.Background(), key)
}
