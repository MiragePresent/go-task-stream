# Changelog

All notable changes to this project should be documented in this file.

## Unreleased

### Added
- N/A

### Changed
- N/A

### Fixed
- N/A

### Breaking Changes
- N/A

## v0.0.3 - 2026-03-04

### Added
- `consumer/loop`: `ConsumeWithErrors` and per-message `ErrorHandler` callback.
- `consumer/pipeline`: `ConsumeWithErrors` and per-message `ErrorHandler` callback.
- `adapter/memory`: coverage for applying stream options during open.

### Changed
- `consumer/loop`: `Consume` now continues processing after handler failures (nacking failed messages first).
- `consumer/pipeline`: `Consume` now continues processing after step/router failures (nacking failed messages first).
- `options`: exported `StreamConfig` so stream options can be interpreted by adapters.
- `adapter/memory`: stream options are now applied rather than rejected.
- docs expanded for memory adapter behavior and consumer package error-handling semantics.

### Fixed
- surfaced nack transport errors in loop/pipeline consumer tests.

### Breaking Changes
- N/A

Template for future entries:
- **What changed:** <short description>
- **Impact:** <what downstream users must update>
- **Migration:** <exact old -> new usage>
