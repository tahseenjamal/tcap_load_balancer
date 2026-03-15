# TCAP Router (Go)

A **high-performance TCAP routing service** implemented in Go.
It performs **dialogue-aware routing of TCAP messages** to backend application servers.

The router is designed to run **behind a SIGTRAN STP**, typically OsmoSTP, which handles:

* SCTP
* M3UA
* routing contexts
* signalling links
* network redundancy

This router focuses purely on **TCAP dialogue routing**.

---

# Architecture

```
SS7 / SIGTRAN Network
        │
        ▼
      osmo-stp
        │
        ▼
   TCAP Router (this project)
        │
        ▼
 TCAP Application Servers
 (HLR / USSD / CAMEL / etc)
```

The router ensures that **all messages belonging to the same TCAP dialogue are sent to the same backend node.**

---

# Features

### High Performance

* Worker pool (`CPU * 4`)
* Sharded transaction table (256 shards)
* Lock contention minimized
* Large packet queue (500k)

### Dialogue Affinity

Routing rules:

| TCAP Message | Routing Strategy                     |
| ------------ | ------------------------------------ |
| BEGIN        | Hash by OTID                         |
| CONTINUE     | Lookup by DTID                       |
| END / ABORT  | Same backend then delete transaction |

This guarantees **dialogue stickiness**.

---

### Backend Connection Pool

Each backend server uses **multiple TCP sockets**.

Example:

```
BackendSockets = 8
Backends = 3
Total outbound sockets = 24
```

Benefits:

* avoids socket bottlenecks
* improves throughput
* reduces latency

---

### Automatic Backend Recovery

If a backend connection fails:

1. connection is closed
2. router attempts reconnect
3. routing continues

---

### Transaction TTL Cleanup

Transactions are automatically cleaned if a dialogue never terminates.

```
txTTL = 60 seconds
cleanup interval = 30 seconds
```

Prevents memory leaks caused by missing END messages.

---

### High Throughput Listener

TCP listener features:

* 4MB socket buffer
* TCP_NODELAY enabled
* atomic packet drop counter
* throttled logging

Dropped packets are logged every **1000 drops** to prevent log storms.

---

# Code Structure

| File         | Purpose                           |
| ------------ | --------------------------------- |
| main.go      | Application entry point           |
| listener.go  | TCP listener and packet ingestion |
| worker.go    | Worker pool consuming packets     |
| router.go    | Dialogue routing logic            |
| backend.go   | Backend connection pools          |
| tcap_asn1.go | Minimal TCAP parser               |
| tcap.go      | TCAP message structures           |
| packet.go    | Packet queue structure            |
| config.go    | Router configuration              |

---

# Configuration

Configuration is currently defined in `config.go`.

Example:

```go
ListenAddr: "0.0.0.0:2905"

BackendSockets: 8

Backends: []string{
    "127.0.0.1:9001",
    "127.0.0.1:9002",
    "127.0.0.1:9003",
}
```

---

# Build

```
go build -o tcap_router
```

---

# Run

```
./tcap_router
```

The router will start listening on:

```
0.0.0.0:2905
```

---

# Expected Throughput

Approximate performance on modern hardware:

| CPU      | Expected TCAP TPS |
| -------- | ----------------- |
| 8 cores  | ~80k – 100k TPS   |
| 16 cores | ~150k TPS         |
| 32 cores | ~250k+ TPS        |

Actual throughput depends on backend processing latency.

---

# Production Deployment

Recommended system tuning.

### Increase file descriptors

```
ulimit -n 200000
```

---

### Linux network tuning

```
net.core.somaxconn = 65535
net.core.netdev_max_backlog = 250000
net.ipv4.tcp_max_syn_backlog = 65535
net.core.rmem_max = 134217728
net.core.wmem_max = 134217728
```

---

# Failure Handling

| Failure                | Behavior                        |
| ---------------------- | ------------------------------- |
| Backend socket failure | automatic reconnect             |
| Packet queue overflow  | packet dropped                  |
| Missing TCAP END       | TTL cleanup removes transaction |

---

# Limitations

This router intentionally **does not implement full TCAP decoding**.

Only the following fields are parsed:

* message type
* OTID
* DTID

It assumes that:

* TCAP is correctly framed
* upstream STP already validated signalling

---

# Use Cases

Typical telecom applications:

* HLR queries
* CAMEL service control
* USSD gateways
* SMS routing
* IN services

---

# Future Improvements

Possible enhancements:

* metrics endpoint (Prometheus)
* backend health monitoring
* sync.Pool packet reuse
* epoll-based network handling
* multi-listener sharding

