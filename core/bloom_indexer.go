// Copyright 2021 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"context"
	"time"
	"github.com/luxfi/evm/core/bloombits"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/geth/common"
)

const (
	// bloomThrottling is the time to wait between processing two consecutive index
	// sections. It's useful during chain upgrades to prevent disk overload.
	bloomThrottling = 100 * time.Millisecond
)

// BloomIndexer implements a core.ChainIndexer, building up a rotated bloom bits index
// for the Ethereum header bloom filters, permitting blazing fast filtering.
type BloomIndexer struct {
	size    uint64               // section size to generate bloombits for
	db      ethdb.Database       // database instance to write index data and metadata into
	gen     *bloombits.Generator // generator to rotate the bloom bits crating the bloom index
	section uint64               // Section is the section number being processed currently
	head    common.Hash          // Head is the hash of the last header processed
}

// NewBloomIndexer returns a chain indexer that generates bloom bits data for the
// canonical chain for fast logs filtering.
func NewBloomIndexer(db ethdb.Database, size, confirms uint64) *ChainIndexer {
	_ = &BloomIndexer{
		db:   db,
		size: size,
	}
	// TODO: Fix bloom indexer - ethdb.NewTable no longer exists
	// table := ethdb.NewTable(db, "B")
	// return NewChainIndexer(db, table, backend, size, confirms, bloomThrottling, "bloombits")
	return nil
}

// Reset implements core.ChainIndexerBackend, starting a new bloombits index
// section.
func (b *BloomIndexer) Reset(ctx context.Context, section uint64, lastSectionHead common.Hash) error {
	gen, err := bloombits.NewGenerator(uint(b.size))
	b.gen, b.section, b.head = gen, section, common.Hash{}
	return err
}

// Process implements core.ChainIndexerBackend, adding a new header's bloom into
// the index.
func (b *BloomIndexer) Process(ctx context.Context, header *types.Header) error {
	b.gen.AddBloom(uint(header.Number.Uint64()-b.section*b.size), header.Bloom)
	b.head = header.Hash()
	return nil
}

// Commit implements core.ChainIndexerBackend, finalizing the bloom section and
// writing it out into the database.
func (b *BloomIndexer) Commit() error {
	batch := b.db.NewBatchWithSize((int(b.size) / 8) * types.BloomBitLength)
	for i := 0; i < types.BloomBitLength; i++ {
		bits, err := b.gen.Bitset(uint(i))
		if err != nil {
			return err
		}
		// TODO: Fix bloom bits writing - rawdb.WriteBloomBits no longer exists
		// rawdb.WriteBloomBits(batch, uint(i), b.section, b.head, bitutil.CompressBytes(bits))
		_ = bits
	}
	return batch.Write()
}

// Prune returns an empty error since we don't support pruning here.
func (b *BloomIndexer) Prune(threshold uint64) error {
	return nil
}
