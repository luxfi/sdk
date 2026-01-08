# Fix NodeID: nodeID lines followed by incorrectly indented Start:
/NodeID: nodeID,$/ {
    print $0
    getline
    # Fix Start: line - add proper indentation
    if ($0 ~ /^\t\tStart:/ || $0 ~ /^\t}$/) {
        if ($0 ~ /^\t}$/) {
            # Skip premature closing brace
            getline
        }
        if ($0 ~ /^\t\tStart:/) {
            sub(/^\t\t/, "\t\t\t")
        }
    }
    print $0
    next
}

# Fix missing }) after nodeID arrays
/0x11, 0x22, 0x33, 0x44,$/ {
    print $0
    getline
    if ($0 ~ /^\tnetID := ids.ID\{/) {
        print "\t})"
        print ""
    }
    print $0
    next
}

# Fix Context structs missing closing brace
/LUXAssetID: luxAssetID,$/ {
    print $0
    getline
    if ($0 !~ /^\t}$/) {
        print "\t}"
    }
    print $0
    next
}

# Fix return statement with trailing comma
/return &AddPermissionlessValidatorTx\{/ {
    in_return = 1
    print $0
    next
}

in_return && /^\t\t\t\t},$/ {
    sub(/},$/, "}")
    print $0
    print "\t\t},"
    in_return = 0
    next
}

{ print $0 }
