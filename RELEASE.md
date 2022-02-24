# Release Guide

This guide is intended for project maintainers.

Shipper container images are built for every push on `main`, tagged as `ghcr.io/neosperience/shipper:main`.

To release a new stable version, simply create a new Git tag using the [SemVer] form `major.minor.patch`. The appropriate [pipeline](.github/workflows/release.yaml) will take care of building and publishing the artifacts. 

A new draft release will be created in the [releases section](https://github.com/Neosperience/shipper/releases). Edit its description by adding the relevant section from the [changelog](CHANGELOG.md), then publish it.

Don't forget to announce the new version in the [Announcements Discussions section](https://github.com/Neosperience/shipper/discussions/categories/announcements)!