---
tags: [Nodes, Lux Node]
description: Reference for all available X-chain config options and flags.
pagination_label: X-Chain Configs
sidebar_position: 2
---

# X-Chain

In order to specify a config for the X-Chain, a JSON config file should be
placed at `{chain-config-dir}/X/config.json`.

For example if `chain-config-dir` has the default value which is
`$HOME/.node/configs/chains`, then `config.json` can be placed at
`$HOME/.node/configs/chains/X/config.json`.

This allows you to specify a config to be passed into the X-Chain. The default
values for this config are:

```json
{
  "checksums-enabled": false
}
```

Default values are overridden only if explicitly specified in the config.

The parameters are as follows:

<<<<<<< HEAD:vms/avm/config.md
=======
## Transaction Indexing

### `index-transactions`

_Boolean_

Enables XVM transaction indexing if set to `true`.
When set to `true`, XVM transactions are indexed against the `address` and
`assetID` involved. This data is available via `xvm.getAddressTxs`
[API](/reference/node/x-chain/api.md#xvmgetaddresstxs).

:::note
If `index-transactions` is set to true, it must always be set to true
for the node's lifetime. If set to `false` after having been set to `true`, the
node will refuse to start unless `index-allow-incomplete` is also set to `true`
(see below).
:::

### `index-allow-incomplete`

_Boolean_

Allows incomplete indices. This config value is ignored if there is no X-Chain indexed data in the DB and
`index-transactions` is set to `false`.

>>>>>>> origin/regenesis-runtime-replay:vms/xvm/config.md
### `checksums-enabled`

_Boolean_

Enables checksums if set to `true`.
