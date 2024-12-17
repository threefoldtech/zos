# Registrar package

The `registrar` pkg is used to handle node registration on ThreeFold Grid. It is used by the `noded` module, which triggers the registration process by first collecting node capacity information, creating a Redis client, and then adding the `registrar` to the intercommunication process using `zbus` for remote procedure calls (RPC).
re-registration occurs automatically every 24 hours or immediately after an IP address change, ensuring the node's information stays up-to-date.
registration process can fail due to various reasons, such as RPC calls failure.

The registration process includes:

1. Collecting node information
2. Creating/Ensuring a twinID exists for the node
3. Registering the node on the blockchain

## Error Handling

`ErrInProgress`: Error raised if the node registration is still in progress.

`ErrFailed`: Error raised if the node registration fails.

## Constants

### Node registration state constants

`Failed`: Node registration failed

`InProgress`: Node registration is in progress

`Done`: Node registration is completed

## Structs

### State Struct

used to store the state of the node registration.

#### Fields

`NodeID`: The ID of the node.

`TwinID`: The twin ID of the node.

`State`: The state of the node registration.

`Msg`: The message associated with the node registration state.

### RegistrationInfo Struct

used to store the capacity, location, and other information of the node.

#### Fields

`Capacity`: The capacity of the node.

`Location`: The location of the node.

`SecureBoot`: State whether the node is booted via efi or not.

`Virtualized`: State whether the node has hypervisor on it or not.

`SerialNumber`: The serial number of the node.

`GPUs`: The GPUs of the node.

#### Methods

`WithCapacity`: Set the capacity of the node, taking a `gridtypes.Capacity` as input.

`WithLocation`: Set the location of the node, taking a `geoip.Location` as input.

`WithSecureBoot`: Set the secure boot status of the node, taking a boolean as input.

`WithVirtualized`: Set the virtualized status of the node, taking a boolean as input.

`WithSerialNumber`: Set the serial number of the node, taking a string as input.

`WithGPUs`: Set the GPUs of the node, taking a string as input.

### Registrar Struct

The registrar is used to register nodes on the ThreeFold Grid.

#### Fields

`state`: The state of the registrar.

`mutex`: A mutex for synchronizing access to the registrar.

#### Methods

`NodeID`: Returns the node ID if the registrar is in the done state, otherwise returns an error.

`TwinID`: Returns the twin ID if the registrar is in the done state, otherwise returns an error.

## Functions

### `NewRegistrar(ctx context.Context, cl Zbus.Client, env environment.Environment, info RegistrationInfo) *Registrar`

Creates a new registrar with the given context, client, environment, and registration information, starts the registration process and returns the registrar.

### `FailedState(err error) State`

Returns a failed state with the given error.

### `InProgressState() State`

Returns an in progress state.

### `DoneState(nodeID , twinID uint32) State`

Returns a done state with the given node ID and twin ID.


