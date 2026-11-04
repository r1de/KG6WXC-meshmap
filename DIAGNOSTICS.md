# Error logging and optional debug mode

Version 0.3.3 is quiet during normal operation and uses two diagnostic files:

| File | When used | Contents |
| --- | --- | --- |
| `error.log` | Always | Everyday warnings/errors: failed cycles, startup failures, snapshot reload failures and HTTP server errors. |
| `debug.log` | Only when debug is enabled | Lifecycle events, version/PID/mode, shutdown signals, polling phases/results, retry scheduling, upload publishing, request timings/status/sizes, memory/goroutine heartbeats and native Go crash output. |

Debug mode mirrors diagnostic messages to the console. Without debug, ordinary
polling, serving, warnings, errors and graceful shutdown do not produce console
output; warnings/errors remain in error.log. One-shot test-mode progress and
summaries also require debug. Explicit help/version commands still print their
requested output. If errors cannot be logged at all, an essential stderr message
is retained; native fatal Go runtime output also retains its normal stderr
behavior. Exit status remains nonzero for failing one-shot/startup commands.

The older meshmap.log and crash.log files are no longer created or written.
Existing files and archives are left untouched so earlier evidence is retained.
The previously reported overnight stops were traced to an external server
reboot script; the diagnostics remain available for future troubleshooting.

## Enable debugging

Rebuild and restart using your normal settings and arguments, adding --debug:

```sh
make build
./meshmap --config settings/user-settings.ini --serve --debug
```

To save the preference in user-settings.ini:

```ini
debug_mode = true
```

Use debug_mode = false for normal quiet service operation. --debug=false
overrides a saved true value for one invocation. CLI overrides never rewrite
saved settings. Restart after changing settings.

New INI settings are appended automatically without resetting customized
values. Debug mode does not alter polling timeouts, discovery rules, the run
mode, database writes, or uploads. The archived speedtest queue is not included.

## Location, privacy and rotation

Defaults:

```ini
debug_mode = false
log_directory = ""
log_max_size_mb = 10
log_backups = 3
debug_heartbeat_interval = "1m"
```

By default, logs are in logs/ beside the installation's settings/ directory.
An absolute log_directory selects that location; a relative path is relative
to the installation directory containing settings/. Give separate processes
separate log directories when running them from the same installation: rotation
coordinates goroutines within one process, not independent processes.

Files append across polling runs and restarts. Both files rotate by size into
.1, .2, etc.; the default retains three archives per file. log_backups=0
retains no archives. In debug mode native crash capture follows rotation to the
current debug.log. Fatal runtime output is written directly by Go and can exceed
the size threshold before the process exits. A crash concurrent with rotation
can leave output in both the current file and an archive; retain both.

New files have mode 0600 and the embedded web server does not publish them.
Structured diagnostics redact known SQL credentials and do not dump HTTP
response bodies or full configuration. Native crash traces can contain extra
process details; review them before sharing publicly.

The existing polling_output.log, polling_errors.log and no_location.log still
describe the most recent run. They are separate from the appended diagnostic
history. Individual unavailable nodes remain detailed in polling_errors.log
and, when enabled, debug.log.

If configuration loading fails before a custom log directory can be loaded,
startup errors use the default logs directory when writable. An unwritable
diagnostic directory prevents startup. Later write failures report to stderr
so loss of logging is not silently hidden.

## Reading debug output

Each polling cycle has a poll_id. Request/node records include durations and
errors. Heartbeats and phase-change samples include the active phase and its
elapsed time, completed/known targets, uptime, goroutines, Go heap allocation
and in-use sizes, runtime-managed memory and GC count. These are Go runtime
memory figures, not total OS resident memory.

Ordinary polling errors in serve mode are logged and retried after the
configured interval. Panics retain their normal failing exit behavior; debug
mode copies native panic/fatal output, including worker-goroutine failures,
into debug.log. There is no separate crash log or startup-only crash header.

SIGINT, SIGTERM and SIGHUP request graceful shutdown. In debug mode the signal
and shutdown are recorded. A shell-launched background process can still stop
when its terminal/SSH session closes. The existing deploy/meshmap.service
example runs independently of a terminal and restarts on failures; it does not
enable debug mode unless configured to do so.

OS memory kills, SIGKILL, reboots and power loss do not produce Go panic output.
Use the system/service journal for those cases. When troubleshooting with debug
enabled, retain error.log, debug.log and their archives.
