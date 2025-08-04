#!/bin/bash

# Revert import paths back to original for luxfi/node v1.13.4

echo "Reverting import paths for luxfi/node v1.13.4..."

# Revert consensus imports back to snow
find . -name "*.go" -type f -exec sed -i.bak \
  -e 's|"github.com/luxfi/node/consensus/validators"|"github.com/luxfi/node/snow/validators"|g' \
  -e 's|"github.com/luxfi/node/consensus/engine/chain"|"github.com/luxfi/node/snow/consensus/snowman"|g' \
  -e 's|"github.com/luxfi/node/consensus/engine/core"|"github.com/luxfi/node/snow/engine/common"|g' \
  -e 's|"github.com/luxfi/node/consensus/engine/enginetest"|"github.com/luxfi/node/snow/engine/enginetest"|g' \
  -e 's|"github.com/luxfi/node/uptime"|"github.com/luxfi/node/snow/uptime"|g' \
  -e 's|"github.com/luxfi/node/consensus/chain/block"|"github.com/luxfi/node/snow/engine/snowman/block"|g' \
  -e 's|"github.com/luxfi/node/consensus/chaintest"|"github.com/luxfi/node/snow/snowtest"|g' \
  -e 's|commonEng "github.com/luxfi/node/consensus/engine/core"|commonEng "github.com/luxfi/node/snow/engine/common"|g' \
  -e 's|"github.com/luxfi/node/consensus"|"github.com/luxfi/node/snow"|g' \
  {} \;

# Revert database imports back to node/database
find . -name "*.go" -type f -exec sed -i.bak \
  -e 's|"github.com/luxfi/database/memdb"|"github.com/luxfi/node/database/memdb"|g' \
  -e 's|"github.com/luxfi/database/prefixdb"|"github.com/luxfi/node/database/prefixdb"|g' \
  -e 's|"github.com/luxfi/database/versiondb"|"github.com/luxfi/node/database/versiondb"|g' \
  -e 's|"github.com/luxfi/database/factory"|"github.com/luxfi/node/database/factory"|g' \
  -e 's|"github.com/luxfi/database/pebbledb"|"github.com/luxfi/node/database/pebbledb"|g' \
  -e 's|luxdatabase "github.com/luxfi/database"|luxdatabase "github.com/luxfi/node/database"|g' \
  -e 's|"github.com/luxfi/database"|"github.com/luxfi/node/database"|g' \
  {} \;

# Clean up backup files
find . -name "*.bak" -delete

echo "Import paths reverted successfully!"