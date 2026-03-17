# Developer Mount

Developer Mount is a Windows-first, editor-optimized remote filesystem project.

This repository currently contains:
- protocol and architecture documents for Iter 0
- a control-plane baseline from Iter 1
- read-only metadata and file access from Iter 2
- metadata cache and directory snapshot logic from Iter 3
- write/save-path baseline from Iter 4
- watcher and journal polling baseline from Iter 5
- recovery and reconnect baseline from Iter 6
- editor-focused optimizations from Iter 7
- productization closure: config loading, status endpoints, audit logging, and packaging scripts

## Current scope

The current code implements:
- control channel: hello / auth / create session / resume session / heartbeat
- metadata channel: lookup / getattr / opendir / readdir / rename
- data channel: open / create / read / write / flush / truncate / set-delete-on-close / close
- events channel: subscribe / poll events / ack / resync snapshot
- recovery channel: recover handles / revalidate nodes / resubscribe watches
- metadata cache / negative cache / dir snapshot cache / root prefetch
- workspace profile / hot dir-file prefetch / small-file cache / priority-aware prefetch
- product-facing runtime pieces: JSON config, `/healthz`, `/status`, JSONL audit log, build/package scripts
- clientcore split for future Windows mount integration: tracked runtime state, handles, watches, and recovery snapshots
- WinFsp-facing read-only mount core: Windows path normalization, mountcore path/handle orchestration, callback bridge, and Windows-tagged host shell

It does not yet implement:
- full WinFsp SDK dispatcher glue and real Explorer-served Windows-host smoke logs (callback bridge, tray shell, diagnostics export, and WinFsp native preflight / dispatcher-v1 backend scaffold now exist)
- push-style watcher streaming
- lease / oplock-style invalidation
- full Windows file semantic coverage
- full handle replay semantics

## Repository layout

```text
cmd/
  devmount-server/
  devmount-client/
  devmount-winfsp/
  devmount-client-win32/
configs/
docs/
  architecture/
  protocol/
internal/
  client/      # compatibility wrapper for the demo CLI
  clientcore/  # protocol-facing runtime core for future WinFsp mount work
  mountcore/  # platform-neutral mount runtime for WinFsp-facing flows
  winclientruntime/  # product-facing mount runtime state machine for the Windows shell
  platform/windows/  # Windows-specific path helpers and later semantic translators
  protocol/
  winfsp/
    adapter/  # thin WinFsp-shaped adapter over mountcore
  server/
  transport/
scripts/
tests/
  integration/
benchmarks/
Task.md
```

## Build

```bash
go build ./...
./scripts/build.sh
```

On Windows:

```powershell
./scripts/build.ps1
```

## Test

```bash
go test ./...
```

核心稳态专项：

```bash
go test ./tests/integration -run 'TestControlPlaneNegative|TestSessionGatingAndProtocolErrorMapping|TestRecovery|TestMultiClientConcurrentCreateWriteRenameAndWatch|TestHeartbeatInterleavesWithFileOperations|TestConnectionJitterRepeatedResumeAndRead|TestRealisticMixedBrowseSaveWatchPressure|TestServerAndClientBinarySmoke'
go test ./internal/server -run 'TestMetadataBackend|TestJournal|TestLoadServerConfig|TestSnapshotStatus|TestAuditLogger'
go test ./internal/client
go test ./internal/winclientstore
go test ./internal/winclientruntime
go test ./internal/transport -run 'TestEncodeDecodeFrameRoundTrip|TestDecodeFrameNegativePaths'
go test ./internal/benchgate
```

性能基线：

```bash
go test ./internal/server ./internal/transport -bench . -run '^$'
./scripts/benchmark_gate.sh
```

WinFsp read-only smoke:

```bash
go run ./cmd/devmount-winfsp -op volume
go run ./cmd/devmount-winfsp -op getattr -path /
go run ./cmd/devmount-winfsp -op readdir -path /
go run ./cmd/devmount-winfsp -op read -path /README.md -length 64
go run ./cmd/devmount-winfsp -op materialize -path / -local-path ./devmount-local
go run ./cmd/devmount-winfsp -op selfcheck
go run ./cmd/devmount-winfsp -op export-diagnostics -diagnostics-path ./devmount-diagnostics.zip
```

