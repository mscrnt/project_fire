# FIRE Telemetry

## Overview

FIRE includes optional telemetry to help improve hardware compatibility and catch crashes. This telemetry is:
- **Anonymous** - No personal information is collected
- **Minimal** - Only hardware compatibility and crash data
- **Opt-out** - Can be disabled with `--telemetry=false`

## What is Collected

### Hardware Compatibility
When FIRE encounters unrecognized hardware, it reports:
- Hardware type code (e.g., "SMBIOSMemoryType: 34")
- Component type (e.g., "unknown_memory_type", "unknown_bus_type")
- OS and architecture (e.g., "windows/amd64")
- App version

### Crash Reports
If FIRE crashes, it reports:
- Stack trace
- Error message
- OS and architecture
- App version

## What is NOT Collected
- User data or personal information
- Hardware serial numbers
- System identifiers
- File paths or filenames
- Test results or benchmarks
- Network information
- Location data

## Disabling Telemetry

### CLI
```bash
bench --telemetry=false [command]
```

### GUI
```bash
fire-gui --telemetry=false
```

### Environment Variable
```bash
export FIRE_TELEMETRY_DISABLED=true
```

## Data Usage

Telemetry data is used to:
1. Add support for new hardware types
2. Fix crashes and improve stability
3. Prioritize hardware compatibility work

## Privacy Commitment

- Data is transmitted over HTTPS
- Data is not sold or shared with third parties
- Data is retained for 90 days
- No tracking cookies or persistent identifiers

## Technical Details

Telemetry endpoint: `https://firelogs.mscrnt.com/fire-logs/`

Data is batched and sent every 30 seconds while the app is running. On crash, data is sent immediately before exit.

Events are stored as JSON files in an S3 bucket with timestamps.

## Questions?

Open an issue on GitHub if you have questions about telemetry.