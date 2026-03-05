# TCAP Load Balancer (Go)

A high-performance **stateless TCAP load balancer** written in Go.
The router receives **M3UA/SCTP traffic**, extracts **SCCP and TCAP dialogs**, and forwards packets to backend servers using a **deterministic hash of the TCAP transaction identifiers**.

This design allows **session stickiness without maintaining state**, enabling horizontal scaling and high throughput.

---

# Features

* **Stateless TCAP routing**
* **Session stickiness via hashing (OTID / DTID)**
* **SCTP listener for telecom signaling**
* **Zero-copy packet ingestion**
* **Memory reuse using `sync.Pool`**
* **Concurrent worker processing**
* **Thread-safe backend writes**
* **Minimal ASN.1 TCAP parser**

---

# Architecture

Traffic flow through the system:

```
SCTP Listener
      │
      ▼
Packet Queue (Buffered Channel)
      │
      ▼
Worker Pool (NumCPU goroutines)
      │
      ▼
M3UA Parser
      │
      ▼
SCCP Parser
      │
      ▼
TCAP ASN.1 Parser
      │
      ▼
Hash Router
      │
      ▼
Backend Pool
```

The router uses **deterministic hashing of TCAP transaction IDs**:

```
backend = hash(OTID or DTID) % backend_count
```

This ensures that **all packets belonging to the same TCAP dialog are routed to the same backend**.

---

# Directory Structure

```
.
├── backend.go       Backend connection pool
├── config.go        Application configuration
├── listener.go      SCTP listener and packet ingestion
├── m3ua.go          M3UA protocol parser
├── main.go          Application entrypoint
├── packet.go        Packet structures and queue
├── router.go        TCAP routing logic
├── sccp.go          SCCP protocol parser
├── tcap_asn1.go     Minimal TCAP ASN.1 decoder
├── tcap.go          TCAP message types
└── worker.go        Worker processing pipeline
```

---

# Protocol Stack

The load balancer processes signaling in the following order:

```
SCTP
 └── M3UA
      └── SCCP
           └── TCAP
```

Only **transaction identifiers** are extracted from TCAP for routing purposes.

---

# Configuration

Configuration is defined in `config.go`.

Example:

```go
Config{
    ListenAddr: "0.0.0.0:2905",
    Backends: []string{
        "127.0.0.1:9001",
        "127.0.0.1:9002",
        "127.0.0.1:9003",
    },
}
```

| Field        | Description                                  |
| ------------ | -------------------------------------------- |
| `ListenAddr` | SCTP address where the load balancer listens |
| `Backends`   | List of backend TCAP servers                 |

---

# TCAP Session Stickiness

Session affinity is achieved without maintaining state.

Routing rules:

| TCAP Message | Identifier Used |
| ------------ | --------------- |
| BEGIN        | OTID            |
| CONTINUE     | DTID            |
| END          | DTID            |
| ABORT        | DTID            |

The identifier is hashed:

```
hash(transaction_id) % backend_count
```

This guarantees that **all messages in a dialog go to the same backend**.

---

# Performance Design

The implementation includes several performance optimizations:

### Zero-Copy Packet Processing

Buffers are reused using `sync.Pool` to avoid allocations.

```
bufferPool → packetQueue → worker → bufferPool
```

This significantly reduces GC pressure.

---

### Concurrent Workers

Workers are spawned based on CPU cores:

```go
workerCount := runtime.NumCPU()
```

Each worker performs:

```
Parse M3UA → Parse SCCP → Parse TCAP → Route packet
```

---

### Thread-Safe Backend Writes

Backend connections are protected using mutexes to allow concurrent access.

---

# Building

Ensure Go is installed:

```
go version
```

Build the project:

```
go build
```

Run the load balancer:

```
go run .
```

---

# Testing

You can test using telecom signaling tools such as:

* **Osmocom STP**
* **SIGTRAN simulators**
* **Custom M3UA generators**
* **SCTP packet replay tools**

Backend servers can be simple TCP listeners for testing.

Example:

```
nc -l 9001
nc -l 9002
nc -l 9003
```

---

# Limitations

This project currently provides a **minimal TCAP router implementation**.

Not implemented yet:

* Full TCAP ASN.1 decoding
* M3UA management messages
* Backend reconnection handling
* Multiple M3UA messages per SCTP packet
* Metrics and monitoring
* SCTP multi-stream support

---

# Future Improvements

Possible enhancements:

* Multi-queue worker architecture
* Backend health checks
* SCTP multistream routing
* Metrics (Prometheus)
* Advanced TCAP parsing
* Connection pooling
* Load balancing strategies

---

