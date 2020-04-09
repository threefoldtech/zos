# Explorer

The explorer is the component that is responsible to host all the public information about the nodes running 0-OS, the farms, the users identity and the capacity reservations.

The explorer exposes both a web UI and a REST API. 

- The web UI allows users to discover all the nodes and farms in the grid.
- The API is used by nodes and users.

## Prerequisites

Following commands can be passed to the explorer:

| Command | Description
| --- | --- | --- |
| `--listen` | listen address, default :8080
| `--dbConf` | connection string to mongo database, default mongodb://localhost:27017
| `--name` | database name, default explorer
| `--seed` | Seed of a valid Stellar address that has balance to support running the explorer
| `--network` | Stellar network, default testnet. Values can be (production, testnet)
| `--asset` | Stellar asset to make reservations payment with, default TFT. Assets supported for now: (TFT, FreeTFT)
| `--backupSigners` | Repeatable flag, expects a valid Stellar address. If 3 are provided, multisig on the escrow accounts will be enabled. This is needed if one wishes to recover funds on the escrow accounts.

> If a seed is passed to the explorer, payments for reservation will be enabled.

> The seed passed to the explorer must have balance of the specified asset!

> To recover funds for an escrow account, check following docs: [tools/stellar/readme.md](tools/stellar/readme.md)

