# Contributing to FIRE

First off, thank you for considering contributing to FIRE! It's people like you that make FIRE such a great tool.

## Code of Conduct

This project and everyone participating in it is governed by our [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check existing issues as you might find out that you don't need to create one. When you are creating a bug report, please include as many details as possible:

* **Use a clear and descriptive title**
* **Describe the exact steps which reproduce the problem**
* **Provide specific examples to demonstrate the steps**
* **Describe the behavior you observed after following the steps**
* **Explain which behavior you expected to see instead and why**
* **Include system information:**
  * OS and version
  * Hardware specs (CPU, GPU, RAM)
  * FIRE version (`bench version` or GUI About dialog)
  * Whether you're running as Administrator/root

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion, please include:

* **Use a clear and descriptive title**
* **Provide a step-by-step description of the suggested enhancement**
* **Provide specific examples to demonstrate the steps**
* **Describe the current behavior and explain which behavior you expected to see instead**
* **Explain why this enhancement would be useful**

### Pull Requests

Please follow these steps to have your contribution considered:

1. **Fork the repo and create your branch from `main`**
2. **If you've added code that should be tested, add tests**
3. **If you've changed APIs, update the documentation**
4. **Ensure the test suite passes** (`go test ./...`)
5. **Make sure your code follows the Go style guidelines** (`go fmt ./...`)
6. **Make sure your code passes linting** (`golangci-lint run`)
7. **Issue that pull request!**

## Development Setup

### Prerequisites

* Go 1.23 or later
* Git
* For GUI development:
  * Windows: CGO support (MinGW-w64 or Visual Studio)
  * Linux: GTK3 development packages
  * macOS: Xcode Command Line Tools

### Building from Source

```bash
# Clone the repository
git clone https://github.com/mscrnt/project_fire.git
cd project_fire

# Download dependencies
go mod download

# Build CLI
go build -o bench ./cmd/fire

# Build GUI (requires CGO)
go build -o fire-gui ./cmd/fire-gui

# Run tests
go test -v ./...

# Run linter
golangci-lint run --timeout=5m
```

### Project Structure

```
project_fire/
├── cmd/              # Entry points
│   ├── fire/         # CLI application
│   └── fire-gui/     # GUI application
├── pkg/              # Reusable packages
│   ├── db/           # Database layer
│   ├── gui/          # GUI components
│   ├── plugin/       # Test plugins
│   ├── report/       # Report generation
│   ├── agent/        # Remote agent
│   └── telemetry/    # Telemetry system
├── internal/         # Internal packages
├── templates/        # HTML report templates
├── docs/             # Documentation
└── tests/            # Integration tests
```

## Coding Guidelines

### General Guidelines

* Write clear, idiomatic Go code
* Follow the [Effective Go](https://golang.org/doc/effective_go.html) guidelines
* Keep functions small and focused
* Write descriptive commit messages
* Add comments for complex logic
* Use meaningful variable and function names

### Testing

* Write unit tests for new functionality
* Ensure tests are deterministic and don't depend on external services
* Use table-driven tests where appropriate
* Mock external dependencies
* Aim for >80% code coverage for new code

### Documentation

* Update README.md if you change functionality
* Add godoc comments to all exported functions and types
* Include examples in documentation where helpful
* Update CHANGELOG.md with notable changes

## Adding New Test Plugins

To add a new test plugin:

1. Create a new package in `pkg/plugin/yourtest/`
2. Implement the `TestPlugin` interface
3. Register your plugin in `init()`
4. Add documentation in the plugin file
5. Create unit tests
6. Update the available tests documentation

Example:
```go
package yourtest

import "github.com/mscrnt/project_fire/pkg/plugin"

func init() {
    plugin.Register("yourtest", &YourTestPlugin{})
}

type YourTestPlugin struct{}

func (p *YourTestPlugin) Info() plugin.TestInfo {
    return plugin.TestInfo{
        Name:        "Your Test",
        Description: "Description of what your test does",
        Category:    plugin.CategoryCPU, // or Memory, Storage, etc.
    }
}

// Implement other required methods...
```

## Telemetry and Privacy

When contributing features that collect data:

* Ensure all data collection is anonymous
* Never collect personal information
* Document what data is collected
* Respect the telemetry opt-out flag
* Update TELEMETRY.md if adding new telemetry

## Questions?

Feel free to open an issue with your question or reach out to the maintainers.

Thank you for contributing to FIRE! 🔥