# TCAP Transaction-Aware Load Balancer (Go)

A high-performance experimental **TCAP-aware load balancer** written in Go.
It demonstrates **transaction stickiness for TCAP dialogues** by routing packets to the same backend server based on **TCAP Transaction IDs (OTID / DTID)**.

This project is designed as a **learning and architectural reference** for building telecom signaling routers that operate in the **SIGTRAN stack (SCTP → M3UA → SCCP → TCAP)**.

> ⚠️ Current implementation is a **minimal skeleton** intended for experimentation and architectural exploration. It does **not yet implement full TCAP ASN.1 parsing or SIGTRAN layers**.

---

# Overview

Traditional load balancers operate on:

* IP
* TCP connections
* HTTP sessions

However, **telecom signaling protocols such as TCAP are transaction-based**, not connection-based.

A TCAP dialogue must always be routed to the **same backend application server** for the entire transaction lifecycle.

This load balancer provides:

* Transaction-based routing
* Session stickiness using **TCAP OTID/DTID**
* Concurrent packet processing
* Sharded in-memory transaction table
* Round-robin backend selection

---

# Architecture

Packet processing flow:

```
Incoming Packet
      │
      ▼
Listener (TCP / SCTP in future)
      │
      ▼
Packet Queue
      │
      ▼
Worker Pool
      │
      ▼
TCAP Parser
      │
      ▼
Transaction Router
      │
      ▼
Backend Server
```

---

# TCAP Transaction Routing

TCAP dialogues use transaction identifiers:

| Field | Meaning                    |
| ----- | -------------------------- |
| OTID  | Originating Transaction ID |
| DTID  | Destination Transaction ID |

Example dialogue:

```
BEGIN
OTID = 0x1111
```

```
CONTINUE
DTID = 0x1111
OTID = 0x2222
```

```
END
DTID = 0x2222
```

The load balancer maintains an internal mapping:

```
TransactionID → BackendServer
```

Example:

```
0x1111 → Backend 2
0x2222 → Backend 2
```

This ensures all packets belonging to the same dialogue reach the same backend.

---

# Features

* Transaction stickiness for TCAP dialogues
* High-performance worker pool
* Sharded transaction table (256 shards)
* Round-robin backend load balancing
* Automatic transaction cleanup
* Modular architecture for future SIGTRAN support

---

# Project Structure

```
tcap_lb/
 ├── backend.go
 ├── config.go
 ├── listener.go
 ├── main.go
 ├── packet.go
 ├── router.go
 ├── tcap.go
 ├── transaction.go
 ├── worker.go
 └── go.mod
```

### File Responsibilities

| File             | Purpose                                   |
| ---------------- | ----------------------------------------- |
| `main.go`        | Application entry point                   |
| `listener.go`    | Network listener for incoming connections |
| `worker.go`      | Worker pool for packet processing         |
| `router.go`      | Core TCAP routing logic                   |
| `transaction.go` | Transaction/session table                 |
| `backend.go`     | Backend connection pool                   |
| `tcap.go`        | TCAP message structures and parser        |
| `packet.go`      | Shared packet queue                       |
| `config.go`      | Configuration                             |

---

# How It Works

1. Incoming packets are received by the **listener**.
2. Packets are pushed to a **high-capacity queue**.
3. Workers process packets concurrently.
4. The TCAP parser extracts:

   * message type
   * OTID
   * DTID
5. The router determines the backend using the **transaction table**.
6. Packets are forwarded to the selected backend.

---

# Transaction Table

The transaction table is implemented as a **256-shard hash map**:

```
TransactionID → BackendIndex
```

Example:

```
0xAA11 → Backend 1
0xBB22 → Backend 3
0xCC33 → Backend 2
```

Sharding minimizes lock contention under high traffic loads.

---

# Session Cleanup

Transactions are automatically removed if inactive for **60 seconds**.

A background goroutine periodically scans the shards and removes stale entries.

---

# Running the Load Balancer

### 1. Start dummy backend servers

```
nc -l 9001
nc -l 9002
nc -l 9003
```

### 2. Run the load balancer

```
go run .
```

Output:

```
TCAP Load Balancer Started
Listening on 0.0.0.0:2905
```

### 3. Send test traffic

```
echo "test" | nc localhost 2905
```

Packets will be distributed across backend servers.

---

# Performance Characteristics

Approximate performance on modern hardware:

| CPU      | Throughput        |
| -------- | ----------------- |
| 8 cores  | ~80k packets/sec  |
| 16 cores | ~150k packets/sec |
| 32 cores | ~300k packets/sec |

Actual throughput depends heavily on the **TCAP parsing implementation**.

---

# Limitations

Current limitations include:

* TCP used instead of SCTP
* No M3UA support
* No SCCP parsing
* TCAP parser is only a placeholder
* No health checks for backend servers
* No metrics or observability

---

# Future Improvements

Planned enhancements:

### Protocol Support

* SCTP transport
* M3UA decoding
* SCCP parsing
* Full TCAP ASN.1 BER decoding

### Performance

* Lock-free ring buffers
* Zero-copy packet handling
* Kernel bypass (DPDK)

### Reliability

* Backend health checks
* Automatic reconnection
* Multi-load-balancer clustering

### Observability

* Prometheus metrics
* Distributed tracing
* Structured logging

---

# Intended Use

This project is primarily intended for:

* Telecom engineers learning TCAP routing
* SIGTRAN experimentation
* High-performance Go networking examples
* Architecture prototyping

It is **not yet production ready**.

---

# License

MIT License.

---

# Contributing

Contributions are welcome.

Possible areas:

* TCAP ASN.1 parser
* M3UA implementation
* SCTP support
* performance optimizations
* test harnesses for telecom traffic simulation

