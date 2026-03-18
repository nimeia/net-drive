# Iter 34 — Windows host validation backfill / installer closure

## Goal

Make the Windows host validation template usable as a real release sign-off record, not only as a static checklist.

## Delivered scope

- `HostValidationRecord` now includes:
  - installer run entries for MSI install / upgrade / uninstall
  - EXE portable launch result entry
  - summary counters and overall status
  - completed-at / completed-by metadata
  - helper methods to backfill scenario, checklist, and installer-run results
- diagnostics export now bundles:
  - `windows-host-validation-template.*`
  - `windows-host-validation-result-template.*`
- release manifest now advertises:
  - validation template file
  - installer result slots expected for sign-off

## Intended workflow

1. Generate release assets and validation templates in the sandbox.
2. Run MSI/EXE validation and Explorer smoke on a real Windows host.
3. Backfill the JSON/Markdown validation record with actual results.
4. Archive the completed record alongside diagnostics ZIP files and installer logs.

## Boundary

This iteration does not claim that MSI/EXE install or Explorer smoke were executed inside the sandbox. It prepares the structured material needed to record those results on a real Windows host.
