# mTLS Certificates for F.I.R.E. Agent

This directory contains the mTLS certificates for secure agent communication.

## Certificate Generation

To generate new certificates, run:

```bash
go run scripts/generate-mtls-certs.go
```

This will create:
- `ca.pem` / `ca-key.pem` - Certificate Authority (keep the key secure!)
- `server.pem` / `server-key.pem` - Server certificate for the agent
- `client.pem` / `client-key.pem` - Client certificate for connecting to agents

## Usage

### Start Agent Server
```bash
bench agent serve --cert certs/server.pem --key certs/server-key.pem --ca certs/ca.pem
```

### Connect as Client
```bash
bench agent connect --host <agent-host> --cert certs/client.pem --key certs/client-key.pem --ca certs/ca.pem
```

## Security Notes

- The private keys (`*-key.pem`) should be kept secure
- These are test certificates valid for 1 year (CA is valid for 10 years)
- The server certificate includes localhost and wildcard domains for testing
- Proper Extended Key Usage (EKU) is set:
  - Server: `ExtKeyUsageServerAuth`
  - Client: `ExtKeyUsageClientAuth`

## Certificate Details

All certificates use proper key usage flags for mTLS:

**Server Certificate:**
- KeyUsage: DigitalSignature, KeyEncipherment
- ExtKeyUsage: ServerAuth
- Valid for: localhost, *.local, all IPs

**Client Certificate:**
- KeyUsage: DigitalSignature
- ExtKeyUsage: ClientAuth
- CN: fire-agent-client