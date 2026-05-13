# k8s-doc

Hackathon prototype: Kubernetes Doctor for disposable k8s-snap labs.

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
