# 🛸 Hoverfly

**The modern, secure, and ultra-lightweight local Hive mock server. Built for 2026 and beyond.**

In nature, the **hoverfly** (_Syrphidae_) is the classic example of Batesian mimicry. It looks and behaves almost exactly like a stinging wasp or honey bee to deter predators, but it is completely harmless and stinger-free. Similarly, `hoverfly` mimics the JSON-RPC surface, network formats, transaction flow, and stateful responses of a real Hive node, offering a fast local testing sandbox without consensus overhead, P2P networking, or live mainnet side effects.

If you are developing or testing Hive applications, bots, SDKs, or scripts, Hoverfly is your single-binary local sandbox.

---

**Mock-First and Stateful:** Hoverfly is written in Go, powered by a high-performance **BadgerDB** state engine, and currently provides developer-useful first-class responses for **215/215 Hive OpenAPI JSON-RPC methods**.

---

## Why Hoverfly?

The Hive ecosystem deserves testing infrastructure that is fast, local, and reliable. No more waiting for public testnets, chasing fragile fixtures, or polluting the live blockchain with test transactions.

### 🔌 Complete Hive JSON-RPC Mock Coverage

Hoverfly tracks the live Hive OpenAPI method surface and routes every known method through a first-class mock handler:

- **215/215 OpenAPI Methods Routed**: `condenser_api`, `database_api`, `account_history_api`, `bridge`, `wallet_bridge_api`, `rc_api`, `market_history_api`, `debug_node_api`, and related API groups all answer locally.
- **No Unrouted Fallbacks Required**: The generic OpenAPI example layer exists as a safety net, but current documented coverage is first-class and developer-useful.
- **Bridge/Hivemind Shapes Included**: Posts, profiles, discussions, communities, ranked posts, notification counts, and relationship/list endpoints return realistic local shapes instead of only echoing docs examples.

See [`HIVE_API_CHECKLIST.md`](HIVE_API_CHECKLIST.md) for the method-by-method coverage notes.

### 🔒 Transaction Verification & Mutation

Hoverfly does enough real transaction work to catch common SDK and script mistakes before they reach mainnet:

- **Signature Recovery**: Recovers public keys from compact ECDSA signatures using `decred/secp256k1`.
- **Hive Wire Serialization**: Reconstructs transaction bytes locally for signature hashing and transaction hex endpoints.
- **State Mutation**: Accepted `transfer`, `transfer_to_savings`, and `comment` operations update local balances, savings balances, posts, replies, transaction history, and account history.

### ⚡ BadgerDB State Storage

Powered by BadgerDB (v4), Hoverfly provides structured, transactional storage for mock accounts, balances, post bodies, metadata, blocks, and transactions:

- **Ephemeral Mode (Default)**: Runs completely in-memory. Stopping the process wipes all simulated accounts and state, providing a perfectly clean slate every run.
- **Persistent Storage**: Pass the `--db` flag to persist accounts, posts, transactions, and keys to a local directory for long-lived manual testing.
- **Resettable Test Runs**: Use `--reset` with `--db` to start from known defaults while keeping the same local database path.

### 🧪 Script-Friendly Local Defaults

Hoverfly is designed for "does my app work?" testing:

- **Seeded Accounts**: `@thecrazygm`, `@alice`, and `@bob` exist out of the box.
- **Local Blocks**: A background ticker simulates real block production every 3 seconds.
- **Useful Empty State**: APIs for escrows, delegations, orders, conversions, subscriptions, and notifications return stable empty local state when no matching data exists.
- **Debug Helpers**: `debug_node_api` methods can advance blocks and inspect local head state without a live node.

### 🚀 Concurrency & Block Ticking

A background ticker goroutine simulates real block production, incrementing the block number and updating global dynamic properties every 3 seconds.

### 💅 Charm-Powered Logging