Win32 config test UI build:

```powershell
go build -ldflags="-H windowsgui" -o .\dist\devmount-client-win32.exe .\cmd\devmount-client-win32
```

The Win32 client now has a product-shaped shell with `Dashboard / Profiles / Diagnostics` pages. Profiles stores named connection and mount defaults under the user config directory, Dashboard surfaces the live mount runtime state machine and mount quick actions, and Diagnostics keeps the advanced `volume|getattr|readdir|read|materialize` smoke tools plus CLI previews. Closing or minimizing the window keeps the client alive in the notification area, where the tray menu can reopen pages, start or stop the mount runtime, and export diagnostics. The `materialize` flow still recursively downloads the remote tree into a local folder so you can inspect it with Explorer, VS Code, or other Windows tools.

The Windows client now also writes a local product log, can run a built-in self-check, and can export a diagnostics ZIP with text/JSON summaries plus recent log tail content. On Windows, the mount runtime performs a real WinFsp host-binding preflight: it discovers the WinFsp DLL, records the launcher path when present, calls the native `FspFileSystemPreflight` API for the requested mount point, and reports both requested/effective backend plus dispatcher-v1 scaffold state in the UI and diagnostics output.

Windows-only host shell compile check:

```bash
GOOS=windows GOARCH=amd64 go build ./internal/winfsp
GOOS=windows GOARCH=amd64 go build ./cmd/devmount-winfsp
```

真实场景混合压力：

```bash
go test ./tests/integration -run TestRealisticMixedBrowseSaveWatchPressure -count=1 -v
```

## Productized startup

```bash
go run ./cmd/devmount-server -config ./configs/devmount.example.json
```

Health and status:

```bash
curl http://127.0.0.1:17891/healthz
curl http://127.0.0.1:17891/status
```

## Demo

Start the server with direct flags:

```bash
go run ./cmd/devmount-server -root /path/to/workspace
```

In another terminal, run the demo client:

```bash
go run ./cmd/devmount-client
```

## Focused reports

- Iter 6 recovery matrix: `docs/architecture/test-report-iter6-recovery-matrix.md`
- Productization closure build report: `docs/architecture/test-report-productization.md`

- Iter 9 core hardening report: `docs/architecture/test-report-iter9-core-hardening.md`
- Iter 10 second-round coverage report: `docs/architecture/test-report-iter10-second-round-coverage.md`
- Iter 11 concurrency / transport-negative / smoke / benchmark-gate report: `docs/architecture/test-report-iter11-concurrency-gate.md`
- Iter 12 ReadDir fast-path optimization and realistic pressure report: `docs/architecture/test-report-iter12-readir-pressure.md`

- Iter 13 Windows client core / WinFsp integration plan: `docs/architecture/windows-client-core-and-winfsp.md`
- Iter 14 WinFsp read-only MVP boundary: `docs/architecture/windows-winfsp-readonly-mvp.md`
- Iter 15 WinFsp callback host / build tags: `docs/architecture/windows-winfsp-callback-host.md`
- Iter 16 Win32 config test UI: `docs/architecture/windows-win32-config-ui.md`
- Iter 17 Local materialize bridge for Windows testing: `docs/architecture/windows-local-materialize.md`

- Windows client productization plan: `docs/architecture/windows-client-productization-plan.md`
- Iter 18 Win32 client profile persistence baseline: `docs/architecture/windows-client-profile-persistence.md`
- Iter 19 Windows client shell pages: `docs/architecture/windows-client-shell-pages.md`
- Iter 20 Windows client mount runtime state machine: `docs/architecture/windows-client-mount-runtime.md`
- Iter 21 Windows client tray / background runtime: `docs/architecture/windows-client-tray-runtime.md`
- Iter 22 WinFsp binding preflight in mount runtime: `docs/architecture/windows-winfsp-binding-runtime.md`
- Iter 23 Windows client logs / self-check / diagnostics export: `docs/architecture/windows-client-diagnostics.md`
- Iter 24 WinFsp dispatcher host v1 scaffold: `docs/architecture/windows-winfsp-dispatcher-v1.md`
