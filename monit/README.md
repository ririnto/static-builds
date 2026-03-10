# Monit Static Build

This target builds a static PIE `monit` binary and links required
libraries statically.

## Modules and Features

### Build Options (Explicit)

- Configure options: `--without-pam` and `--with-largefiles`.
- Build output is a static PIE `${TARGET_PREFIX}/bin/monit` from
  `src` and `libmonit/src` archives.
- Static linkage explicitly includes OpenSSL (`libssl.a`,
  `libcrypto.a`) and zlib (`libz.a`).

### Runtime/Packaging Snapshot

- Build-feature summary and command surface are captured in
  [Runtime Introspection Output](#runtime-introspection-output) with
  `monit -V` and `monit -h`.

## Allowed Target-Specific Variations

- This target intentionally relinks built object archives into a final
  static PIE `bin/monit` binary instead of relying only on the default
  upstream install output.
- Static linkage against OpenSSL and zlib archives is an approved part
  of this target profile.
- The approved release artifact for this target is `monit/bin/monit`.

## Runtime Defaults

Build features (confirmed from `monit -V` output):

- `Built with ssl, with ipv6, with compression, without pam and with
  large files`

Command line options and services (from `monit -h`):

- Default daemon mode runs in background
- Control file can be specified via `-c file`
- Daemon interval can be set via `-d n` (n seconds)
- Group name for commands can be set via `-g name`
- Log file can be specified via `-l logfile`
- PID file can be specified via `-p pidfile`
- State file can be set via `-s statefile`
- Foreground mode available via `-I` flag
- Batch command line mode available via `-B` flag
- Syntax check via `-t` flag
- Verbose modes: `-v` and `-vv`

Commands include: `start`, `stop`, `restart`, `monitor`, `unmonitor`,
`reload`, `status`, `summary`, `report`, `quit`, `validate`, and
`procmatch`.

## Runtime Introspection Output

The following sections show the exact runtime introspection outputs from
the built binary.

### monit -V

```text
This is Monit version 5.35.2
Built with ssl, with ipv6, with compression, without pam and with large files
Copyright (C) 2001-2025 Tildeslash Ltd. All Rights Reserved.
```

### monit -h

```text
Usage: monit [options]+ [command]
Options are as follows:
 -c file       Use this control file
 -d n          Run as a daemon once per n seconds
 -g name       Set group name for monit commands
 -l logfile    Print log information to this file
 -p pidfile    Use this lock file in daemon mode
 -s statefile  Set the file monit should write state information to
 -I            Do not run in background (needed when run from init)
 --id          Print Monit's unique ID
 --resetid     Reset Monit's unique ID. Use with caution
 -B            Batch command line mode (do not output tables or colors)
 -t            Run syntax check for the control file
 -v            Verbose mode, work noisy (diagnostic output)
 -vv           Very verbose mode, same as -v plus log stacktrace on error
 -H [filename] Print SHA1 and MD5 hashes of the file or of stdin if the
               filename is omitted; monit will exit afterwards
 -V            Print version number and patchlevel
 -h            Print this text
Optional commands are as follows:
start all             - Start all services
start <name>          - Only start the named service
stop all              - Stop all services
stop <name>           - Stop the named service
restart all           - Stop and start all services
restart <name>        - Only restart the named service
monitor all           - Enable monitoring of all services
monitor <name>        - Only enable monitoring of the named service
unmonitor all         - Disable monitoring of all services
unmonitor <name>      - Only disable monitoring of the named service
reload                - Reinitialize monit
status [name]         - Print full status information for service(s)
summary [name]        - Print short status information for service(s)
report [up|down|..]   - Report state of services. See manual for options
quit                  - Kill the monit daemon process
validate              - Check all services and start if not running
procmatch <pattern>   - Test process matching pattern
```

## How to Confirm Defaults

Build monit and inspect the feature set:

```bash
CI= GITHUB_ACTIONS= make build monit
./out/monit/bin/monit -V
./out/monit/bin/monit -h
```

The `-V` flag shows the build feature summary, and `-h` shows all
available command line options and commands. See the [Runtime
Introspection Output](#runtime-introspection-output) section above
for the actual output.

## How to Verify

> [!NOTE]
> In CI outputs are under `monit/`. Override with `BUILD_OUTPUT_DEST`.

```bash
./out/monit/bin/monit -V
```
