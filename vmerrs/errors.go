package vmerrs

import "errors"

// Common VM errors
var (
    ErrInvalidJump = errors.New("invalid jump")
    ErrOutOfGas = errors.New("out of gas")
    ErrMaxInitCodeSizeExceeded = errors.New("max init code size exceeded")
    ErrSenderAddressNotAllowListed = errors.New("sender address not allow listed")
    ErrExecutionReverted = errors.New("execution reverted")
    ErrWriteProtection = errors.New("write protection")
    ErrDepth = errors.New("max call depth exceeded")
    ErrInsufficientBalance = errors.New("insufficient balance for transfer")
    ErrAddrProhibited = errors.New("address prohibited")
    ErrNonceUintOverflow = errors.New("nonce uint64 overflow")
    ErrContractAddressCollision = errors.New("contract address collision")
    ErrMaxCodeSizeExceeded = errors.New("max code size exceeded")
    ErrInvalidCode = errors.New("invalid code: must not begin with 0xef")
    ErrCodeStoreOutOfGas = errors.New("contract creation code storage out of gas")
    ErrGasUintOverflow = errors.New("gas uint64 overflow")
    ErrReturnDataOutOfBounds = errors.New("return data out of bounds")
    ErrInvalidCoinbase = errors.New("invalid coinbase")
)
