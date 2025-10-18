#!/usr/bin/env python3

import os
import re

# Fix 1: Remove unused variable in checker.go
def fix_checker():
    path = "/Users/chalkan3/.projects/do-droplet-create/pkg/health/checker.go"
    with open(path, 'r') as f:
        lines = f.readlines()

    # Find and remove the unused totalCount variable around line 377
    for i in range(370, 385):
        if i < len(lines) and "totalCount" in lines[i]:
            # Comment out the line instead of deleting
            lines[i] = "//" + lines[i]
            break

    with open(path, 'w') as f:
        f.writelines(lines)
    print("Fixed checker.go")

# Fix 2: Convert map[string]interface{} to pulumi.Map
def fix_pulumi_map_conversions():
    # Fix rke.go
    rke_path = "/Users/chalkan3/.projects/do-droplet-create/pkg/cluster/rke.go"
    with open(rke_path, 'r') as f:
        content = f.read()

    # Replace pulumi.Map(nodeInfo) with proper conversion
    content = content.replace(
        "pulumi.Map(nodeInfo)",
        "pulumi.ToMap(nodeInfo)"
    )

    with open(rke_path, 'w') as f:
        f.write(content)
    print("Fixed rke.go")

    # Fix dns/manager.go
    dns_path = "/Users/chalkan3/.projects/do-droplet-create/pkg/dns/manager.go"
    with open(dns_path, 'r') as f:
        content = f.read()

    # Replace pulumi.Map(dnsInfo) with proper conversion
    content = content.replace(
        "pulumi.Map(dnsInfo)",
        "pulumi.ToMap(dnsInfo)"
    )

    # Fix nil comparisons for StringOutput
    content = content.replace(
        "initialIP == nil",
        'initialIP == pulumi.String("").ToStringOutput()'
    )

    with open(dns_path, 'w') as f:
        f.write(content)
    print("Fixed dns/manager.go")

    # Fix wireguard.go
    wg_path = "/Users/chalkan3/.projects/do-droplet-create/pkg/security/wireguard.go"
    with open(wg_path, 'r') as f:
        content = f.read()

    # Replace pulumi.Map(nodeIPs) with proper conversion
    content = content.replace(
        "pulumi.Map(nodeIPs)",
        "pulumi.ToMap(nodeIPs)"
    )

    with open(wg_path, 'w') as f:
        f.write(content)
    print("Fixed wireguard.go")

    # Fix sshkeys.go
    ssh_path = "/Users/chalkan3/.projects/do-droplet-create/pkg/security/sshkeys.go"
    with open(ssh_path, 'r') as f:
        content = f.read()

    # Fix nil comparisons for StringOutput
    content = content.replace(
        "s.publicKey == nil",
        's.publicKey == pulumi.String("").ToStringOutput()'
    )
    content = content.replace(
        "s.privateKey == nil",
        's.privateKey == pulumi.String("").ToStringOutput()'
    )

    with open(ssh_path, 'w') as f:
        f.write(content)
    print("Fixed sshkeys.go")

# Run all fixes
fix_checker()
fix_pulumi_map_conversions()

print("All compilation fixes applied")