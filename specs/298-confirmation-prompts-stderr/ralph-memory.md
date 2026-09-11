# Ralph Memory

Feature: 298-confirmation-prompts-stderr
Started: 2026-09-11T06:40:57Z

## Codebase Patterns

- `testx/cmd_subprocess_runner.go` selects helper subprocesses exactly with `-test.run=^<quoted name>$` and sets both `GO_WANT_HELPER_PROCESS=1` and `C8VOLT_TEST_HELPER_PROCESS_NAME`.
- Terminal acceptance must attach only child stdin to a PTY; stdout and stderr remain independent captures, and each response waits for its complete stderr prompt.
- Linux PTY allocation uses `/dev/ptmx`, `TIOCSPTLCK`, and `TIOCGPTN`; the slave stays canonical while `ECHO` and `ECHONL` are cleared.
- Darwin PTY allocation obtains the slave name through `TIOCPTYGNAME`, then grants and unlocks it before opening; it uses `TIOCGETA`/`TIOCSETA` for canonical no-echo input.
- Non-Linux/Darwin targets select a build-constrained allocator that reports an explicit unsupported-platform reason and has no Unix dependency.
- `testx.NewCmdTerminalRunner` re-executes the exact helper test with only stdin attached to the PTY, observes synchronized stderr snapshots, and advances a search offset so repeated prompt text cannot consume a stale occurrence.
- The runner closes the parent's slave descriptor after start, drains the no-echo PTY master, and kills then reaps the child before closing descriptors on every timeout/error return.
- Both confirmation seams now have the signature `(io.Writer, bool, string) error`; production callers pass `cmd.ErrOrStderr()`, while migrated test stubs may ignore the writer until their story-specific routing assertions are added.
- Native Darwin/arm64 foundation validation with Go 1.26.2 discovers all six terminal allocator/runner tests and passes both the terminal-support and focused command suites.

## Decisions

- `testx/cmd_terminal_runner.go` defines the shared request/result, exchange, timeout, allocator, and runner contracts. Platform allocators provide a master/slave pair plus the canonical EOF control byte.
- Unsupported platforms are represented explicitly with `Supported=false` and a reason; allocation failures on supported Linux/macOS targets remain errors.

## Gotchas

- `cmd/ops_execute_smoketest.go` contains an actual `confirmCmdOrAbortFn` call, not only a related comment, so it belongs in the T006 migration.
- The exact T006 production and test migration inventory is recorded in `quickstart.md`; use it together with a fresh `rg` so new references cannot be missed.

## Reusable Commands

- `rg -l 'confirmCmdOrAbort|confirmProcessDefinitionSelectorListVisibleFn' cmd --glob '*.go' --glob '!*_test.go' | sort`
- `rg -l 'confirmCmdOrAbort|confirmProcessDefinitionSelectorListVisibleFn' cmd --glob '*_test.go' | sort`
- `go test ./cmd -run 'Confirm|Paging|Selector' -count=1`
- `go test ./testx -count=1`
- `GOOS=linux GOARCH=arm64 go test -c -o /tmp/c8volt-testx-linux.test ./testx`
- `docker run --rm -v /tmp/c8volt-testx-linux.test:/testx.test:ro alpine:3.22 /testx.test -test.run '^TestLinuxCmdTerminalAllocator' -test.v`

## Do Not Repeat

- Pipe-backed stdin is not acceptable proof for FR-008; the later runner and acceptance tests must verify that child stdin is an actual terminal.
- A successful helper test that returns normally adds the Go test harness `PASS` line to stdout; helpers requiring byte-exact stdout can call `os.Exit(0)` after their child assertions succeed.

## Current Handoff
- Start T008 in US1 by adding real-terminal default-no helper coverage in `cmd/cmd_confirmation_terminal_test.go`; prompt writes intentionally remain on stdout until T010 so routing assertions should first demonstrate the defect.