No more dry, unreadable terminal logs. Hoverfly uses **[charmbracelet/log](https://github.com/charmbracelet/log)** to output beautiful, color-coded structured logs tracking incoming JSON-RPC calls, transaction status, block ticks, and state changes.

### 🔌 Ecosystem Alignment

Hoverfly is the local testing companion to **[Anther](https://github.com/thecrazygm/anther)** (Go), **[Pollen](https://github.com/srbde/pollen)** (TypeScript), **[Xylem](https://github.com/srbde/xylem)** (Rust), and **[Nectar](https://github.com/srbde/hive-nectar)** (Python). Together, they form a unified, secure foundation for building cross-platform Hive applications under the **SRBDE** umbrella.

---

## 🚀 Quick Start

Requires Go >= 1.20.

### Installation

Clone the repository and build:

```bash
git clone https://github.com/TheCrazyGM/libraries
cd libraries/hoverfly
go build -o hoverfly main.go
```

### Running the Server

#### Ephemeral In-Memory Mode (Default)

Starts the mock server immediately in-memory:

```bash
./hoverfly
```

#### Persistent State Mode

Persists accounts, balances, and keys to a local database directory:

```bash
./hoverfly --db ./hoverfly_db
```

#### Wipe and Start Fresh

Deletes the local database directory on boot:

```bash
./hoverfly --db ./hoverfly_db --reset
```

#### Custom Bind Port

By default, Hoverfly binds to port `8090` (matching default Hive nodes). Change it using:

```bash
./hoverfly --port 8080
```

---

## ⚙️ CLI Reference

| Flag      | Type     | Default | Description                                             |
| --------- | -------- | ------- | ------------------------------------------------------- |
| `--port`  | `int`    | `8090`  | Port to bind the HTTP JSON-RPC server                   |
| `--db`    | `string` | `""`    | Directory path to BadgerDB. If empty, runs in-memory.   |
| `--reset` | `bool`   | `false` | If true, deletes the BadgerDB directory before booting. |
| `--debug` | `bool`   | `false` | Enables verbose request and state-change logging.       |

---

## 📡 API Coverage

Hoverfly currently provides developer-useful first-class mocks for **215/215** Hive OpenAPI JSON-RPC methods.

| Area                   | Coverage | Notes                                              |
| ---------------------- | -------- | -------------------------------------------------- |
| Core chain APIs        | 100%     | Blocks, dynamic properties, config, version, TAPOS |
| Account APIs           | 100%     | Lookup, lists, key references, RC, balances        |
| Broadcast APIs         | 100%     | Saves transactions and mutates supported state     |
| Content APIs           | 100%     | Posts, replies, discussions, votes, blogs, search  |
| Bridge/Hivemind APIs   | 100%     | Profiles, communities, ranked posts, notifications |
| History APIs           | 100%     | Transactions, account history, ops-in-block        |
| Debug APIs             | 100%     | Local block generation and head-state inspection   |
| Market/governance APIs | 100%     | Stable local templates and empty-state responses   |

Hoverfly is not a consensus node and does not run P2P networking, witness scheduling, or real economics. It is intentionally a local app-development target: fast enough for tests, stateful enough for scripts, and compatible enough for SDK integration work.

---

## 🛸 Client SDK Integration

To test your applications locally, configure your client instance to point to your `hoverfly` endpoint:

### Go (`anther`)

```go
package main

import (
	"fmt"
	"log"

	"github.com/thecrazygm/anther/client"
	"github.com/thecrazygm/anther/transaction"
)

func main() {
	// Point to the local Hoverfly server
	api := client.NewClient([]string{"http://localhost:8090"}, 30)
	tx := transaction.NewTransaction(api)

	// Append transfer
	tx.AppendOp(&transaction.Transfer{
		From:   "thecrazygm",
		To:     "alice",
		Amount: "10.000 HIVE",
		Memo:   "Testing locally with Hoverfly 🛸",
	})

	// Sign and broadcast using standard test/active WIF
	wif := "5KQwrPbwdL6PhUMCYFZd4C29VJpqZLfL96AUtDQm2h5pt8E6bCS"
	if err := tx.Sign(wif); err != nil {
		log.Fatalf("failed to sign: %v", err)
	}

	result, err := tx.Broadcast()
	if err != nil {
		log.Fatalf("failed to broadcast: %v", err)
	}
	fmt.Printf("Broadcast result: %v\n", result)
}
```

### Rust (`xylem`)

```rust
use xylem::{Client, Transaction};
use xylem::operations::Transfer;
use xylem::types::HiveTime;
use chrono::Utc;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Point to the local Hoverfly server
    let client = Client::new(vec!["http://localhost:8090".to_string()], 30);

    let props = client.get_dynamic_global_properties().await?;
    let ref_block_num = (props.head_block_number & 0xFFFF) as u16;
    let prefix_bytes = hex::decode(&props.head_block_id[8..16])?;
    let ref_block_prefix = u32::from_le_bytes(prefix_bytes.try_into().unwrap());

    let expiration = HiveTime(Utc::now().naive_utc() + chrono::Duration::minutes(1));
    let mut tx = Transaction::new(ref_block_num, ref_block_prefix, expiration);

    tx.append_op(Box::new(Transfer {
        from: "thecrazygm".to_string(),
        to: "alice".to_string(),
        amount: "10.000 HIVE".to_string(),
        memo: "Testing locally with Hoverfly 🛸".to_string(),
    }));

    let active_wif = "5KQwrPbwdL6PhUMCYFZd4C29VJpqZLfL96AUtDQm2h5pt8E6bCS";
    let chain_id = "beeab0de00000000000000000000000000000000000000000000000000000000";
    tx.sign(active_wif, chain_id)?;

    let response = client.broadcast_transaction(&tx).await?;
    println!("Broadcast Result: {}", response);

    Ok(())
}
```

---

## 🧪 Seeding & Mock Defaults

When Hoverfly boots, it pre-seeds the following test entities so standard SDK examples run out of the box:

- **Mock Accounts**: `@thecrazygm` (seeded with `1000.000 HIVE` and `500.000 HBD`), `@alice`, and `@bob`.
- **Active Key Registry**: Maps `@thecrazygm`'s active public key (`STM5kQ1uy2CGNSwibSeYyLELWFng3HTyYVSsQd4Bjd4sWfqgKgtgJ`) so key reference lookups (`get_key_references`) resolve successfully.
- **Dynamic Properties**: Simulates block number progression starting at block `100,000,000` and ticking up every 3 seconds.
- **Mutable Local State**: Broadcast transfers update balances, savings transfers update savings balances, comments create posts/replies, and saved transactions become visible through transaction/history APIs.

---

## 🛠️ Building & Testing

Hoverfly uses standard Go tooling:

```bash
# Run unit tests
go test ./...

# Format the codebase
go fmt ./...

# Compile release binary
go build -ldflags="-s -w" -o hoverfly main.go
```

---

## 📜 Standing on Shoulders

Hoverfly is a completely original Hive mocking server designed from the ground up to bring local-first development and testing to the Hive ecosystem. It implements the Hive JSON-RPC API surface, transaction signature recovery, TAPOS parameters, state mutation, and local history needed to test real client behavior without requiring a public testnet.

---

## 🌐 Built by SRBDE

**Hoverfly** is developed and maintained by the **Sustainable Resource and Business Development Enterprise (SRBDE)** — an open-source infrastructure organization building tools and platforms for communities that build things together.

We apply the logic of agricultural sustainability to software: the goal is always to return more to the ecosystem than we extract.

- **Open source is our value, not just our business model.**
- **Our commercial products fund our open-source core. The open work is the mission.**

### Explore the Ecosystem

| Project                                               | Description                       |
| ----------------------------------------------------- | --------------------------------- |
| [Pollen](https://github.com/srbde/pollen)             | The modern Hive TypeScript SDK    |
| [Anther](https://github.com/srbde/anther)             | The modern Hive Go SDK            |
| [Xylem](https://github.com/srbde/xylem)               | The modern Hive Rust SDK          |
| [Nectar](https://github.com/srbde/hive-nectar)        | The modern Hive Python SDK        |
| [nectarengine](https://github.com/srbde/nectarengine) | The Hive-Engine sidechain library |
| [ecoinstats.net](https://ecoinstats.net)              | SRBDE corporate hub               |
| [thecrazygm.com](https://thecrazygm.com)              | Open gaming tools & TTRPGs        |

---

## 🤝 Contributing

Audits, forks, and pull requests are welcome. **Hoverfly** is built to last for the decade, not the quarter. If you find a security issue, please open a private advisory rather than a public issue.
