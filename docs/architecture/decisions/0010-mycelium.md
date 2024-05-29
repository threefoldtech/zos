# 1. Mycelium

Date: 2024-05-29

## Status

Accepted

## Context

Support mycelium network for zdbs and zos host to serve flists

## Decision

Integrate mycelium in zos and allow zos host to have mycelium IPs, serve flists over mycelium, and support mycelium on zdbs

## Consequences

Using mycelium IP is optional. Support will be added to all clients and old zdbs will be migrated to allow mycelium too. 
Old clients should work normally without breakage.
