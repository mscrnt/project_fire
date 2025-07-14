# WinRing0 Setup for Enhanced Memory Detection

F.I.R.E. GUI can use WinRing0 driver for enhanced memory detection, providing the same level of detail as CPU-Z.

## Features with WinRing0

When WinRing0 is available, F.I.R.E. can read SPD (Serial Presence Detect) data directly from memory modules, providing:

- Accurate memory type detection (no more DDR5 showing as LPDDR4X)
- Real manufacturer identification from JEDEC codes
- Actual timing parameters (CAS latency, tRCD, tRP, tRAS)
- XMP/EXPO profile detection
- Manufacturing date and location
- Chip manufacturer detection

## Setup Instructions

1. **Download WinRing0**
   - Visit: https://github.com/QCute/WinRing0/releases
   - Download the latest release (e.g., WinRing0_1_3_1b.zip)

2. **Extract Required Files**
   Place these files in the same directory as `fire-gui.exe`:
   - `OlsApi.dll` (from `WinRing0_1_3_1b\Bin\x64\`)
   - `WinRing0x64.sys` (from `WinRing0_1_3_1b\Bin\x64\`)

3. **Run as Administrator**
   - Right-click `fire-gui.exe`
   - Select "Run as administrator"
   - The WinRing0 driver requires administrator privileges

## Verification

When WinRing0 is properly installed:
- The debug log will show: "Using SPD reader for memory detection"
- Memory details will include more accurate information
- Manufacturing dates and chip manufacturers will be displayed

## Fallback Behavior

If WinRing0 is not available or cannot be initialized:
- F.I.R.E. will automatically fall back to WMI-based detection
- The debug log will show: "Falling back to WMI for memory detection"
- Basic memory information will still be available

## Troubleshooting

1. **"WinRing0 DLL not found"**
   - Ensure OlsApi.dll is in the same directory as fire-gui.exe
   - Check that you have the 64-bit version

2. **"Failed to initialize WinRing0 driver"**
   - Run fire-gui.exe as Administrator
   - Check Windows Defender or antivirus isn't blocking the driver

3. **No enhanced data showing**
   - Verify both DLL and SYS files are present
   - Check the debug logs for specific error messages

## Security Note

WinRing0 is a kernel driver that provides low-level hardware access. Only download it from official sources and be aware that some antivirus software may flag it due to its kernel-level access capabilities.