package vmerrs

import "errors"

// Common VM errors
var (
    ErrInvalidJump = errors.New("invalid jump")
    ErrOutOfGas = errors.New("out of gas")
)
