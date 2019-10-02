# 0-OS [![Actions Status](https://github.com/threefoldtech/zos/workflows/build/badge.svg)](https://github.com/threefoldtech/zos/actions) [![Build Status](https://travis-ci.com/threefoldtech/zos.svg?branch=master)](https://travis-ci.com/threefoldtech/zos) [![Go Report Card](https://goreportcard.com/badge/github.com/threefoldtech/zos)](https://goreportcard.com/report/github.com/threefoldtech/zos)

0-OS is an autonomous operating system design to expose raw compute, storage and network capacity.

This repository host the V2 of 0-OS which is a complete rewrite from scratch. If you want to know about the history and decision that motivated the creation of the V2, you can read [this article](docs/history/readme.md)

0-OS is mainly used to run node on the Threefold Grid. Head to <https://threefold.io> to learn more about Threefold and the grid.

## Documentation

Start exploring the code base by first checking the [documentation](/docs) and [specification documents](/specs)

## Setting up your development environment

If you want to contribute read the [contribution guideline](CONTRIBUTING.md) and the documentation to setup your [development environment](qemu/README.md)

## Grid Networks

0-OS is deployed on 3 different "flavor" of network:

- **production network**: Released of stable version. Used to run the real grid with real money. Cannot be reset ever. Only stable and battle tested feature reach this level. (At the time of writhing this network is not live yet)
- **test network**: Mostly stable features that need to be tested at scale, allow preview and test of new features. Always the latest and greatest. This network can be reset sometimes, but should be relatively stable. Uses BCDB hosted at `bcdb.test.grid.tf:8901`
- **dev network**: ephemeral network only setup to develop and test new features. Can be created and reset at anytime. Uses a [mock of BCDB](tools/bcdb_mock) hosted at `https://bcdb.dev.grid.tf`. This mock is also meant for developer to run locally in their development environment.

Learn more about the different network by reading the [upgrade documentation](docs/upgrade/readme.md#philosophy)

## Owners

[@zaibon](https://github.com/zaibon) [@maxux](https://github.com/maxux) [@muhamadazmy](https://github.com/muhamadazmy) [@delandtj](https://github.com/delandtj) [@leesmet](https://github.com/leesmet)

## Meetings

The team holds a update meeting twice a week, on monday and thursday at 10AM.

Zoom URL: https://tinyurl.com/zosupdate
