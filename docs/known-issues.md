# Known Issues

## GUI

### Locale Warning in WSL/Minimal Environments
When running the GUI in WSL or minimal Linux environments, you may see:
```
Fyne error:  Error parsing user locale C
  Cause: language: tag is not well-formed
```

This is a harmless warning that occurs when the system locale is set to "C". The GUI will function normally despite this warning.

**Workaround**: Set the LANG environment variable before running:
```bash
LANG=en_US.UTF-8 ./fire-gui
```

Or use the provided wrapper script:
```bash
./scripts/run-gui.sh
```

### X11 Escape Code Warning
You may see:
```
Dropped Escape call with ulEscapeCode : 0x03007703
```

This is a harmless X11 warning that can be safely ignored.