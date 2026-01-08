# GetAllValidatorsAt RPC Endpoint

## Overview
The `GetAllValidatorsAt` endpoint returns the validator sets of all networks (including the primary network) at a specified height.

## Endpoint
```
POST /ext/bc/P
```

## Request Format

### Method
```json
{
  "jsonrpc": "2.0",
  "method": "platform.getAllValidatorsAt",
  "params": {
    "height": 100
  },
  "id": 1
}
```

### Parameters
- `height` (number | "proposed"): The blockchain height to query. Use `"proposed"` for the current proposed height.

## Response Format

```json
{
  "jsonrpc": "2.0",
  "result": {
    "validatorSets": {
      "11111111111111111111111111111111LpoYY": {
        "NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg": {
          "nodeID": "NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg",
          "publicKey": "0x8f95423f7142d00a48e1014a3de8d28907d420dc33b3052a6dee03a3f2941a393c2351e354704ca66a3fc29870282e15",
          "weight": 2000000000000
        }
      },
      "2DeHa5NWHHHB8yS8QuAXBDXQbSYbvq5bBP": {
        "NodeID-GvPWg2xDazqLeTw7r4sQP8D2S8kGe5FWH": {
          "nodeID": "NodeID-GvPWg2xDazqLeTw7r4sQP8D2S8kGe5FWH",
          "publicKey": "0x9f95423f7142d00a48e1014a3de8d28907d420dc33b3052a6dee03a3f2941a393c2351e354704ca66a3fc29870282e16",
          "weight": 1000000000000
        }
      }
    }
  },
  "id": 1
}
```

### Response Fields
- `validatorSets`: Map of network ID to validator set
  - Key: Network ID (string)
  - Value: Map of node ID to validator information
    - `nodeID`: The validator's node ID
    - `publicKey`: The validator's BLS public key (hex-encoded with 0x prefix)
    - `weight`: The validator's staking weight in nLUX

## Examples

### Get validators at specific height
```bash
curl -X POST --data '{
  "jsonrpc":"2.0",
  "id"     :1,
  "method" :"platform.getAllValidatorsAt",
  "params": {
    "height": 1000
  }
}' -H 'content-type:application/json;' http://127.0.0.1:9650/ext/bc/P
```

### Get validators at proposed height
```bash
curl -X POST --data '{
  "jsonrpc":"2.0",
  "id"     :1,
  "method" :"platform.getAllValidatorsAt",
  "params": {
    "height": "proposed"
  }
}' -H 'content-type:application/json;' http://127.0.0.1:9650/ext/bc/P
```

## Use Cases

1. **Network Monitoring**: Track validator participation across all networks
2. **Historical Queries**: Analyze validator sets at specific historical heights
3. **Cross-Network Analysis**: Compare validator distributions across multiple networks
4. **Consensus Verification**: Verify validator state for cross-chain operations

## Implementation Notes

- The endpoint returns all networks with at least one validator
- Primary network (ID: `11111111111111111111111111111111LpoYY`) is always included
- Validator weights are in nLUX (1 LUX = 1,000,000,000 nLUX)
- BLS public keys are 48-byte values encoded as hex with "0x" prefix

## Related Endpoints

- `platform.getValidatorsAt`: Get validators for a specific network at a height
- `platform.getCurrentValidators`: Get current validators for a network
- `platform.getPendingValidators`: Get pending validators for a network
