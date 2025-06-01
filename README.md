# ⚙️  Three-Phase Commit (3PC)

This project is a full implementation of the **Three-Phase Commit (3PC)** protocol in Go. It handles distributed transactions with fault tolerance, log-based recovery, and concurrent coordination.

![Three-Phase Commit Diagram](./3pc-diagram.jpeg)

---

## 🧠 Overview

This system coordinates atomic transactions across multiple key-value servers using a structured **3PC protocol** with the following guarantees:

- Lock acquisition and voting (Phase 1)
- Pre-commit agreement (Phase 2)
- Final commit or abort with fault recovery (Phase 3)

It simulates **server crashes**, **delays**, and **restarts**—including full recovery via **Query RPCs** and **phase inference**.

---

## 🛠️ Features

- ✅ **Full 3PC protocol** (Prepare, PreCommit, Commit/Abort)
- 🔐 Lock-based concurrency control using `sync.RWMutex`
- 🔁 **Retry logic** for RPCs in all phases (timeouts, ticker-based)
- 💾 Durable state recovery using inferred coordination phase
- 🧪 Tested with randomized failure, crash-restart, and timeout scenarios

---

## 📂 Structure

| File              | Purpose                                  |
|-------------------|------------------------------------------|
| `coordinator.go`  | Orchestrates 3PC logic, retries, recovery |
| `server.go`       | Simulates a key-value store with txn ops |
| `config.go`       | Protocol enums and message structs       |
| `3pc.go`          | Entry-point and client transaction setup |

---

## 📦 Build & Run

To test locally:

```bash
go run 3pc.go
