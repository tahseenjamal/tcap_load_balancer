# Future Improvement Note — Worker Queue Sharding for High Throughput

## Background

The current TCAP load balancer implementation uses a **single global packet queue**:

```go
var packetQueue = make(chan Packet, 500000)
```

Pipeline:

```
SCTP Listener
      │
      ▼
Global Packet Queue
      │
      ▼
Worker Goroutines
```

While this design works well for moderate traffic, the **single channel becomes a contention point** under heavy load. Multiple goroutines pushing packets into the same queue can significantly limit throughput.

To reach **telecom-grade performance (300k–1M+ TCAP transactions per second)**, the architecture should be upgraded to use **CPU-sharded worker queues**.

---

# Proposed Architecture

Replace the single global queue with **multiple worker queues**, one per CPU core.

```
SCTP Listener
      │
hash(packet) % workerCount
      │
workerQueue[i]
      │
Worker i
      │
Parse M3UA
      │
Parse SCCP
      │
Parse TCAP
      │
Router
      │
Backend
```

This eliminates the global contention point and improves cache locality.

---

# Implementation Plan

## 1. Replace Global Queue

Current:

```go
var packetQueue = make(chan Packet, 500000)
```

Replace with:

```go
var workerQueues []chan Packet
```

---

## 2. Initialize Worker Queues

In `main.go`:

```go
workerCount := runtime.NumCPU()

workerQueues = make([]chan Packet, workerCount)

for i := 0; i < workerCount; i++ {

    workerQueues[i] = make(chan Packet, 100000)

    go StartWorker(router, workerQueues[i])
}
```

Each worker now processes packets from its **dedicated queue**.

---

## 3. Update Worker

Modify the worker function to read from its own queue.

```go
func StartWorker(router *Router, queue chan Packet) {

    for pkt := range queue {

        m3ua, ok := ParseM3UA(pkt.Data)
        if ok {

            sccp, ok := ParseSCCP(m3ua.Payload)
            if ok {

                tcap, ok := ParseTCAPASN1(sccp.Payload)
                if ok {

                    router.Route(tcap, pkt.Data)
                }
            }
        }

        bufferPool.Put(pkt.Buffer)
    }
}
```

---

## 4. Route Packets to Worker Queues

Modify `listener.go`.

Replace:

```go
packetQueue <- packet
```

with:

```go
idx := int(packet.Data[0]) % len(workerQueues)

select {

case workerQueues[idx] <- packet:

default:

    log.Println("worker queue full, dropping packet")

    bufferPool.Put(bufPtr)
}
```

This distributes packets across worker queues.

For maximum correctness later, the hash may be based on:

```
TCAP OTID
```

instead of the first byte of the packet.

---

# Expected Performance Improvement

| Architecture          | Estimated Throughput |
| --------------------- | -------------------- |
| Single global queue   | ~80k TPS             |
| Sharded worker queues | ~300k TPS            |
| With SCTP multistream | 1M+ TPS              |

---

# Benefits

1. Removes channel contention
2. Improves CPU cache locality
3. Enables horizontal scaling with CPU cores
4. Supports very high packet throughput

---

# Optional Future Enhancements

Additional improvements that could further increase performance:

### 1. Multi-message M3UA parsing

SCTP frames can contain multiple M3UA messages.

Current implementation processes only one.

Future improvement: iterate over multiple M3UA frames in the same SCTP packet.

---

### 2. Backend writer goroutines

Instead of using a mutex on backend connections:

```
worker → backend writer queue → backend connection
```

This removes write locks entirely.

---

### 3. SCTP multistream support

Using multiple SCTP streams can significantly increase throughput in telecom signaling environments.

---

# Summary

The current implementation is already stable and functional.
However, implementing **worker queue sharding** will allow the router to scale to **carrier-grade throughput levels**.

This change is architectural but relatively small in terms of code modification and can be implemented later without impacting the existing protocol logic.

