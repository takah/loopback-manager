# Development Guidelines for Claude

## Documentation Language
**All documentation must be written in English.** This includes:
- README files
- Code comments
- Commit messages
- Pull request descriptions
- Issue descriptions
- Any other documentation files

## Version Management and Tagging

### Creating a New Version
1. After merging changes to main branch, create a semantic version tag:
```bash
git tag v0.0.X
git push origin v0.0.X
```

2. Follow semantic versioning (https://semver.org/):
   - MAJOR version (v1.0.0): Incompatible API changes
   - MINOR version (v0.1.0): New functionality, backwards compatible
   - PATCH version (v0.0.1): Bug fixes, backwards compatible

### Go Proxy Version Recognition
After creating a new tag, ensure the Go proxy recognizes it:

```bash
# Force proxy to fetch the new version
GOPROXY=https://proxy.golang.org go list -m github.com/takah/loopback-manager@v0.0.X

# Verify available versions
go list -m -versions github.com/takah/loopback-manager
```

The proxy usually discovers new tags automatically within a few minutes, but the above command forces immediate recognition.

## Building and Testing

### Local Build
```bash
# Build with version information
make build

# Or direct go build (version will be detected from git)
go build -o loopback-manager
```

### Installation Methods
Users can install via:
1. `go install github.com/takah/loopback-manager@latest` (requires Go environment)
2. Binary download from releases
3. Build from source

## Version Command
The `version` command shows the current version:
- When built with `make`: Uses git tags via ldflags
- When installed with `go install`: Uses module version from build info
- Falls back to "dev" for development builds

## Release Process
1. Make changes and test locally
2. Create PR with English description
3. Merge to main
4. Create version tag (e.g., v0.0.5)
5. Push tag to origin
6. Force proxy recognition if needed
7. Verify with `go install github.com/takah/loopback-manager@latest`

## Important Notes
- Always write documentation in English
- Use semantic versioning for tags
- Test version command after releases
- Ensure Go proxy recognizes new versions before announcing