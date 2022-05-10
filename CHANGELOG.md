# Changelog

This monorepo adheres to [Semantic Versioning](https://semver.org/). All projects found in this monorepo are individually versioned and released. Please consult the relevant changelog:

- [gauntlet-terra](./packages-ts/gauntlet-terra/CHANGELOG.md)
- [gauntlet-terra-contracts](./packages-ts/gauntlet-terra-contracts/CHANGELOG.md)
- [gauntlet-terra-cw-plus](./packages-ts/gauntlet-terra-cw-plus/CHANGELOG.md)

Project releases can be found here: https://github.com/smartcontractkit/chainlink-terra/releases

## Guidelines

All release headers in a changelog must contain the version number followed by the release date.

i.e. v1.0.0 - 2022/05/10

Each changelog must have an 'Unreleased' section, which tracks changes that have yet to be released. Upon release, these changes should be moved to a new release section.

The latest release must be at the top of the changelog file, directly under 'Unreleased'.

Changes should be grouped by type:
- `Added` for new features
- `Changed` for changes in existing functionality (note any breaking changes)
- `Deprecated` for soon-to-be removed features
- `Removed` for now removed features
- `Fixed` for any bug fixes
- `Security` in case of vulnerabilities
