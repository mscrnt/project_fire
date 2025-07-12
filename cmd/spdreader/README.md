# SPD Reader

A single-file Windows executable that provides full SPD EEPROM access for every DIMM slot. No external dependencies, no separate installer required.

## Features

- **Single-file executable** - No DLLs or external files needed
- **Embedded signed driver** - SMBus driver bundled within the exe
- **Automatic driver management** - Installs on startup, uninstalls on exit
- **WMI fallback** - Falls back to WMI if driver installation fails
- **Full SPD parsing** - Parses JEDEC SPD-Rev-5 data into rich structures
- **Comprehensive data extraction**:
  - Memory type (DDR4, DDR5, etc.)
  - Capacity, speed, and PC rating
  - JEDEC manufacturer ID
  - Part number and serial number
  - Detailed timing parameters
  - Manufacturing date

## Requirements

- Windows 10/11 (x64)
- Administrator privileges (for driver installation)
- .NET Framework 4.5+ (for WMI fallback)

## Installation

No installation required! Simply download `spdreader.exe` and run it.

## Usage

```powershell
# Run as Administrator

# Display memory modules in table format
spdreader.exe -list

# Output JSON format (for programmatic use)
spdreader.exe

# Show help
spdreader.exe -help

# Enable verbose logging
spdreader.exe -v -list
```

### Example Output

```
Slot   Type     Speed      Size     Ranks  Width    Manufacturer         Part Number      Serial    
----------------------------------------------------------------------------------------------------------
0      DDR4     3200 MT/s  16 GB    2      x64      Samsung              M378A2K43CB1-CTD 12345678  
1      DDR4     3200 MT/s  16 GB    2      x64      Samsung              M378A2K43CB1-CTD 12345679  
2      DDR4     3200 MT/s  16 GB    2      x64      Samsung              M378A2K43CB1-CTD 1234567A  
3      DDR4     3200 MT/s  16 GB    2      x64      Samsung              M378A2K43CB1-CTD 1234567B  
```

## Building from Source

### Prerequisites

- Go 1.20 or later
- Windows SDK (for Windows service APIs)
- Signed SMBus driver files (RWEverything.sys and EWD.dll)

### Build Steps

1. Place the signed driver files in `pkg/spdreader/driver/`:
   - `RWEverything.sys` - The kernel driver
   - `EWD.dll` - The driver interface DLL

2. Run the build script:
   ```powershell
   .\build_spdreader.ps1
   ```

3. The output will be a single `spdreader.exe` file.

## How It Works

1. **Driver Embedding**: The SMBus driver files are embedded into the Go binary using `//go:embed` directives.

2. **Runtime Extraction**: On startup, the driver files are extracted to a temporary directory.

3. **Service Installation**: The driver is installed as a Windows kernel service using the Service Control Manager API.

4. **SMBus Communication**: The tool uses the driver's API to communicate with the SMBus controller and read SPD data from memory modules.

5. **SPD Parsing**: Raw SPD bytes are parsed according to JEDEC specifications to extract meaningful information.

6. **Cleanup**: On exit (normal or interrupt), the driver service is stopped, deleted, and temporary files are removed.

## API Usage

The SPD reader can be used as a library in other Go applications:

```go
import "github.com/mscrnt/project_fire/pkg/spdreader"

func main() {
    reader, err := spdreader.New()
    if err != nil {
        log.Fatal(err)
    }
    defer reader.Close()

    modules, err := reader.ReadAllModules()
    if err != nil {
        log.Fatal(err)
    }

    for _, module := range modules {
        fmt.Printf("Slot %d: %s %d MT/s %.0f GB\n", 
            module.Slot, module.Type, module.DataRateMTs, module.CapacityGB)
    }
}
```

## Supported Memory Types

- DDR4 SDRAM
- DDR4E SDRAM
- DDR5 SDRAM
- LPDDR4
- LPDDR4X
- LPDDR5

## Technical Details

### SPD Data Structure

The tool reads and parses SPD (Serial Presence Detect) data according to:
- JEDEC Standard No. 21-C (DDR4 SPD)
- JEDEC Standard No. 21-C (DDR5 SPD)

### SMBus Addresses

Memory modules are typically found at SMBus addresses 0x50-0x57, corresponding to DIMM slots 0-7.

### Driver Requirements

The embedded driver must be signed with a valid kernel code signing certificate for Windows to load it. The tool supports:
- RWEverything driver
- CPU-Z driver (as fallback)

## Troubleshooting

### "Access Denied" Error

Run the tool as Administrator. Right-click on `spdreader.exe` and select "Run as administrator".

### "Driver Installation Failed"

1. Ensure you're running as Administrator
2. Check Windows Event Log for driver-related errors
3. Verify the embedded driver is properly signed
4. Try disabling antivirus temporarily
5. The tool will automatically fall back to WMI if driver installation fails

### "No Memory Modules Found"

1. Ensure memory is properly installed
2. Try the `-v` flag for verbose output
3. Check if WMI service is running: `sc query winmgmt`

### Antivirus Detection

Some antivirus software may flag the tool due to driver installation. This is a false positive. You can:
1. Add an exception for `spdreader.exe`
2. Use the WMI-only mode (automatic if driver fails)

## Security Considerations

- The tool requires Administrator privileges to install the kernel driver
- Driver files are extracted to a secure temporary directory
- All temporary files are cleaned up on exit
- The driver service is removed after use
- No permanent system modifications are made

## License

This tool is part of the FIRE benchmarking suite. See the main project LICENSE file for details.

## Contributing

Contributions are welcome! Please ensure:
- Code follows Go best practices
- Tests pass (`go test ./pkg/spdreader/...`)
- No memory leaks or resource leaks
- Proper error handling and cleanup