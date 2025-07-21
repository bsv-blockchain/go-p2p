# 🚀 go-p2p

<table>
  <thead>
    <tr>
      <th>CI&nbsp;/&nbsp;CD</th>
      <th>Quality&nbsp;&amp;&nbsp;Security</th>
      <th>Docs&nbsp;&amp;&nbsp;Meta</th>
      <th>Community</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td valign="top" align="left">
        <a href="https://github.com/bsv-blockchain/go-p2p/releases">
          <img src="https://img.shields.io/github/release-pre/bsv-blockchain/go-p2p?logo=github&style=flat" alt="Latest Release">
        </a><br/>
        <a href="https://github.com/bsv-blockchain/go-p2p/actions">
          <img src="https://img.shields.io/github/actions/workflow/status/bsv-blockchain/go-p2p/fortress.yml?branch=master&logo=github&style=flat" alt="Build Status">
        </a><br/>
		<a href="https://github.com/bsv-blockchain/go-p2p/actions">
          <img src="https://github.com/bsv-blockchain/go-p2p/actions/workflows/codeql-analysis.yml/badge.svg?style=flat" alt="CodeQL">
        </a><br/>
		<a href="https://sonarcloud.io/project/overview?id=bsv-blockchain_go-p2p">
          <img src="https://sonarcloud.io/api/project_badges/measure?project=bsv-blockchain_go-p2p&metric=alert_status&style-flat" alt="SonarCloud">
        </a>
      </td>
      <td valign="top" align="left">
        <a href="https://goreportcard.com/report/github.com/bsv-blockchain/go-p2p">
          <img src="https://goreportcard.com/badge/github.com/bsv-blockchain/go-p2p?style=flat" alt="Go Report Card">
        </a><br/>
		<a href="https://codecov.io/gh/bsv-blockchain/go-p2p/tree/master">
          <img src="https://codecov.io/gh/bsv-blockchain/go-p2p/branch/master/graph/badge.svg?style=flat" alt="Code Coverage">
        </a><br/>
		<a href="https://scorecard.dev/viewer/?uri=github.com/bsv-blockchain/go-p2p">
          <img src="https://api.scorecard.dev/projects/github.com/bsv-blockchain/go-p2p/badge?logo=springsecurity&logoColor=white" alt="OpenSSF Scorecard">
        </a><br/>
		<a href=".github/SECURITY.md">
          <img src="https://img.shields.io/badge/security-policy-blue?style=flat&logo=springsecurity&logoColor=white" alt="Security policy">
        </a>
      </td>
      <td valign="top" align="left">
        <a href="https://golang.org/">
          <img src="https://img.shields.io/github/go-mod/go-version/bsv-blockchain/go-p2p?style=flat" alt="Go version">
        </a><br/>
        <a href="https://pkg.go.dev/github.com/bsv-blockchain/go-p2p?tab=doc">
          <img src="https://pkg.go.dev/badge/github.com/bsv-blockchain/go-p2p.svg?style=flat" alt="Go docs">
        </a><br/>
        <a href=".github/AGENTS.md">
          <img src="https://img.shields.io/badge/AGENTS.md-found-40b814?style=flat&logo=openai" alt="AGENTS.md rules">
        </a><br/>
        <a href="Makefile">
          <img src="https://img.shields.io/badge/Makefile-supported-brightgreen?style=flat&logo=probot&logoColor=white" alt="Makefile Supported">
        </a><br/>
		<a href=".github/dependabot.yml">
          <img src="https://img.shields.io/badge/dependencies-automatic-blue?logo=dependabot&style=flat" alt="Dependabot">
        </a>
      </td>
      <td valign="top" align="left">
        <a href="https://github.com/bsv-blockchain/go-p2p/graphs/contributors">
          <img src="https://img.shields.io/github/contributors/bsv-blockchain/go-p2p?style=flat&logo=contentful&logoColor=white" alt="Contributors">
        </a><br/>
		<a href="https://github.com/bsv-blockchain/go-p2p/commits/master">
		  <img src="https://img.shields.io/github/last-commit/bsv-blockchain/go-p2p?style=flat&logo=clockify&logoColor=white" alt="Last commit">
		</a><br/>
        <a href="https://github.com/sponsors/bsv-blockchain">
          <img src="https://img.shields.io/badge/sponsor-BSV-181717.svg?logo=github&style=flat" alt="Sponsor">
        </a><br/>
      </td>
    </tr>
  </tbody>
