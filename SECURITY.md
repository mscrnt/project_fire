# Security Policy

## Supported Versions

We release patches for security vulnerabilities. Which versions are eligible for receiving such patches depends on the CVSS v3.0 Rating:

| Version | Supported          |
| ------- | ------------------ |
| 0.2.x   | :white_check_mark: |
| 0.1.x   | :x:                |

## Reporting a Vulnerability

If you discover a security vulnerability within FIRE, please send an email to the maintainers. All security vulnerabilities will be promptly addressed.

Please do **NOT** create a public GitHub issue for security vulnerabilities.

### What to Include

When reporting a vulnerability, please include:

* The version of FIRE affected
* A description of the vulnerability
* Steps to reproduce the issue
* Potential impact of the vulnerability
* Any suggested fixes (if you have them)

### What to Expect

* **Acknowledgment**: We'll acknowledge receipt of your vulnerability report within 48 hours
* **Communication**: We'll keep you informed about the progress of fixing the vulnerability
* **Fix Timeline**: We aim to release patches for critical vulnerabilities within 7 days
* **Credit**: We'll credit you for the discovery (unless you prefer to remain anonymous)

## Security Best Practices for Users

### Running FIRE Safely

1. **Permissions**: Only run FIRE with Administrator/root privileges when necessary
2. **Network**: Be cautious when using the remote agent feature
   * Use strong certificates for mTLS
   * Don't expose the agent port to the internet
   * Use a firewall to restrict access
3. **Telemetry**: Review what data is sent (see TELEMETRY.md)
   * Disable telemetry if required by your security policy
   * No personal data is collected, but verify this meets your requirements

### Certificate Management

When using the remote agent with mTLS:

```bash
# Generate strong certificates
bench cert generate --key-size 4096

# Protect certificate files
chmod 600 certs/*
```

### Data Storage

* Test results are stored in a local SQLite database
* By default located at `~/.fire/fire.db`
* Ensure appropriate file permissions on the database
* No encryption is applied by default

### Log Files

* Logs may contain system information
* Rotate and secure log files appropriately
* Default locations:
  * CLI: `./fire.log`
  * GUI: `./fire-gui.log`
  * Debug: `./logs/`

## Security Features

FIRE includes several security features:

* **mTLS Authentication**: Remote agent requires mutual TLS authentication
* **Input Validation**: All user inputs are validated
* **No External Dependencies**: Minimal attack surface
* **Telemetry Opt-out**: Can be completely disabled
* **Local-only by Default**: No network access unless explicitly configured

## Disclosure Policy

* Security issues are disclosed after a fix is available
* We'll coordinate disclosure with affected parties
* A security advisory will be published on GitHub

## Comments on this Policy

If you have suggestions on how this process could be improved, please submit a pull request.