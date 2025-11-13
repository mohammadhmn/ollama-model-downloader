# Contributing to Ollama Model Downloader

Thank you for your interest in contributing! This document provides guidelines for contributing to this project.

## Development Setup

1. **Prerequisites**
   - Go 1.21 or higher
   - Git

2. **Clone the repository**
   ```bash
   git clone https://github.com/your-username/ollama-model-downloader.git
   cd ollama-model-downloader
   ```

3. **Install dependencies**
   ```bash
   make deps
   ```

4. **Run the application**
   ```bash
   make run
   ```

## Code Style

- Follow Go conventions and best practices
- Use `gofmt` to format code
- Run `make lint` to check for issues
- Write meaningful commit messages

## Testing

- Run tests with `make test`
- Run tests with coverage with `make test-coverage`
- Write tests for new features

## Project Structure

```
ollama-model-downloader/
├── cmd/                 # Command-line interface
├── config/              # Configuration management
├── internal/            # Internal packages
│   ├── errors/         # Error handling
│   └── registry/       # Registry client
├── models/             # Data models
├── web/               # Web interface
│   ├── handlers/       # HTTP handlers
│   └── templates/      # HTML templates
├── templates/          # Embedded templates
├── Dockerfile         # Docker configuration
├── Makefile          # Build automation
└── README.md         # Project documentation
```

## Submitting Changes

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Reporting Issues

- Use GitHub Issues for bug reports
- Provide detailed information about the issue
- Include steps to reproduce
- Specify your environment (OS, Go version, etc.)

## License

By contributing, you agree that your contributions will be licensed under the same license as the project.