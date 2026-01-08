// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package txs

import (
	"errors"
	"fmt"
	"math"
	"reflect"

	"github.com/luxfi/log"
	"github.com/luxfi/sdk/node/codec"
	"github.com/luxfi/sdk/node/codec/linearcodec"
	"github.com/luxfi/sdk/utils/timer/mockable"
	"github.com/luxfi/sdk/node/vms/exchangevm/fxs"
)

// CodecVersion is the current default codec version
const CodecVersion = 0

var _ Parser = (*parser)(nil)

type Parser interface {
	Codec() codec.Manager
	GenesisCodec() codec.Manager

	CodecRegistry() codec.Registry
	GenesisCodecRegistry() codec.Registry

	ParseTx(bytes []byte) (*Tx, error)
	ParseGenesisTx(bytes []byte) (*Tx, error)
}

type parser struct {
	cm  codec.Manager
	gcm codec.Manager
	c   linearcodec.Codec
	gc  linearcodec.Codec
}

func NewParser(fxs []fxs.Fx) (Parser, error) {
	// Create a basic logger for parsing
	return NewCustomParser(
		make(map[reflect.Type]int),
		&mockable.Clock{},
		log.NoLog{},
		fxs,
	)
}

func NewCustomParser(
	typeToFxIndex map[reflect.Type]int,
	clock *mockable.Clock,
	log log.Logger,
	fxs []fxs.Fx,
) (Parser, error) {
	gc := linearcodec.NewDefault()
	c := linearcodec.NewDefault()

	gcm := codec.NewManager(math.MaxInt32)
	cm := codec.NewDefaultManager()

	err := errors.Join(
		c.RegisterType(&BaseTx{}),
		c.RegisterType(&CreateAssetTx{}),
		c.RegisterType(&OperationTx{}),
		c.RegisterType(&ImportTx{}),
		c.RegisterType(&ExportTx{}),
		cm.RegisterCodec(CodecVersion, c),

		gc.RegisterType(&BaseTx{}),
		gc.RegisterType(&CreateAssetTx{}),
		gc.RegisterType(&OperationTx{}),
		gc.RegisterType(&ImportTx{}),
		gc.RegisterType(&ExportTx{}),
		gcm.RegisterCodec(CodecVersion, gc),
	)
	if err != nil {
		return nil, err
	}

	vm := &fxVM{
		typeToFxIndex: typeToFxIndex,
		clock:         clock,
		log:           log,
	}
	for i, fx := range fxs {
		vm.codecRegistry = &codecRegistry{
			codecs:      []codec.Registry{gc, c},
			index:       i,
			typeToIndex: vm.typeToFxIndex,
		}
		// Initialize with a proper VM that has a logger
		if err := fx.Initialize(vm); err != nil {
			return nil, err
		}
	}
	return &parser{
		cm:  cm,
		gcm: gcm,
		c:   c,
		gc:  gc,
	}, nil
}

func (p *parser) Codec() codec.Manager {
	return p.cm
}

func (p *parser) GenesisCodec() codec.Manager {
	return p.gcm
}

func (p *parser) CodecRegistry() codec.Registry {
	return p.c
}

func (p *parser) GenesisCodecRegistry() codec.Registry {
	return p.gc
}

func (p *parser) ParseTx(bytes []byte) (*Tx, error) {
	return parse(p.cm, bytes)
}

func (p *parser) ParseGenesisTx(bytes []byte) (*Tx, error) {
	return parse(p.gcm, bytes)
}

func parse(cm codec.Manager, signedBytes []byte) (*Tx, error) {
	tx := &Tx{}
	parsedVersion, err := cm.Unmarshal(signedBytes, tx)
	if err != nil {
		return nil, err
	}
	if parsedVersion != CodecVersion {
		return nil, fmt.Errorf("expected codec version %d but got %d", CodecVersion, parsedVersion)
	}

	unsignedBytesLen, err := cm.Size(CodecVersion, &tx.Unsigned)
	if err != nil {
		return nil, fmt.Errorf("couldn't calculate UnsignedTx marshal length: %w", err)
	}

	unsignedBytes := signedBytes[:unsignedBytesLen]
	tx.SetBytes(unsignedBytes, signedBytes)
	return tx, nil
}
