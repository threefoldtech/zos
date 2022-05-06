# Provision Module

## ZBus

This module is autonomous module and is not reachable over zbus.

## Introduction

This module is responsible to provision/decommission workload on the node.

The provision module constantly watch for new reservation address to it. Upon retrieval of a new reservation the node will automatically try to provision the deployment and update consumption on the grid.

## Supported workload

0-OS currently support 8 type of workloads:
- network
- zmachine (vm)
- zmount (disk): usable only by a zmachine
- public-ip (v4 and/or v6): usable only by a zmachine
- [zdb](https://github.com/threefoldtech/0-DB) namespace
- [qsfs](https://github.com/threefoldtech/quantum-storage)
- zlogs
- gateway

Check the [provision.md](provision.md) file to see the expected reservation
schema for each type of workload

## Provisioning flows

See the [IT contract documentation](it_contract.md)

## Payment of the reservations

See the [reservation payment documentation](reservation_payment.md)
