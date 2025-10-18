#!/usr/bin/env python3

import re

# Fix orchestrator.go
path = "/Users/chalkan3/.projects/do-droplet-create/internal/orchestrator/orchestrator.go"

with open(path, 'r') as f:
    content = f.read()

# Fix 1: Remove Message field from LogArgs
content = re.sub(
    r'&pulumi\.LogArgs\{\s*Message:\s*pulumi\.Sprintf\([^)]+\)\s*\}',
    'nil',
    content
)

# Fix 2: Change o.configureOSFirewalls to comment
content = content.replace(
    'if err := o.configureOSFirewalls(); err != nil {',
    '// OS firewall configuration moved to component\n\t// if err := o.configureOSFirewalls(); err != nil {'
)
content = content.replace(
    'return fmt.Errorf("failed to configure OS firewalls: %w", err)\n\t}',
    '// \treturn fmt.Errorf("failed to configure OS firewalls: %w", err)\n\t// }'
)

# Fix 3: Change nodeConfig and poolConfig to pointers
content = re.sub(
    r'for _, nodeConfig := range o\.config\.Nodes \{',
    'for i := range o.config.Nodes {\n\t\tnodeConfig := &o.config.Nodes[i]',
    content
)

content = re.sub(
    r'for poolName, poolConfig := range o\.config\.NodePools \{',
    'for poolName := range o.config.NodePools {\n\t\tpoolConfig := o.config.NodePools[poolName]',
    content
)

# Fix 4: Replace o.config.DNS with o.config.Network.DNS
content = content.replace('o.config.DNS.', 'o.config.Network.DNS.')

# Fix 5: Fix ingressIP comparison
content = content.replace(
    'if o.dnsManager != nil && ingressIP != nil {',
    'if o.dnsManager != nil {'
)

with open(path, 'w') as f:
    f.write(content)

print("Fixed orchestrator.go")