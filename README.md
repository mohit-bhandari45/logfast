# logfast

**High-Throughput Log Stream Analyzer & Aggregator**

A high-speed CLI tool built in Go that ingests, parses, filters, and aggregates gigabytes of structured (JSON/Logfmt) or unstructured log streams in real-time with ultra-low memory allocations.

## Features to Implement

- **Zero-copy string parsing**: Parse log lines without unnecessary memory allocations using `unsafe` or custom `bufio.Scanner`.
- **Fan-out / Fan-in concurrency**: Use worker pools to process chunks of log files in parallel across multiple CPU cores.
- **Garbage Collection Minimization**: Keep heap allocations extremely low by reusing objects with `sync.Pool`.
- **Real-time Aggregation**: Calculate p50, p95, and p99 latency metrics over streaming windows using ring buffers.
- **Zero-dependency CLI**: Use Cobra to build the command-line interface.

## Tech Stack
- Go (Golang)

## Getting Started
*(You will build this out!)*