</table>

<br/>

## 🗂️ Table of Contents
* [What's Inside](#-whats-inside)
* [Installation](#-installation)
* [Documentation](#-documentation)
* [Examples & Tests](#-examples--tests)
* [Benchmarks](#-benchmarks)
* [Code Standards](#-code-standards)
* [AI Compliance](#-ai-compliance)
* [Maintainers](#-maintainers)
* [Contributing](#-contributing)
* [License](#-license)

<br/>

## 🧩 What's Inside

**go-p2p** is a comprehensive peer-to-peer networking library for BSV blockchain nodes, built on top of [libp2p](https://libp2p.io/). It provides robust networking capabilities for distributed blockchain applications with support for both public and private networks.

### Core Components

- **P2PNode** - The main networking node implementation that manages peer connections, message routing, and network discovery
- **P2PNodeI** - Interface defining the complete P2P networking API for testability and modularity  
- **P2PConfig** - Configuration structure for network parameters, bootstrap peers, and security settings
- **Handler** - Function type for processing incoming topic messages in pub/sub communications

### Key Features

- **Peer Discovery** - Automatic peer discovery using Kademlia DHT (Distributed Hash Table)
- **Pub/Sub Messaging** - Topic-based message distribution using GossipSub protocol
- **Direct Peer Communication** - Stream-based messaging for direct peer-to-peer communication
- **Private Networks** - Support for isolated private networks using pre-shared keys
- **Connection Management** - Automatic connection handling, retry logic, and peer lifecycle management
- **Metrics & Monitoring** - Built-in bandwidth tracking, peer statistics, and connection monitoring
- **Security** - Message signing/verification, peer authentication, and encrypted communications

### Network Modes

- **Public Network** - Connect to the global IPFS DHT for open peer discovery
- **Private Network** - Create isolated networks using shared keys for controlled environments
- **Hybrid Mode** - Static peer connections combined with dynamic discovery

## 📦 Installation

**go-p2p** requires a [supported release of Go](https://golang.org/doc/devel/release.html#policy).
```shell script
go get -u github.com/bsv-blockchain/go-p2p
```

<br/>

## 📚 Documentation

- **API Reference** – Dive into the godocs at [pkg.go.dev/github.com/bsv-blockchain/go-p2p](https://pkg.go.dev/github.com/bsv-blockchain/go-p2p)
- **Usage Examples** – Browse practical patterns in the [examples directory](examples) showing how to create and configure P2P nodes
- **Interface Documentation** – Review the [P2PNodeI interface](interface.go) for the complete API specification
- **Core Implementation** – Examine the main [P2P node implementation](node.go) and [configuration types](types.go)

> **Good to know:** `go-p2p` ships with *zero* runtime dependencies.  
> The only external package we use is `testify`—and that's strictly for tests.

<br/>

<details>
<summary><strong><code>Repository Features</code></strong></summary>
<br/>

* **Continuous Integration on Autopilot** with [GitHub Actions](https://github.com/features/actions) – every push is built, tested, and reported in minutes.
* **Pull‑Request Flow That Merges Itself** thanks to [auto‑merge](.github/workflows/auto-merge-on-approval.yml) and hands‑free [Dependabot auto‑merge](.github/workflows/dependabot-auto-merge.yml).
* **One‑Command Builds** powered by battle‑tested [Make](https://www.gnu.org/software/make) targets for linting, testing, releases, and more.
* **First‑Class Dependency Management** using native [Go Modules](https://github.com/golang/go/wiki/Modules).
* **Uniform Code Style** via [gofumpt](https://github.com/mvdan/gofumpt) plus zero‑noise linting with [golangci‑lint](https://github.com/golangci/golangci-lint).
* **Confidence‑Boosting Tests** with [testify](https://github.com/stretchr/testify), the Go [race detector](https://blog.golang.org/race-detector), crystal‑clear [HTML coverage](https://blog.golang.org/cover) snapshots, and automatic uploads to [Codecov](https://codecov.io/).
* **Hands‑Free Releases** delivered by [GoReleaser](https://github.com/goreleaser/goreleaser) whenever you create a [new Tag](https://git-scm.com/book/en/v2/Git-Basics-Tagging).
* **Relentless Dependency & Vulnerability Scans** via [Dependabot](https://dependabot.com), [Nancy](https://github.com/sonatype-nexus-community/nancy), and [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck).
* **Security Posture by Default** with [CodeQL](https://docs.github.com/en/github/finding-security-vulnerabilities-and-errors-in-your-code/about-code-scanning), [OpenSSF Scorecard](https://openssf.org), and secret‑leak detection via [gitleaks](https://github.com/gitleaks/gitleaks).
* **Automatic Syndication** to [pkg.go.dev](https://pkg.go.dev/) on every release for instant godoc visibility.
* **Polished Community Experience** using rich templates for [Issues & PRs](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/configuring-issue-templates-for-your-repository).
* **All the Right Meta Files** (`LICENSE`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SUPPORT.md`, `SECURITY.md`) pre‑filled and ready.
* **Code Ownership** clarified through a [CODEOWNERS](.github/CODEOWNERS) file, keeping reviews fast and focused.
* **Zero‑Noise Dev Environments** with tuned editor settings (`.editorconfig`) plus curated *ignore* files for [VS Code](.editorconfig), [Docker](.dockerignore), and [Git](.gitignore).
* **Label Sync Magic**: your repo labels stay in lock‑step with [.github/labels.yml](.github/labels.yml).
* **Friendly First PR Workflow** – newcomers get a warm welcome thanks to a dedicated [workflow](.github/workflows/pull-request-management.yml).
* **Standards‑Compliant Docs** adhering to the [standard‑readme](https://github.com/RichardLitt/standard-readme/blob/master/spec.md) spec.
* **Instant Cloud Workspaces** via [Gitpod](https://gitpod.io/) – spin up a fully configured dev environment with automatic linting and tests.
* **Out‑of‑the‑Box VS Code Happiness** with a preconfigured [Go](https://code.visualstudio.com/docs/languages/go) workspace and [`.vscode`](.vscode) folder with all the right settings.
* **Optional Release Broadcasts** to your community via [Slack](https://slack.com), [Discord](https://discord.com), or [Twitter](https://twitter.com) – plug in your webhook.
* **AI Compliance Playbook** – machine‑readable guidelines ([AGENTS.md](.github/AGENTS.md), [CLAUDE.md](.github/CLAUDE.md), [.cursorrules](.cursorrules), [sweep.yaml](.github/sweep.yaml)) keep ChatGPT, Claude, Cursor & Sweep aligned with your repo’s rules.
* **Pre-commit Hooks for Consistency** powered by [pre-commit](https://pre-commit.com) and the [.pre-commit-config.yaml](.pre-commit-config.yaml) file—run the same formatting, linting, and tests before every commit, just like CI.
* **Automated Hook Updates** keep the [.pre-commit-config.yaml](.pre-commit-config.yaml) current via a weekly [workflow](.github/workflows/update-pre-commit-hooks.yml).
* **DevContainers for Instant Onboarding** – Launch a ready-to-code environment in seconds with [VS Code DevContainers](https://containers.dev/) and the included [.devcontainer.json](.devcontainer.json) config.

</details>

<details>
<summary><strong><code>Library Deployment</code></strong></summary>
<br/>

This project uses [goreleaser](https://github.com/goreleaser/goreleaser) for streamlined binary and library deployment to GitHub. To get started, install it via:

```bash
brew install goreleaser
```

The release process is defined in the [.goreleaser.yml](.goreleaser.yml) configuration file.

To generate a snapshot (non-versioned) release for testing purposes, run:

```bash
make release-snap
```

Then create and push a new Git tag using:

```bash
make tag version=x.y.z
```

This process ensures consistent, repeatable releases with properly versioned artifacts and citation metadata.

</details>

<details>
<summary><strong><code>Makefile Commands</code></strong></summary>
<br/>

View all `makefile` commands

```bash script
make help
```

List of all current commands:

<!-- make-help-start -->
```text
bench                 ## Run all benchmarks in the Go application
build-go              ## Build the Go application (locally)
citation              ## Update version in CITATION.cff (use version=X.Y.Z)
clean-mods            ## Remove all the Go mod cache
coverage              ## Show test coverage
diff                  ## Show git diff and fail if uncommitted changes exist
fumpt                 ## Run fumpt to format Go code
generate              ## Run go generate in the base of the repo
godocs                ## Trigger GoDocs tag sync
govulncheck-install   ## Install govulncheck (pass VERSION= to override)
govulncheck           ## Scan for vulnerabilities
help                  ## Display this help message
install-go            ## Install using go install with specific version
install-releaser      ## Install GoReleaser
install-stdlib        ## Install the Go standard library for the host platform
install-template      ## Kick-start a fresh copy of go-p2p (run once!)
install               ## Install the application binary
lint-version          ## Show the golangci-lint version
lint                  ## Run the golangci-lint application (install if not found)
loc                   ## Total lines of code table
mod-download          ## Download Go module dependencies
mod-tidy              ## Clean up go.mod and go.sum
pre-build             ## Pre-build all packages to warm cache
release-snap          ## Build snapshot binaries
release-test          ## Run release dry-run (no publish)
release               ## Run production release (requires github_token)
tag-remove            ## Remove local and remote tag (use version=X.Y.Z)
tag-update            ## Force-update tag to current commit (use version=X.Y.Z)
tag                   ## Create and push a new tag (use version=X.Y.Z)
test-ci-no-race       ## CI test suite without race detector
test-ci               ## CI test runs tests with race detection and coverage (no lint - handled separately)
test-cover-race       ## Runs unit tests with race detector and outputs coverage
test-cover            ## Unit tests with coverage (no race)
test-fuzz             ## Run fuzz tests only (no unit tests)
test-no-lint          ## Run only tests (no lint)
test-parallel         ## Run tests in parallel (faster for large repos)
test-race             ## Unit tests with race detector (no coverage)
test-short            ## Run tests excluding integration tests (no lint)
test                  ## Default testing uses lint + unit tests (fast)
uninstall             ## Uninstall the Go binary
update-linter         ## Upgrade golangci-lint (macOS only)
update-releaser       ## Reinstall GoReleaser
update                ## Update dependencies
vet-parallel          ## Run go vet in parallel (faster for large repos)
vet                   ## Run go vet only on your module packages
```
<!-- make-help-end -->

</details>

<details>
<summary><strong><code>GitHub Workflows</code></strong></summary>
<br/>


### 🎛️ The Workflow Control Center

All GitHub Actions workflows in this repository are powered by a single configuration file: [**.env.shared**](.github/.env.shared) – your one-stop shop for tweaking CI/CD behavior without touching a single YAML file! 🎯

This magical file controls everything from:
- **🚀 Go version matrix** (test on multiple versions or just one)
- **🏃 Runner selection** (Ubuntu or macOS, your wallet decides)
- **🔬 Feature toggles** (coverage, fuzzing, linting, race detection)
- **🛡️ Security tool versions** (gitleaks, nancy, govulncheck)
- **🤖 Auto-merge behaviors** (how aggressive should the bots be?)
- **🏷️ PR management rules** (size labels, auto-assignment, welcome messages)

> **Pro tip:** Want to disable code coverage? Just flip `ENABLE_CODE_COVERAGE=false` in [.env.shared](.github/.env.shared) and push. No YAML archaeology required!

<br/>

| Workflow Name                                                                      | Description                                                                                                            |
|------------------------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------|
| [auto-merge-on-approval.yml](.github/workflows/auto-merge-on-approval.yml)         | Automatically merges PRs after approval and all required checks, following strict rules.                               |
| [codeql-analysis.yml](.github/workflows/codeql-analysis.yml)                       | Analyzes code for security vulnerabilities using [GitHub CodeQL](https://codeql.github.com/).                          |
| [dependabot-auto-merge.yml](.github/workflows/dependabot-auto-merge.yml)           | Automatically merges [Dependabot](https://github.com/dependabot) PRs that meet all requirements.                       |
| [fortress.yml](.github/workflows/fortress.yml)                                     | Runs the GoFortress security and testing workflow, including linting, testing, releasing, and vulnerability checks.    |
| [pull-request-management.yml](.github/workflows/pull-request-management.yml)       | Labels PRs by branch prefix, assigns a default user if none is assigned, and welcomes new contributors with a comment. |
| [scorecard.yml](.github/workflows/scorecard.yml)                                   | Runs [OpenSSF](https://openssf.org/) Scorecard to assess supply chain security.                                        |
| [stale.yml](.github/workflows/stale-check.yml)                                     | Warns about (and optionally closes) inactive issues and PRs on a schedule or manual trigger.                           |
| [sync-labels.yml](.github/workflows/sync-labels.yml)                               | Keeps GitHub labels in sync with the declarative manifest at [`.github/labels.yml`](./.github/labels.yml).             |
| [update-python-dependencies.yml](.github/workflows/update-python-dependencies.yml) | Updates Python dependencies for pre-commit hooks in the repository.                                                    |
| [update-pre-commit-hooks.yml](.github/workflows/update-pre-commit-hooks.yml)       | Automatically update versions for [pre-commit](https://pre-commit.com/) hooks                                          |

</details>

<details>
<summary><strong><code>Updating Dependencies</code></strong></summary>
<br/>

To update all dependencies (Go modules, linters, and related tools), run:

```bash
make update
```

This command ensures all dependencies are brought up to date in a single step, including Go modules and any tools managed by the Makefile. It is the recommended way to keep your development environment and CI in sync with the latest versions.

</details>

<details>
<summary><strong><code>Pre-commit Hooks</code></strong></summary>
<br/>

Set up the optional [pre-commit](https://pre-commit.com) hooks to run the same formatting, linting, and tests defined in [AGENTS.md](.github/AGENTS.md) before every commit:

```bash
pip install pre-commit
pre-commit install
```

The hooks are configured in [.pre-commit-config.yaml](.pre-commit-config.yaml) and mirror the CI pipeline.

</details>

<br/>

## 🧪 Examples & Tests

All unit tests and [examples](examples) run via [GitHub Actions](https://github.com/bsv-blockchain/go-p2p/actions) and use [Go version 1.24.x](https://go.dev/doc/go1.24). View the [configuration file](.github/workflows/fortress.yml).

Run all tests (fast):

```bash script
make test
```

Run all tests with race detector (slower):
```bash script
make test-race
```

<br/>

## ⚡ Benchmarks

P2P networking benchmarks can be run using:

```bash script
make bench
```

> **Note:** Comprehensive benchmarks for P2P operations (peer discovery, message throughput, connection establishment) are planned for future releases. The current focus is on correctness and stability of the networking implementation.

<br/>

## 🛠️ Code Standards
Read more about this Go project's [code standards](.github/CODE_STANDARDS.md).

<br/>

## 🤖 AI Compliance
This project documents expectations for AI assistants using a few dedicated files:

- [AGENTS.md](.github/AGENTS.md) — canonical rules for coding style, workflows, and pull requests used by [Codex](https://chatgpt.com/codex).
- [CLAUDE.md](.github/CLAUDE.md) — quick checklist for the [Claude](https://www.anthropic.com/product) agent.
- [.cursorrules](.cursorrules) — machine-readable subset of the policies for [Cursor](https://www.cursor.so/) and similar tools.
- [sweep.yaml](.github/sweep.yaml) — rules for [Sweep](https://github.com/sweepai/sweep), a tool for code review and pull request management.

Edit `AGENTS.md` first when adjusting these policies, and keep the other files in sync within the same pull request.

<br/>

## 👥 Maintainers
| [<img src="https://github.com/icellan.png" height="50" alt="Siggi" />](https://github.com/icellan) |
|:--------------------------------------------------------------------------------------------------:|
|                                [Siggi](https://github.com/icellan)                                 |

<br/>

## 🤝 Contributing
View the [contributing guidelines](.github/CONTRIBUTING.md) and please follow the [code of conduct](.github/CODE_OF_CONDUCT.md).

### How can I help?
All kinds of contributions are welcome :raised_hands:!
The most basic way to show your support is to star :star2: the project, or to raise issues :speech_balloon:.
You can also support this project by [becoming a sponsor on GitHub](https://github.com/sponsors/bsv-blockchain) :clap:

[![Stars](https://img.shields.io/github/stars/bsv-blockchain/go-p2p?label=Please%20like%20us&style=social&v=1)](https://github.com/bsv-blockchain/go-p2p/stargazers)

<br/>

## 📝 License

[![License](https://img.shields.io/badge/license-OpenBSV-blue?style=flat&logo=springsecurity&logoColor=white)](LICENSE)
