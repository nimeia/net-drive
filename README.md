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
- full WinFsp SDK dispatcher ABI glue, production Explorer-served Windows-host smoke logs, and full Windows file semantic coverage (callback bridge v1, tray shell, structured diagnostics, and WinFsp native preflight / dispatcher-v1 backend now exist)
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

The Windows client now also writes a structured local product log, runs a graded self-check, and exports a diagnostics ZIP with text/JSON summaries plus recent log tail content. On Windows, the mount runtime performs a real WinFsp host-binding preflight: it discovers the WinFsp DLL, records the launcher path when present, calls the native `FspFileSystemPreflight` API for the requested mount point, and reports both requested/effective backend plus dispatcher bridge state in the UI and diagnostics output. Dispatcher-v1 now includes a first callback bridge that warms up volume + root getattr paths before entering the host lifecycle. Iter 31 extends the read-only bridge to cover `Cleanup`, `Flush`, `GetSecurityByName`, and `GetSecurity`; Iter 32 adds Windows-host validation record templates to diagnostics/release artifacts so Explorer smoke and installer closure can be captured on a real machine. Iter 33 upgrades the WinFsp security path to emit richer native-style read-only descriptors, explicit cleanup/flush state, and explicit `CanDelete` / `SetDeleteOnClose` denial semantics for Explorer. Iter 34 extends validation templates so real Windows-host MSI install / upgrade / uninstall and EXE portable launch results can be backfilled into a structured validation record.

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

## Stress suite

Use the packaged stress wrapper to rerun the same pressure paths exercised in the sandbox:

```bash
./scripts/run-stress-suite.sh
```

Quick wrapper self-check without the full long run:

```bash
INTEGRATION_REPEAT=1 MIXED_REPEAT=1 RUN_BENCH=0 ./scripts/run-stress-suite.sh
```

On Windows PowerShell:

```powershell
./scripts/run-stress-suite.ps1
```

The wrapper writes raw logs to `dist/stress/` and covers:

- mixed browse/save/watch pressure
- concurrent create/write/rename/watch
- heartbeat interleaved with file operations
- repeated disconnect/resume/read
- metadata backend benchmarks

Current result summary and interpretation: `docs/architecture/test-report-iter43-stress-suite.md`

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

- Iter 25 Windows diagnostics grading / structured logs: `docs/architecture/windows-client-diagnostics-v2.md`
- Iter 26 WinFsp dispatcher callback bridge v1: `docs/architecture/windows-winfsp-dispatcher-bridge-v1.md`


- Iter 27 ABI bridge / dispatcher service loop v1: `docs/architecture/windows-winfsp-abi-bridge-v1.md`
- Iter 28 Explorer smoke / installer / crash recovery: `docs/architecture/windows-windows-host-smoke-installer-recovery.md`
- Iter 29 native callback table / Explorer request matrix: `docs/architecture/windows-winfsp-native-callback-table.md`
- Iter 30 MSI/EXE packaging + Windows host validation: `docs/architecture/windows-installer-msi-exe-validation.md`

Windows installer stage:

```powershell
./scripts/build.ps1
./scripts/package-windows-installer.ps1
```

Diagnostics export now includes `explorer-smoke.md`, `explorer-smoke.json`, `explorer-request-matrix.md`, `explorer-request-matrix.json`, `winfsp-native-callbacks.md`, `winfsp-native-callbacks.json`, `recovery.json`, `windows-host-validation-template.md`, `windows-host-validation-template.json`, `windows-host-validation-result-template.md`, `windows-host-validation-result-template.json`, `windows-host-backfill-patch-template.md`, `windows-host-backfill-patch-template.json`, `windows-release-closure-template.md`, `windows-release-closure-template.json`, `windows-pre-release-issues.md`, and `windows-pre-release-issues.json`.


Windows release packaging:

```powershell
./scripts/package-windows-release.ps1 -Version 0.1.0
```

This produces an EXE staging bundle, a WiX-based MSI source/output directory, a release validation checklist, host/installer result templates, a Windows-host backfill patch template, and pre-release issue templates that can be regenerated after each real Windows-host validation round.

