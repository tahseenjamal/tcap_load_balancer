# Technical Note: Roadmap to Convert Current TCAP Router into a Carrier-Grade SIGTRAN Router

## Purpose

This note outlines the architectural and protocol enhancements required to transform the current **stateless TCAP load balancer prototype** into a **carrier-grade SIGTRAN router suitable for production telecom environments**.

The existing implementation already provides:

* Stateless TCAP dialog routing using transaction ID hashing
* SCTP ingress
* M3UA → SCCP → TCAP protocol parsing
* Zero-copy packet handling using `sync.Pool`
* Concurrent worker processing
* Backend routing

However, production deployment in SS7/SIGTRAN networks requires additional **protocol compliance, reliability mechanisms, and operational capabilities**.

---

# Current Architecture

Current processing pipeline:

```
SCTP Listener
      │
Packet Queue
      │
Worker Goroutines
      │
Parse M3UA
      │
Parse SCCP
      │
Parse TCAP ASN.1
      │
Stateless Router (OTID / DTID hash)
      │
Backend Pool
```

Routing logic:

```
TCAP BEGIN     → hash(OTID)
TCAP CONTINUE  → hash(DTID)
TCAP END       → hash(DTID)
TCAP ABORT     → hash(DTID)
```

This ensures dialog affinity without maintaining session state.

---

# Required Enhancements for Production Deployment

## 1. Full M3UA Protocol Support

The current implementation processes only **DATA messages**. Production SIGTRAN nodes must support the full **M3UA protocol state machine**.

### Required Message Classes

| Message Class           | Messages                           |
| ----------------------- | ---------------------------------- |
| Management              | ERR, NTFY                          |
| ASP State Maintenance   | ASPUP, ASPUP_ACK, ASPDN, ASPDN_ACK |
| ASP Traffic Maintenance | ASPAC, ASPAC_ACK, ASPIA, ASPIA_ACK |
| Heartbeat               | BEAT, BEAT_ACK                     |
| Transfer                | DATA                               |

### Required Behavior

The router must maintain the ASP state machine:

```
DOWN
  ↓ ASPUP
INACTIVE
  ↓ ASPAC
ACTIVE
```

Traffic must only be forwarded when the association is **ACTIVE**.

---

## 2. Backend Transport Using SCTP

The backend connections currently use TCP:

```
net.Dial("tcp", addr)
```

Production SIGTRAN networks require **SCTP associations**.

Required change:

```
sctp.DialSCTP("sctp", nil, remoteAddr)
```

Benefits:

* multi-homing support
* heartbeat monitoring
* failover
* multistream capability

---

## 3. Support Multiple M3UA Messages per SCTP Packet

An SCTP payload can contain **multiple M3UA messages**.

Example:

```
SCTP Packet
 ├── M3UA Message
 ├── M3UA Message
 └── M3UA Message
```

Current implementation parses only a single message.

Required improvement:

Iterate through messages using the length field:

```
offset = 0
while offset < packetLength
    parse message length
    process message
    offset += message length
```

---

## 4. Routing Based on SCCP Addressing

Production SIGTRAN routers often route traffic using **SCCP addressing fields**.

Important fields:

* Destination Point Code (DPC)
* Originating Point Code (OPC)
* Subsystem Number (SSN)

Example routing rule:

```
DPC 1234 → Backend Cluster A
DPC 5678 → Backend Cluster B
```

This enables flexible routing across multiple service platforms.

---

## 5. Backend Health Monitoring and Reconnection

Current behavior:

```
Write error → backend remains broken
```

Required improvements:

### Automatic Reconnection

If a write fails:

```
close connection
attempt reconnect
restore association
```

### Health Monitoring

Track:

* backend availability
* response latency
* error rate

Remove unhealthy backends from routing temporarily.

---

## 6. Worker Queue Sharding

The current design uses a **single global packet queue**, which can become a contention point under high load.

Current design:

```
Listener → Single Queue → Workers
```

Recommended design:

```
Listener
   │
hash(packet) % workerCount
   │
WorkerQueue[i]
   │
Worker i
```

Benefits:

* removes channel contention
* improves CPU cache locality
* increases throughput significantly

---

## 7. Backend Write Architecture

Currently backend writes use mutex protection:

```
worker → mutex → backend connection
```

For higher throughput, introduce **dedicated backend writer goroutines**.

Improved model:

```
worker → backend queue → backend writer goroutine → connection
```

This removes lock contention in the hot path.

---

## 8. SCTP Multistream Support

M3UA supports multiple streams inside one SCTP association.

Example:

```
Stream 0 → control traffic
Stream 1 → TCAP
Stream 2 → TCAP
Stream 3 → TCAP
```

Benefits:

* parallel delivery
* reduced head-of-line blocking
* higher signaling throughput

---

## 9. Observability and Monitoring

Carrier networks require operational visibility.

Recommended metrics:

| Metric          | Description          |
| --------------- | -------------------- |
| TCAP TPS        | traffic rate         |
| Dialog count    | active dialogs       |
| Queue depth     | congestion indicator |
| Error rate      | protocol failures    |
| Backend latency | service health       |

Expose metrics using a **Prometheus endpoint**.

---

## 10. High Availability Deployment

Production deployments require redundant routers.

Example topology:

```
           STP
            │
      ┌─────────────┐
      │ SIGTRAN LB  │
      └─────────────┘
        │         │
        │         │
      Backend1  Backend2
```

Deploy multiple routers for redundancy.

---

## 11. Configuration Management

Move configuration out of source code.

Example configuration format:

```yaml
listen:
  ip: 0.0.0.0
  port: 2905

backends:
  - name: hlr1
    ip: 10.0.0.1
    port: 2905
  - name: hlr2
    ip: 10.0.0.2
    port: 2905
```

Support configuration reload without restart.

---

## 12. Security Controls

Recommended safeguards:

* peer IP allow lists
* rate limiting
* malformed packet protection
* optional IPSec

---

# Suggested Implementation Phases

## Phase 1 — Protocol Compliance

* Implement M3UA management messages
* Use SCTP for backend connections
* Add backend reconnection logic

## Phase 2 — Performance Improvements

* Worker queue sharding
* Multi-message M3UA parsing
* Backend health checks

## Phase 3 — Operational Features

* metrics and monitoring
* routing based on point codes
* configuration management
* HA deployment

---

# Expected Throughput Improvements

| Stage                  | Approximate Throughput |
| ---------------------- | ---------------------- |
| Current prototype      | ~50k TPS               |
| Worker-sharded design  | ~300k TPS              |
| Multistream SCTP       | ~800k TPS              |
| Fully optimized router | 1M+ TPS                |

---

# Conclusion

The current TCAP router already provides a **strong architectural foundation**:

* stateless dialog routing
* efficient packet processing
* concurrent worker pipeline

By implementing the enhancements described in this document, the system can evolve into a **carrier-grade SIGTRAN router capable of handling production telecom signaling workloads**.

