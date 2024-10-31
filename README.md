# 0-OS ![Tests](https://github.com/threefoldtech/zos4/workflows/Tests%20and%20Coverage/badge.svg) [![Go Report Card](https://goreportcard.com/badge/github.com/threefoldtech/zos)](https://goreportcard.com/report/github.com/threefoldtech/zos)

0-OS is an autonomous operating system design to expose raw compute, storage and network capacity.

This repository host the V2 of 0-OS which is a complete rewrite from scratch. If you want to know about the history and decision that motivated the creation of the V2, you can read [this article](docs/internals/history/readme.md)

0-OS is mainly used to run node on the Threefold Grid.
Head to https://threefold.io and https://wiki.threefold.io to learn more about Threefold and the grid.

## Documentation

Start exploring the code base by first checking the [documentation](/docs) and [specification documents](/specs).

An [FAQ](./docs/faq/readme.md) is also available for all the common questions.

## Setting up your development environment

If you want to contribute read the [contribution guideline](CONTRIBUTING.md) and the documentation to setup your [development environment](qemu/README.md)

## Grid Networks

0-OS is deployed on 3 different "flavor" of network:

- **production network**: Released of stable version. Used to run the real grid with real money. Cannot be reset ever. Only stable and battle tested feature reach this level. You can find the [dashboard here](https://dashboard.grid.tf/)
- **test network**: Mostly stable features that need to be tested at scale, allow preview and test of new features. Always the latest and greatest. This network can be reset sometimes, but should be relatively stable. You can find the [dashboard here](https://dashboard.test.grid.tf/)
- **QA network**: Mostly unstable features that need to be tested internally, allow preview and test of new features. Can be behind development. This network can be reset sometimes, but should be relatively stable. You can find the [dashboard here](https://dashboard.qa.grid.tf/)
- **dev network**: ephemeral network only setup to develop and test new features. Can be created and reset at anytime. You can find the [dashboard here](https://dashboard.dev.grid.tf/)

Learn more about the different network by reading the [upgrade documentation](/docs/internals/identity/upgrade.md#philosophy)

### Provisioning of workloads

ZOS does not expose an interface, instead of wait for reservation to happen on a trusted
source, and once this reservation is available, the node will actually apply it to reality. You can start reading about [provisioning](./docs/provision) in this document.

## Owners

[@maxux](https://github.com/maxux) [@muhamadazmy](https://github.com/muhamadazmy) [@delandtj](https://github.com/delandtj) [@leesmet](https://github.com/leesmet)

## Community

If you have some questions or just want to hang out, you can find us on:
- telegram: https://t.me/zero_os_tech
- Matrix: #zero-os:matrix.org
