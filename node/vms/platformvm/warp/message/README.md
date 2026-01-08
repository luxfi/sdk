## P-Chain warp message payloads

This package defines parsing and serialization for the payloads of unsigned warp messages on the P-Chain.

These payloads are specified in [LP-77](https://github.com/luxfi/LPs/blob/main/LPs/77-reinventing-subnets/README.md), and are expected as part of the `payload` field of an `AddressedCall` message, with an empty source address.