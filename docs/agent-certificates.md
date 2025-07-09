# Agent Certificate Setup Guide

The F.I.R.E. agent uses mutual TLS (mTLS) authentication to secure communications between the agent server and clients. This guide explains how to generate and manage certificates for the agent.

## Prerequisites

- The F.I.R.E. CA must be initialized: `bench cert init`
- OpenSSL installed on your system

## Certificate Types

The agent requires three types of certificates:

1. **CA Certificate** - Already created with `bench cert init`
2. **Server Certificate** - Used by the agent server
3. **Client Certificate** - Used by clients connecting to the agent

## Generating Certificates

### 1. Locate Your CA

First, find your CA files:

```bash
# Default location
ls ~/.fire/ca/
# Should show: ca.crt and ca.key
```

### 2. Generate Server Certificate

Create a private key and certificate for the agent server:

```bash
# Generate server private key
openssl genrsa -out server.key 2048

# Create certificate signing request
openssl req -new -key server.key -out server.csr \
  -subj "/CN=fire-agent-server/O=F.I.R.E. Agent"

# Sign with CA
openssl x509 -req -in server.csr -CA ~/.fire/ca/ca.crt -CAkey ~/.fire/ca/ca.key \
  -CAcreateserial -out server.pem -days 365 \
  -extensions v3_req -extfile <(echo "[v3_req]
subjectAltName = DNS:localhost,DNS:*.local,IP:127.0.0.1")

# Clean up CSR
rm server.csr
```

### 3. Generate Client Certificate

Create a private key and certificate for agent clients:

```bash
# Generate client private key
openssl genrsa -out client.key 2048

# Create certificate signing request
openssl req -new -key client.key -out client.csr \
  -subj "/CN=fire-agent-client/O=F.I.R.E. Client"

# Sign with CA
openssl x509 -req -in client.csr -CA ~/.fire/ca/ca.crt -CAkey ~/.fire/ca/ca.key \
  -CAcreateserial -out client.pem -days 365

# Clean up CSR
rm client.csr
```

### 4. Secure the Files

Set appropriate permissions:

```bash
# Make keys readable only by owner
chmod 600 server.key client.key

# Make certificates readable
chmod 644 server.pem client.pem
```

## Using Certificates

### Starting the Agent Server

```bash
bench agent serve \
  --cert server.pem \
  --key server.key \
  --ca ~/.fire/ca/ca.crt \
  --port 2223
```

### Connecting to the Agent

```bash
bench agent connect \
  --host 192.168.1.100 \
  --cert client.pem \
  --key client.key \
  --ca ~/.fire/ca/ca.crt \
  --endpoint sysinfo
```

## Environment Variables

To avoid specifying certificates on every command, you can use environment variables:

```bash
# Server environment
export FIRE_AGENT_CERT=server.pem
export FIRE_AGENT_KEY=server.key
export FIRE_AGENT_CA=~/.fire/ca/ca.crt
export FIRE_AGENT_PORT=2223

# Client environment
export FIRE_CLIENT_CERT=client.pem
export FIRE_CLIENT_KEY=client.key
export FIRE_CLIENT_CA=~/.fire/ca/ca.crt
```

## Certificate Distribution

For remote agents, you'll need to:

1. Copy the server certificate and key to the target machine
2. Copy the CA certificate to both server and client machines
3. Keep client certificates on management workstations

Example using SCP:

```bash
# Copy to remote agent
scp server.pem server.key ~/.fire/ca/ca.crt user@remote:/path/to/agent/

# Copy client certs to workstation
scp client.pem client.key ~/.fire/ca/ca.crt user@workstation:/path/to/certs/
```

## Security Best Practices

1. **Protect Private Keys**: Never share or transmit private keys over insecure channels
2. **Use Strong Passphrases**: Consider encrypting private keys with passphrases
3. **Rotate Certificates**: Replace certificates before they expire
4. **Limit Access**: Use file permissions to restrict certificate access
5. **Separate Certificates**: Use different certificates for different agents/clients

## Troubleshooting

### Certificate Verification Failed

- Ensure all certificates are signed by the same CA
- Check certificate dates (not expired)
- Verify certificate CN matches expectations

### Connection Refused

- Check firewall rules for port 2223
- Ensure agent is running with correct certificates
- Verify network connectivity

### Permission Denied

- Check file permissions on certificate and key files
- Ensure the agent process can read the files

## Advanced: Certificate with Subject Alternative Names

For agents that need to be accessed by multiple hostnames:

```bash
# Create config file
cat > server.conf <<EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req

[req_distinguished_name]
CN = fire-agent-server

[v3_req]
keyUsage = keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = agent.local
DNS.3 = *.fire.local
IP.1 = 127.0.0.1
IP.2 = 192.168.1.100
EOF

# Generate certificate with SANs
openssl req -new -key server.key -out server.csr -config server.conf
openssl x509 -req -in server.csr -CA ~/.fire/ca/ca.crt -CAkey ~/.fire/ca/ca.key \
  -CAcreateserial -out server.pem -days 365 -extensions v3_req -extfile server.conf
```

This allows the agent to be accessed using any of the specified hostnames or IP addresses.