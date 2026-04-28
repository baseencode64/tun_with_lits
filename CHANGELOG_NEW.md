# Changelog - Server Selection Feature

## [Unreleased] - 2025-01-XX

### Added

#### 🎯 Automatic Server Selection from Raw Lists

**New Components:**

1. **`pkg/client/link_parser.go`**
   - `LinkParser` struct for parsing and validating VLESS links
   - `ParseLinksFromRaw()` - extracts VLESS links from raw text
   - `ValidateLink()` - validates single VLESS link format
   - Ignores comments (lines starting with #) and empty lines

2. **`pkg/client/server_selector.go`**
   - `ServerSelector` struct for intelligent server selection
   - `FetchRawLinks()` - downloads and parses VLESS links from URL
   - `CheckLatency()` - measures server latency via TCP connection
   - `SelectBest()` - selects optimal server from list based on latency
   - `SelectBestFromURL()` - one-call method to fetch and select best server
   - Concurrent server checking with configurable max parallelism
   - `ServerInfo` struct with link, host, port, latency, and availability info

3. **`pkg/client/slog_adapter.go`**
   - `SlogAdapter` - bridges slog.Logger with project's Logger interface
   - Allows seamless integration of slog with existing logging

4. **`pkg/client/interfaces.go`**
   - Added `Logger` interface with Debug, Info, and Error methods
   - Standardized logging across the project

5. **`main.go` enhancements**
   - Added `--from-raw` flag support
   - Usage: `goxray --from-raw <https://example.com/links.txt>`
   - Automatic workflow: fetch → parse → check latency → select best → connect

6. **Test coverage:**
   - `pkg/client/link_parser_test.go` - comprehensive tests for link parsing
   - `pkg/client/server_selector_test.go` - tests for server selection logic

7. **Documentation:**
   - Updated README.md with new feature documentation
   - Added `example_links.txt` template file
   - Multiple usage examples for library integration

### Features

- ✅ **Parallel server checking** - concurrent latency measurement
- ✅ **Configurable concurrency** - control max simultaneous checks (default: 10)
- ✅ **Timeout per server** - fast rejection of unreachable servers (default: 5s)
- ✅ **Smart sorting** - servers sorted by latency, best first
- ✅ **Fallback logic** - if best server fails, next one is selected
- ✅ **Progress logging** - visible feedback during server checking
- ✅ **Metrics** - shows which server was selected and its latency
- ✅ **Comment support** - ignores lines starting with #
- ✅ **Invalid link filtering** - automatically skips malformed links

### API Changes

**New Types:**
```go
type Logger interface {
    Debug(msg string, keysAndValues ...interface{})
    Info(msg string, keysAndValues ...interface{})
    Error(msg string, keysAndValues ...interface{})
}

type ServerInfo struct {
    Link      string
    Host      string
    Port      string
    Latency   time.Duration
    Available bool
}

type LinkParser struct { ... }
type ServerSelector struct { ... }
type SlogAdapter struct { ... }
```

**New Functions:**
```go
func NewLinkParser(logger Logger) *LinkParser
func NewServerSelector(logger Logger, timeout time.Duration, maxConcurrent int) *ServerSelector
func NewSlogAdapter(logger *slog.Logger) Logger
func (s *ServerSelector) FetchRawLinks(rawURL string) ([]string, error)
func (s *ServerSelector) CheckLatency(link string) (time.Duration, error)
func (s *ServerSelector) SelectBest(links []string) (*ServerInfo, error)
func (s *ServerSelector) SelectBestFromURL(rawURL string) (*ServerInfo, error)
```

### Backward Compatibility

✅ All existing functionality remains unchanged
✅ Original usage (`goxray <vless://...>`) still works
✅ No breaking changes to existing API

### Usage Examples

**Command line:**
```bash
# Direct link (existing)
sudo go run . vless://uuid@server.com:443

# From raw list (new)
sudo go run . --from-raw https://example.com/links.txt
```

**As library:**
```go
selector := client.NewServerSelector(logger, 5*time.Second, 10)
best, _ := selector.SelectBestFromURL("https://example.com/links.txt")
vpn.Connect(best.Link)
```
