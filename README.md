# Tools

### Build

```bash
go mod tidy
```

```bash
go test ./...
```

```bash
golangci-lint run
```

### Release

* Get list of tags by `git tag`
* Set new tag by `git tag v1.0.0`
* Push tags `git push --tags