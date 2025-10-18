#!/usr/bin/env python3

import re

# Fix orchestrator.go
orch_path = "/Users/chalkan3/.projects/do-droplet-create/internal/orchestrator/orchestrator.go"
with open(orch_path, 'r') as f:
    content = f.read()

# Fix 1: Storage comparison
content = content.replace(
    'if o.config.Storage != nil && len(o.config.Storage.Classes) > 0',
    'if len(o.config.Storage.Classes) > 0'
)

# Fix 2: LoadBalancers - change to LoadBalancer (singular)
content = content.replace('o.config.LoadBalancers', '[]*config.LoadBalancerConfig{&o.config.LoadBalancer}')

# Fix 3: Fix pulumi.Map conversions
content = re.sub(
    r'pulumi\.Map\((\w+Outputs)\)',
    r'pulumi.ToMap(\1)',
    content
)

# Fix 4: Remove remaining Message field from LogArgs
content = re.sub(
    r'&pulumi\.LogArgs\{\s*Message:\s*pulumi\.Sprintf\([^)]+\),?\s*\}',
    'nil',
    content
)

with open(orch_path, 'w') as f:
    f.write(content)

print("Fixed orchestrator.go")

# Fix orchestrator_component.go
comp_path = "/Users/chalkan3/.projects/do-droplet-create/internal/orchestrator/orchestrator_component.go"
with open(comp_path, 'r') as f:
    content = f.read()

# Fix 1: Remove unused variable
content = content.replace(
    'addonsComponent, err := NewAddonsComponent',
    '_, err = NewAddonsComponent'
)

# Fix 2: Fix Nodes.ToMapOutput - Nodes is an array, not a map
content = content.replace(
    'component.Nodes = nodeComponent.Nodes.ToMapOutput()',
    'component.Nodes = pulumi.Map{"nodes": nodeComponent.Nodes}.ToMapOutput()'
)

with open(comp_path, 'w') as f:
    f.write(content)

print("Fixed orchestrator_component.go")
print("All errors fixed!")