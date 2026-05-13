# k8s-doc (Kubernetes Doctor)

Hackathon prototype: Kubernetes Doctor for disposable k8s-snap labs.

## What It Does & How AI Is Used

**k8s-doc** is an AI-assisted Kubernetes diagnostic tool that lowers the barrier to debugging cluster failures. When a user describes a problem — starting with DNS/CoreDNS breakage — the system automatically collects live cluster evidence via diagnostic playbooks, retrieves the most relevant documentation through vector-based semantic search, and presents a structured, actionable diagnosis complete with evidence, a recommended fix, and verification steps.

### The Problem

Debugging a broken Kubernetes cluster requires piecing together scattered upstream documentation, knowing exactly which `kubectl` commands to run, and interpreting raw output against what "healthy" looks like. This is slow, error-prone, and inaccessible to less experienced operators. k8s-doc replaces the manual forensics loop with a single conversational interface backed by real cluster data.

### How AI Is Used

The system employs a **RAG (Retrieval-Augmented Generation)** pipeline — Kubernetes documentation (both k8s-snap and upstream) is chunked and embedded into vectors at indexing time. When a user submits a question, the query is embedded in the same vector space, and cosine similarity search retrieves the semantically closest documentation snippets. These citations ground every diagnosis in authoritative sources. An **LLM abstraction layer** (supporting OpenAI, Ollama, or local models) coordinates tool calls to assemble complete diagnoses. The output is a structured schema (Summary, Diagnosis, Evidence, Fix, Verification, Citations) that ensures every result is traceable, auditable, and actionable.

## MVP

- Web chat UI
- k8s-snap and upstream Kubernetes docs retrieval
- LXD-backed local lab
- k8s snap install from `latest/stable`
- DNS/CoreDNS break/fix diagnosis
- Append-only audit log

## Run

```bash
go run ./cmd/k8s-doc serve
```

## Test

```bash
go test ./...
```