- Iter 33 WinFsp native security descriptor / cleanup / delete-on-close semantics: `docs/architecture/windows-winfsp-native-security-delete-semantics.md`
- Iter 34 Windows host validation backfill / installer closure: `docs/architecture/windows-host-validation-backfill-installer-closure.md`

- Iter 35 final native callback matrix: `docs/architecture/windows-winfsp-native-callback-matrix-final.md`
- Iter 36 final Windows release closure: `docs/architecture/windows-release-final-closure.md`
- Iter 37 Windows host first-pass backfill: `docs/architecture/windows-host-first-pass-backfill.md`
- Iter 38 pre-release issues pipeline: `docs/architecture/windows-pre-release-issues.md`
- Iter 39 first-pass fixes intake/fix-plan: `docs/architecture/windows-host-first-pass-fixes.md`
- Iter 40 release candidate finalization: `docs/architecture/windows-release-rc-finalization.md`
- Iter 41 Windows host targeted fixes intake: `docs/architecture/windows-host-targeted-fixes.md`
- Iter 42 final release closure and signoff: `docs/architecture/windows-final-release-closure.md`

Windows final release packaging:

```powershell
./scripts/finalize-windows-release.ps1 -ReleaseDir .\dist\windows-release -ValidationResultJson .\dist\windows-release\windows-host-validation-result-template.json -CompletedBy "<tester>"
./scripts/package-windows-final.ps1 -ReleaseDir .\dist\windows-release
```

This now produces a validation intake report, final release status, and sign-off template alongside the RC metadata.


Finalize Windows release closure after backfilling the host validation result:

```powershell
./scripts/finalize-windows-release.ps1 -CompletedBy "<tester>"
```

- Iter 35 WinFsp native callback matrix final closure: `docs/architecture/windows-winfsp-native-callback-matrix-final.md`
- Iter 36 Windows release final closure: `docs/architecture/windows-release-final-closure.md`
- Iter 37 Windows host first-pass backfill: `docs/architecture/windows-host-first-pass-backfill.md`
- Iter 38 Windows pre-release issues: `docs/architecture/windows-pre-release-issues.md`


Apply a first-pass Windows host backfill patch:

```powershell
./scripts/backfill-windows-validation.ps1 \
  -ValidationResultJson .\dist\windows-release\windows-host-validation-result-template.json \
  -PatchJson .\dist\windows-release\windows-host-backfill-patch-template.json \
  -CompletedBy "<tester>"
```


## Windows host result backfill and RC packaging

- First-pass Windows host fixes: `docs/architecture/windows-host-first-pass-fixes.md`
- RC finalization: `docs/architecture/windows-release-rc-finalization.md`
- Backfill validation: `scripts/backfill-windows-validation.ps1`
- Finalize release closure / issue list / fix plan / RC: `scripts/finalize-windows-release.ps1`
- Package RC directory: `scripts/package-windows-rc.ps1`

## Sampled soak with runtime snapshot

Use the sampled soak runner to collect runtime snapshot and metadata lock-wait counters during a 3~5 minute mixed workload run:

```bash
./scripts/run-sampled-soak.sh
```

Quick plan / dry-run only:

```bash
go run ./cmd/devmount-soak -dry-run
```

On Windows PowerShell:

```powershell
./scripts/run-sampled-soak.ps1
```

The runner writes:

- `dist/stress/sampled-soak-samples.csv`
- `dist/stress/sampled-soak-report.md`

Additional status endpoint:

```bash
curl http://127.0.0.1:17891/runtimez
```

The runtime snapshot includes metadata/session/journal counts plus read/write lock wait counters for the server-side RW locks.
\n
## Iter 48 regression compare

To run the full local regression compare wrapper:

```bash
./scripts/run-regression-compare.sh
```

On Windows PowerShell:

```powershell
./scripts/run-regression-compare.ps1
```

Dry-run only:

```bash
DRY_RUN=1 ./scripts/run-regression-compare.sh
```

This wrapper uses the existing stress/soak scripts and then renders a unified markdown report at `dist/regression/regression-compare-report.md`.
\n