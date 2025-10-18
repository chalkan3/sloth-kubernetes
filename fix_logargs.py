#!/usr/bin/env python3

import re
import os

def fix_logargs(filepath):
    """Fix LogArgs usage in Go files."""
    with open(filepath, 'r') as f:
        content = f.read()

    original_content = content

    # Pattern 1: Fix LogArgs with Message field
    # Match: Log.Info("text", &pulumi.LogArgs{Message: pulumi.Sprintf("...")})
    # Replace with: Log.Info("text", nil)
    pattern1 = r'(Log\.\w+\([^,]+,\s*)&pulumi\.LogArgs\{\s*Message:\s*(pulumi\.Sprintf\([^)]+\))\s*\}'
    content = re.sub(pattern1, r'\1nil', content)

    # Pattern 2: If we need to keep the Sprintf, change the format
    # This is for cases where Message is important
    pattern2 = r'(ctx\.Log\.\w+)\(([^,]+),\s*&pulumi\.LogArgs\{[^}]*Message:\s*(pulumi\.Sprintf\([^)]+\))[^}]*\}\)'
    def replace_message(match):
        method = match.group(1)
        msg = match.group(2)
        sprintf = match.group(3)
        # Remove the LogArgs and just pass nil
        return f'{method}({msg}, nil)'

    content = re.sub(pattern2, replace_message, content)

    # Save if changed
    if content != original_content:
        with open(filepath, 'w') as f:
            f.write(content)
        print(f"Fixed: {filepath}")
        return True
    return False

# Fix all Go files in pkg/
go_files = [
    "pkg/dns/manager.go",
    "pkg/health/checker.go",
    "pkg/health/validator.go",
    "pkg/network/manager.go",
    "pkg/network/vpn_connectivity.go",
    "pkg/security/os_firewall.go",
    "pkg/security/sshkeys.go",
]

fixed_count = 0
for filepath in go_files:
    full_path = f"/Users/chalkan3/.projects/do-droplet-create/{filepath}"
    if os.path.exists(full_path):
        if fix_logargs(full_path):
            fixed_count += 1

print(f"Fixed {fixed_count} files")