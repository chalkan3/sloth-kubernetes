#!/bin/bash

echo "=========================================="
echo "Get WireGuard Server Information"
echo "=========================================="
echo ""
echo "This script will help you get the WireGuard configuration"
echo "from your AWS server."
echo ""

# Method 1: Direct SSH
echo "Method 1: Direct SSH to your WireGuard server"
echo "----------------------------------------------"
echo "If you know your server IP, try:"
echo ""
echo "  ssh -i ~/.ssh/aws-sloth-runner-01 ubuntu@YOUR_SERVER_IP 'sudo cat /etc/wireguard/publickey'"
echo "  OR"
echo "  ssh -i ~/.ssh/aws-sloth-runner-01 ec2-user@YOUR_SERVER_IP 'sudo cat /etc/wireguard/publickey'"
echo ""

# Method 2: Using AWS CLI
echo "Method 2: Using AWS CLI"
echo "------------------------"
echo "Find your WireGuard server:"
echo ""

if command -v aws &> /dev/null; then
    echo "Checking AWS for running instances..."

    # Try different profiles
    for profile in default sloth-runner sloth minio; do
        echo ""
        echo "Trying profile: $profile"
        aws ec2 describe-instances --profile $profile \
            --query "Reservations[*].Instances[?State.Name=='running'].[InstanceId,PublicIpAddress,PrivateIpAddress,Tags[?Key=='Name'].Value|[0]]" \
            --output table 2>/dev/null && break
    done

    echo ""
    echo "If you see your WireGuard server above, note its IP address."
else
    echo "AWS CLI not installed."
fi

echo ""
echo "Method 3: Manual configuration"
echo "-------------------------------"
echo "If you have access to your WireGuard server through other means:"
echo ""
echo "1. Connect to your WireGuard server"
echo "2. Run: sudo cat /etc/wireguard/publickey"
echo "3. Run: sudo wg show wg0"
echo "4. Note the public key and listening port"
echo ""
echo "Then set these environment variables:"
echo ""
echo "  export WG_SERVER_ENDPOINT='YOUR_SERVER_IP:51820'"
echo "  export WG_SERVER_PUBLIC_KEY='YOUR_PUBLIC_KEY'"
echo ""
echo "=========================================="