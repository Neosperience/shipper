# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.0] - 2022-04-27

### Added

- Support for Gitea as git provider (`--repo-kind gitea`)
- Added support for updating multiple images at once, either specifying `--container-image`, `--container-tag` (and other related tags) multiple times, or by using the environmental variables `CONTAINER_IMAGES` and `CONTAINER_TAGS` with comma-separated values. Multiple files can also be changed with one single run (resulting in a single commit).

### Changed

- Shipper now has some informational logs about its parameters and, if the git provider supports it, a URL to the commit that was created.

## [0.3.0] - 2022-02-28

### Added

- `--no-verify-tls` will make all HTTPS requests skip TLS cerficate validation (eg. for running selfsigned certificates)

### Changed

- Every HTTP request now uses the default Go transport options, most notably timeout is now 90 seconds instead of 30, plus a few extra sanity checks.

## [0.2.0] - 2022-02-28

### Added

- Added support for GitHub as a git provider (`--repo-kind github`), works with both hosted GitHub and GitHub Enteprise Server.

### Changed

- Set default value for `--gitlab-endpoint`/`-gl-url` to GitLab.com's API endpoint

## [0.1.0] - 2022-02-28

### Added

- Support for GitLab as a Git provider
- Support for Bitbucket cloud as a Git provider
- Support for Helm as a templater
- Support for Kustomize as a templater

[unreleased]: https://github.com/neosperience/shipper/compare/0.4.0...HEAD
[0.4.0]: https://github.com/neosperience/shipper/compare/0.3.0...0.4.0
[0.3.0]: https://github.com/neosperience/shipper/compare/0.2.0...0.3.0
[0.2.0]: https://github.com/neosperience/shipper/compare/0.1.0...0.2.0
[0.1.0]: https://github.com/neosperience/shipper/releases/tag/0.1.0
