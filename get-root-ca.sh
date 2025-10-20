#!/bin/bash
# Script to retrieve the mkcert root CA from the keite-guica server
# This will properly configure SSL verification for the Bitwarden CLI

set -e

SERVER="chalkan3@10.8.0.6"
TARGET="/tmp/keite-guica-rootCA.pem"

echo "Retrieving root CA from $SERVER..."

# Try to get the root CA from the server via SSH
if ssh "$SERVER" "test -d ~/.local/share/mkcert" 2>/dev/null; then
    echo "Using ~/.local/share/mkcert on remote server"
    ssh "$SERVER" "cat ~/.local/share/mkcert/rootCA.pem" > "$TARGET"
elif ssh "$SERVER" "test -d \$HOME/Library/Application\ Support/mkcert" 2>/dev/null; then
    echo "Using Library/Application Support/mkcert on remote server"
    ssh "$SERVER" "cat \"\$HOME/Library/Application Support/mkcert/rootCA.pem\"" > "$TARGET"
else
    echo "Trying to find mkcert root CA on remote server..."
    ssh "$SERVER" "mkcert -CAROOT && cat \$(mkcert -CAROOT)/rootCA.pem" > "$TARGET"
fi

if [ -s "$TARGET" ]; then
    echo "✅ Root CA successfully retrieved and saved to $TARGET"
    echo ""
    echo "Verifying certificate..."
    openssl x509 -in "$TARGET" -text -noout | grep -E "(Issuer|Subject|Not Before|Not After)" || true
    echo ""
    echo "Now you can update your ~/.zshrc to use NODE_EXTRA_CA_CERTS instead of NODE_TLS_REJECT_UNAUTHORIZED"
    echo "The current configuration will work, but disabling SSL verification is not recommended for production."
else
    echo "❌ Failed to retrieve root CA"
    exit 1
fi
