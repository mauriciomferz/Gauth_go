# Contributing to GAuth

We love your input! We want to make contributing to GAuth as easy and transparent as possible, whether it's:

- Reporting a bug
- Discussing the current state of the code
- Submitting a fix
- Proposing new features
- Becoming a maintainer

## Start Here: Onboarding & Documentation

Before contributing, please review our onboarding and documentation resources:

- [README.md](./README.md): Project overview, goals, and quickstart
- [GETTING_STARTED.md](./GETTING_STARTED.md): Step-by-step onboarding for new contributors
- [LIBRARY.md](./LIBRARY.md): Library usage, API reference, and modular structure

These documents will help you understand the modular architecture, demo/library separation, and best practices for contributing.

## We Develop with Github
We use Github to host code, to track issues and feature requests, as well as accept pull requests.

## We Use [Github Flow](https://guides.github.com/introduction/flow/index.html)
All code changes happen through pull requests. Pull requests are the best way to propose changes to the codebase.

1. Fork the repo and create your branch from `main`
2. If you've added code that should be tested, add tests
3. If you've changed APIs, update the documentation
4. Ensure the test suite passes
5. Make sure your code lints
6. Issue that pull request!

## Development Process

1. Clone the repository:
   ```bash
   git clone https://github.com/Gimel-Foundation/gauth.git
   cd gauth
   ```

2. Run the cleanup script to fix common issues:
   ```bash
   chmod +x cleanup.sh
   ./cleanup.sh
   ```

## Code Organization

GAuth follows a standard Go project layout:

```
/cmd         - Main applications
/pkg         - Public libraries that can be imported by other projects
/internal    - Private code not intended for external use
/examples    - Example applications demonstrating usage
/test        - Testing utilities and benchmarks
/docs        - Documentation
```

### Modular Structure & Demo/Library Separation

- **Library code** lives in `/pkg` and is designed for type safety, modularity, and reusability.
- **Demo and example code** is in `/examples` and `/cmd/demo`, and should not be mixed with core library code.
- All onboarding and usage examples reference the demo code for clarity.
```

## Package Structure

- **pkg/token**: Core token management functionality (our primary package)
  - Use the main implementations in this package directly (e.g., `MemoryStore`)
  - Legacy code in sub-packages is kept for backward compatibility
  
- **pkg/auth**: Authentication mechanisms and protocols
- **pkg/authz**: Authorization and permission management
- **pkg/events**: Event handling system
- **pkg/rate**: Rate limiting functionality
- **pkg/resilience**: Resilience patterns and circuit breakers
- **pkg/store**: Storage implementations

## Common Issues to Avoid

1. **Duplicate Implementations**: Don't create multiple versions of the same functionality
2. **Package Declaration Mismatches**: Ensure package declarations match directory names
3. **Circular Dependencies**: Avoid importing packages that import your package
4. **Wrong Error Types**: Use the error types from `pkg/errors` consistently

5. **Type Safety**: All public APIs must be type-safe. Avoid `map[string]interface{}` in exported APIs. See [LIBRARY.md](./LIBRARY.md) for examples.

6. **RFC111 Compliance**: All protocol logic must follow [RFC111](https://datatracker.ietf.org/doc/html/rfc111). See [README.md](./README.md) for compliance mapping.

2. Install development tools:
   ```bash
   make install-tools
   ```

3. Run tests:
   ```bash
   make test
   ```

4. Build the project:
   ```bash
   make build
   ```

## Continuous Integration (CI)

All pull requests are automatically tested using GitHub Actions. See `.github/workflows/go.yml` for details. Please ensure your code passes all CI checks before requesting review.

## Code Style Guidelines

- Follow standard Go coding conventions
- Use `gofmt` to format your code
- Add comments for exported functions and types
- Write meaningful commit messages
- Include tests for new functionality

## Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make coverage

# Run benchmarks
make bench
```

## Documentation

- Keep documentation up to date
- Document all exported functions and types
- Include examples in godoc format
- Update README.md when adding new features
- Reference onboarding docs ([README.md](./README.md), [GETTING_STARTED.md](./GETTING_STARTED.md), [LIBRARY.md](./LIBRARY.md)) for structure and usage patterns

## Pull Request Process

1. Update the README.md with details of changes to the interface
2. Update the CHANGELOG.md with a note describing your changes
3. The PR will be merged once you have the sign-off of two other developers

## Community Standards

- Be respectful and constructive in all discussions
- Follow the [Code of Conduct](./CODE_OF_CONDUCT.md) (if present)

## Any contributions you make will be under the MIT Software License
In short, when you submit code changes, your submissions are understood to be under the same [MIT License](http://choosealicense.com/licenses/mit/) that covers the project. Feel free to contact the maintainers if that's a concern.

## Report bugs using Github's [issue tracker](https://github.com/Gimel-Foundation/gauth/issues)
We use GitHub issues to track public bugs. Report a bug by [opening a new issue](https://github.com/Gimel-Foundation/gauth/issues/new).

## License
By contributing, you agree that your contributions will be licensed under its MIT License.