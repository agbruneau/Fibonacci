// Package progress defines the types and machinery used to report the
// progress of a Fibonacci calculation from the algorithm layer to the user
// interface.
//
// It exposes:
//
//   - ProgressUpdate: the DTO flowing over channels from calculators to UI.
//   - ProgressCallback: a lightweight functional contract used by the core
//     algorithms, decoupling them from any specific transport.
//   - An Observer-pattern pair (ProgressObserver / ProgressSubject) that lets
//     multiple consumers (TUI dashboard, logger, metrics sink) subscribe to
//     the same stream of updates without tight coupling.
//   - ProgressReportThreshold: the minimum fractional change (default 1 %)
//     below which updates are coalesced, to avoid flooding the UI.
//
// The package has no dependency on any UI library; it is deliberately a pure
// Go domain layer.
//
// A-19 — production path: the only path exercised in production is the
// snapshot-based ProgressSubject.Freeze(calcIndex) (lock-free fan-out with a
// per-observer recover). The dynamic Register/Unregister/Notify API is a
// retained extension surface used by tests and embedders; it is NOT on the
// hot path. New contributors should wire progress through Freeze, not Notify.
package progress
