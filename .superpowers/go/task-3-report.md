# Task 3 Report — internal/daemon

## Status: DONE

## Commits
- `875dff1` feat(go): add internal/daemon with Unix socket and serial bridge

## Tests
All 5 tests PASS in 1.5s:
- TestDaemonForwardsValidState — valid state written to serial
- TestDaemonIgnoresInvalidState — invalid state silently dropped
- TestDaemonRepliesPONGOnPing — ping returns "PONG\n", nothing to serial
- TestDaemonRemovesSocketOnShutdown — socket file removed on ctx cancel
- TestDaemonAcceptsSequentialStates — four sequential states arrive in order

## Implementation Notes
- `portWriter` interface defined locally in package daemon (unexported), not imported from serial
- `runWithOpener` is unexported and fully testable via fakePort mock
- `Run` wires real `serial.FindPort()` + `goserial.Open` as the opener
- sync.Mutex guards all serial Write calls and the `current` pointer update on write error
- Accept loop uses goroutine+channel pattern to select on ctx.Done() without SetDeadline on listener
- Reconnect ticker fires every 2s in background goroutine, stops on ctx.Done()
- Shutdown: serial Close → listener Close → os.Remove(socketPath) via defer

## Concerns
None.

---

## Fix Report — Concurrency Hardening (commit `a6d2401`)

### Fix 1: Collapse double-lock in handle()
- **Problem:** Two separate lock/unlock sections around the nil check and the Write left a race window: another goroutine could nil `current` between the two critical sections, causing a write on a nil/closed port.
- **Fix:** Single lock section covers both the nil check and the Write. On nil, release the lock early and return; otherwise, write and release at the end.

### Fix 2: Release mutex before openPort() in reconnect loop
- **Problem:** `openPort()` (a potentially slow serial device open) was called while holding the mutex, blocking all concurrent writes for the duration.
- **Fix:** Check `current == nil` under lock, release, call `openPort()` outside the lock. Re-acquire to assign only if `current` is still nil; if another goroutine already reconnected, close the redundant port.

### Fix 3: Remove "ping" from validStates
- **Problem:** `"ping"` was listed in `validStates` even though it is handled (and returned early) before the map check — misleading and dead code.
- **Fix:** Removed `"ping"` from the map.

### Fix 4: Add defer cancel() in TestDaemonRemovesSocketOnShutdown
- **Problem:** Missing `defer cancel()` meant context cancellation was not guaranteed on test failure paths before the explicit `cancel()` call.
- **Fix:** Added `defer cancel()` immediately after `startDaemon`.

### Tests (post-fix)
All 5 PASS in 1.339s:
```
--- PASS: TestDaemonForwardsValidState (0.06s)
--- PASS: TestDaemonIgnoresInvalidState (0.06s)
--- PASS: TestDaemonRepliesPONGOnPing (0.04s)
--- PASS: TestDaemonRemovesSocketOnShutdown (0.16s)
--- PASS: TestDaemonAcceptsSequentialStates (0.19s)
```
