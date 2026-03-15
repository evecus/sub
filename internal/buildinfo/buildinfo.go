package buildinfo

// Version is injected at build time via:
// -ldflags "-X 'github.com/evecus/sub/internal/buildinfo.Version=v1.0.0'"
var Version = "dev"
