# Setup and Development

This guide covers the development environment setup and operational commands for the Kerberos project.

## 1. Prerequisites

You can set up the environment automatically using **Nix** (recommended) or manually by installing the required tools.

### Option A: Automated Setup (Nix + Direnv)

This method ensures you have the exact tool versions defined for the project.

1.  **Install Nix**: [Download Nix](https://nixos.org/download.html)
2.  **Install Direnv**: [Download Direnv](https://direnv.net/)
3.  **Hook Direnv**: Add the [hook](https://direnv.net/docs/hook.html) to your shell configuration (e.g., `.zshrc`, `.bashrc`).

**Initialization:**

Navigate to the project root and allow the environment to load:

```bash
cd path/to/kerberos
direnv allow
```

This will automatically download Go, linters, and build tools defined in `flake.nix`.

### Option B: Manual Setup

If you prefer not to use Nix, you must install the following tools manually:

1.  **Go**: Version 1.25+ (Check `go.mod` for the specific version).
2.  **Make**: Required to run project commands.
3.  **golangci-lint**: [Install Guide](https://golangci-lint.run/usage/install/). Required for `make check-quality`.
4.  **(Optional) LaTeX Distribution**: `latexmk` and a TeX distribution (e.g., TeX Live) are required to build the documentation.

## 2. Running the Project

We use a `Makefile` to standardize development tasks.

### Build and Run

To compile and run the application:

```bash
make run
```

_This compiles the binary to `./out/kerberos` and executes it._

### Building Only

To compile without running:

```bash
make build
```

## 3. Testing & Verification

Before submitting changes, ensure your environment is clean and tests pass.

### Run Tests

```bash
make test
```

_Runs unit tests, generates a coverage report, and outputs to `report.json`._

### Quality Checks

```bash
make check-quality
```

_Runs `lint`, `fmt`, and `vet`. Requires `golangci-lint` to be installed._

## 4. Documentation

The documentation is written in LaTeX and managed in the `docs/` directory.

To build and live-preview the documentation:

```bash
cd docs
make run
```

_This uses `latexmk` to watch for changes and auto-update the PDF. Ensure you have a LaTeX environment installed if not using Nix._
