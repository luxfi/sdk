# Nets

The Lux network consists of the Primary Network and a collection of
sub-networks (subnets).

## Net Creation

Nets are created by issuing a *CreateNetTx*. After a *CreateNetTx* is
accepted, a new subnet will exist with the *NetID* equal to the *TxID* of the
*CreateNetTx*. The *CreateNetTx* creates a permissioned subnet. The
*Owner* field in *CreateNetTx* specifies who can modify the state of the
subnet.

## Permissioned Nets

A permissioned subnet can be modified by a few different transactions.

- CreateChainTx
  - Creates a new chain that will be validated by all validators of the subnet.
- AddNetValidatorTx
  - Adds a new validator to the subnet with the specified *StartTime*,
    *EndTime*, and *Weight*.
- RemoveNetValidatorTx
  - Removes a validator from the subnet.
- TransformNetTx
  - Converts the permissioned subnet into a permissionless subnet.
  - Specifies all of the staking parameters.
    - LUX is not allowed to be used as a staking token. In general, it is not
      advisable to have multiple subnets using the same staking token.
  - After becoming a permissionless subnet, previously added permissioned
    validators will remain to finish their staking period.
  - No more chains will be able to be added to the subnet.

### Permissionless Nets

Currently, nothing can be performed on a permissionless subnet.
