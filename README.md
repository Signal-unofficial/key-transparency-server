key-transparency-server
========================

In an end-to-end encrypted messaging ecosystem, users must have some way to obtain the public keys of the users they wish to message.
This typically happens by users uploading a mapping of their public identifier (for example, their phone number)
to their public key to a central directory or service, and other users querying this service.
However, this requires users to trust that the service operator will behave honestly and not tamper with the keys.

_Key transparency_ is a mechanism to ensure that a service operator cannot do this without being detected.
It does so by sequentially recording updates to the mappings, where an update is either an insert of a new mapping
from a search key to a value, or an update of an existing mapping.
In this context, "search key" refers to something like "n:+18005550101", and the value it maps to could be the public key associated with that phone number.

Every update—also referred to as a log entry—is appended to the _key transparency log_,
a globally consistent, cryptographically-protected, append-only log.
Although the key transparency log is an append-only construct, callers will need to update existing search keys
from time to time (e.g. if they lose their phone and wind up generating a new public key).
The log allows for updates by appending new log entries that supersede previous entries for the same search key.

This repo defines a key transparency server that maintains such a log for Signal, as well as associated testing and development tools.
It uses this [protocol][ietf-protocol] in the IETF's datatracker, although it
accommodates Signal's specific application requirements and therefore contains some modifications.

[ietf-protocol]: https://datatracker.ietf.org/doc/draft-ietf-keytrans-protocol/

Repo Overview
-------------
Key transparency code:
- `tree`: Defines the data structures used in the key transparency log, along with the logic to interact with them and generate proofs for various operations.

Operating the log:
- `cmd/kt-server`: Defines the server logic for interacting with the key transparency log.
- `db`: Defines various database and cache implementations.
- `filter-key-updates`: Defines the lambda that reads account changes from a DynamoDB stream and writes relevant updates to a Kinesis stream.

Tools/testing:
- `cmd/kt-client`: A tool for running various operations against a local or remote key transparency server.
- `cmd/kt-stress`: A tool for stress testing a key transparency server.
- `cmd/generate-auditing-test-vectors`: Generates test vectors for an auditor implementation.
- `cmd/generate-keys`: Generates a new set of private keys needed to run the key transparency server.

Key Transparency Services
-------------------------

The responsibilities of the key transparency server are split into three services:

- `KeyTransparencyQueryService` is a read-only server. It contains the `Search` and `Monitor` endpoints that clients use to query the log.
- `KeyTransparencyService` is a read-and-write server. It contains the audit endpoints and internally also writes updates to the log.
- `KeyTransparencyTestService` is a write-only server. It contains the `Update` endpoint and only exists for local testing and development use by `kt-client` and `kt-stress`.

All three services communicate via [gRPC](https://grpc.io/). Their definitions can be found in the `cmd/kt-server/pb` directory.

Data Structures
---------------
The key transparency log consists of two main types of data structures: a _log tree_ and _prefix trees_.

Log entries are stored as leaves in the log tree, and prefix trees are used to facilitate
efficient lookups of a specific mapping in the large and ever-growing log tree.
The transparency log uses various cryptographic techniques to append to the log tree and update prefix trees in a verifiable manner.

More details about each data structure and how they're used can be found in the [protocol][ietf-protocol].

Tests
-----

To run all tests:

```go
go test ./...
```

Quickstart
----------
You can run a key transparency server locally using [LevelDB](https://github.com/google/leveldb) as a backing database.
To do so, first run the `generate-keys` command to generate a new set of private keys:

```shell
go run github.com/signalapp/keytransparency/cmd/generate-keys
```

Copy-paste the keys into `example/config.yaml`. Then run the read-only, audit, and test servers locally (all three are required for
`kt-client` to have full functionality):

```shell
go run github.com/signalapp/keytransparency/cmd/kt-server -config ./example/config.yaml
```

You can now access metrics and the transparency
servers at the configured addresses. You can interact with the transparency log
via `kt-client`:

```shell
# Add an ACI to the log
go run github.com/signalapp/keytransparency/cmd/kt-client update aci \
<UUID> <base64_encoded_aci_identity_key>

# Add an E164 to the log. Be sure to use the same ACI as a previous update.
go run github.com/signalapp/keytransparency/cmd/kt-client update aci \
<e164_formatted_number> <UUID>

# Search for the ACI and provide the value it's mapped to
go run github.com/signalapp/keytransparency/cmd/kt-client search \
<UUID> <base64_encoded_aci_identity_key>

# Look up the distinguished key
go run github.com/signalapp/keytransparency/cmd/kt-client distinguished
```

To look up a username hash, you must also provide the ACI and ACI identity key:
```shell
# Search for an E164
go run github.com/signalapp/keytransparency/cmd/kt-client \
-username_hash <base64url_encoded_username_hash> \
search <UUID> <base64_encoded_aci_identity_key>

```

To look up an E164, you must provide the ACI, the ACI identity key, and an unidentified access key.
If using the `mock` AccountDB configuration, the default `-uak` value matches the one used by the mock implementation and can be left out.
```shell
# Search for an E164
go run github.com/signalapp/keytransparency/cmd/kt-client \
-e164 <e164_formatted_number> -uak <base64_encoded_uak> \
search <UUID> <base64_encoded_aci_identity_key>
```

To time the latency of a search or monitor request:
```shell
go run github.com/signalapp/keytransparency/cmd/kt-client -sample-size 50 -num-samples 10 \
  search-timing <UUID> <base64_encoded_aci_identity_key>

# Note that this will make a search request first so that it can compute the commitment index
# necessary for a monitor request
go run github.com/signalapp/keytransparency/cmd/kt-client -sample-size 50 -num-samples 10 \
  monitor-timing <UUID> <base64_encoded_aci_identity_key>
```

References
----------

- The `docs/` folder contains details on the format of the data stored in the database.
- [Key Transparency Architecture](https://datatracker.ietf.org/doc/draft-ietf-keytrans-architecture/) describes
the various deployment modes of key transparency and how to apply it to messaging and other applications.

You can also read these academic papers for background:

- [Merkle^2: A Low-Latency Transparency Log System](https://eprint.iacr.org/2021/453)
- [CONIKS: Bringing Key Transparency to End Users](https://eprint.iacr.org/2014/1004)
