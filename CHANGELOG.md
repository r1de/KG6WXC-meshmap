# Updated Go and yet more UI (3.1.5)
Updated go version and depends.  
Changed the name of the github release files so they are consistent  
Attempt to fix some weirdness in how map tile servers were displayed. 🤞  
Stopped the duplicate node entries in the database. 🤞  
 - This requires a DB wipe and restart, Sorry.

Fixed Light/Dark mode.  
Cleaned up some typos (I am sure there are _many_ more.  

# More UI Updates (3.1.4)
moved zoom controls and map scale to bottom right.  
Added `Map Controls` for Tiles, Re-Centering, Fit Network, etc.  
Map, Link and legend controls now have 80% transparency.  

# version display (3.1.3.1)
Fixed my missed version change.

# More Changes to Web Interface (3.1.3)
Added current polling indication. Only show status on the actual "live" polling device (serve mode).  
A "serve-only" device will show that it is "serve-only" and will not show any indication of the current polling status.  

There is now the ability to force a polling run independently of the configured schedule.
Run the meshmap program again, as the same user it runs as but give the option of `--poll-now` of `--poll-in 10m`
The `--poll-in X` option accepts time frames of 1 minute to 10 minutes, values like `90s` work, as does `5m` or `233s`, it just has to be with 1-10 minutes.
Once the manual run is scheduled it takes precedent over the scheduled run, unless it is after the scheduled run.
The "Earliest Next Run" take precendent.
The new polling indicator will update to show the time until next run.

# Web Page Changes (3.1.2)
Show program version on the bottom right of the web page.  
  
Removed "Refresh Data" button since it didn't really do anything, it could return if it becomes useful again.  
It was mostly used when forcing a new data file to a remote "-serve-only" host.  
The page should auto refresh now when a new file is available.  
  
Added mouse pointer current co-ordinates in the page footer.  
The co-ords only display when the cursor is over the map.  
There could possibly be a right-click menu to copy the mouse co-ordinates, but not sure about implementing that yet.  
  
Fixed manual co-ordinate entry in the URL  

# Added Windows and Mac Binaries (3.1.1)
Added Windows and Mac Binaries  

# Clean up node popups (3.0.5)
Removed null values from the node informational popups.  
The values are still in the Database, they just do not clutter the map view any longer.  
Also made the pop ups fit a little better in their < div>.  
Moved the node icons z-axis (the circles) to hopefully have them sit on top of the link lines.  

# Bumped version to 3.0.3
Bumped major version number to better reflect that this is actually the 3rd version of my map.  
(It's probably really like the 37th version, but whatever. :-) )

#### (ChatGPT Generated below here)

# Persistent error logs and debug mode (0.3.3)

Version 0.3.3 keeps normal operation quiet and records warnings/errors in
`logs/error.log`. Use `--debug` or `debug_mode = true` to create one combined
`logs/debug.log` containing lifecycle events, request timings, memory/goroutine
heartbeats and native Go crash output. Debug mode also mirrors diagnostics to
the console. No separate `meshmap.log` or `crash.log` is created. New INI settings
are added automatically without resetting user values. See
[DIAGNOSTICS.md](DIAGNOSTICS.md) for rotation and configuration.

An advisory lock prevents overlapping runs sharing the same logs directory.
Normal runs are quiet; errors return a nonzero exit status. HTTP and database
I/O timeout defaults to 30 seconds per request; override with `--timeout 10s`.

# Automatic settings upgrades (0.3.1)

On startup the executable adds any missing active default settings to your
existing `user-settings.ini`. It preserves every existing value, even when that
value is empty, false, or equal to an older default. Existing comments, spacing,
line endings, unknown settings and section order remain unchanged. Added entries
are grouped at the end with explanatory default comments; a repeated
`[user-settings]` header is normal and avoids adding entries under an unrelated
section. Existing scalar and array values are never reset. Single/double-quoted
array keys are recognized as the same setting.

There is no need to replace your settings file or copy a new template over it.
Missing keys come from the executable's embedded defaults, with an optional
sibling `user-settings.ini-default` retaining its existing override role. An old
sibling default file cannot hide newly introduced embedded settings. Defaults
that are only commented-out examples are not enabled automatically.

The first upgrade needs permission to replace the settings file in its directory.
Changes are locked against concurrent MeshMap starts and written by atomic
replacement, preserving file ownership, permissions and any symlink. Complete
settings files are not rewritten or locked, so a read-only completed configuration
can be used. If missing defaults cannot be written, startup reports an error
rather than pretending the settings were saved. Malformed INI files are not
rewritten. Configuration changes take effect on restart, not during a running
poll cycle.

These server/runtime settings are now available in the INI:

```ini
run_mode = "poll"
poll_interval = "4h"
http_timeout = "30s"
listen_address = "127.0.0.1:8088"
tls_cert = ""
tls_key = ""
```

For example, to run the combined server/poller and wait 15 minutes after each
completed polling run:

```ini
run_mode = "serve"
poll_interval = "15m"
listen_address = "0.0.0.0:8088"
```

Then start `./meshmap` with no extra server flags. `run_mode` accepts `poll`
(one-shot, the backward-compatible default), `serve`, or `serve-only`.
Durations accept units such as `30s`, `15m` or `1h`; the interval starts after
completion, so lengthy polls do not overlap. `http_timeout` sets the per-node
HTTP budget across endpoint fallbacks and the database I/O timeout.

Explicit CLI options still win: `--serve`, `--serve-only`, `--poll-interval`,
`--listen`, `--timeout`, `--tls-cert` and `--tls-key` override their corresponding
INI values for that invocation only. They are never persisted. One-shot
maintenance/test flags (`--data-files-only`, `--test-mode-no-sql`,
`--test-mode-with-sql`) also select one-shot mode unless a server-mode flag is
explicitly supplied. `--init` creates the settings file or adds missing defaults
to an existing file while preserving saved values. Normal startup also adds
missing defaults.

# Serve the embedded map (0.3.0)

For a new installation, initialize and edit settings as above, then run:

```sh
./meshmap --serve
```

Open `http://127.0.0.1:8088/`. The server stays running while the poller works in
its own goroutine. The first cycle preloads existing database records so an
upgraded installation can display its stored map while the next poll runs.
By default, further cycles start four hours after the preceding cycle finishes. Do not also run the old cron job when using combined mode.

```sh
# Explicitly listen on all IPv4 interfaces; choose your own port.
./meshmap --serve --listen 0.0.0.0:8088 --poll-interval 4h
# Keep a separate cron-driven poller; this process only serves completed files.
./meshmap --serve-only --listen 0.0.0.0:8088
# The original one-shot polling behavior remains the default.
./meshmap
```

Serving-only does not open the database or contact nodes. If upgrading from
0.2.0, first run `./meshmap --data-files-only` to generate `public_map.json` from
existing records. Both serving modes follow later atomic snapshot replacements.
A failed or canceled poll retains the last completed public snapshot. Web
requests never trigger scans, execute SQL, or modify configuration. Ctrl+C or
SIGTERM cancels polling and gracefully shuts down the server.

Direct HTTPS accepts a certificate and private key in PEM format:

```sh
./meshmap --serve --listen 0.0.0.0:8443 \
  --tls-cert /path/to/fullchain.pem --tls-key /path/to/privkey.pem
```

Use a certificate valid for your public hostname. Certificate issuance/renewal
is external in this release; restart the executable after replacing certificates.
An existing HTTPS reverse proxy is also supported, with the executable bound to
loopback. No proxy is required for direct HTTPS. Serve at the website root;
subdirectory/prefix hosting is not implemented. Run under an unprivileged account
on a high port such as 8088/8443. The default address is loopback, not public.

The server sets read/header/write/idle timeouts, a 16 KiB header limit, restrictive
script policy, framing protection, and HTTPS HSTS. It serves only embedded assets
and a read-only `/api/map` projection. It never serves the working directory,
configuration, database, logs, raw observations or full private telemetry exports.
Node-supplied strings render as text; service URLs allow HTTP/HTTPS only. Public
map data intentionally includes station names, locations, IPs, services and
selected metrics. It has no authentication or administration endpoints. Host it
only where that map data is intended to be public, and keep builds updated.

`webpageDataDir` now also holds the completed `public_map.json` snapshot; it need
not be a web directory. Prefer a private writable directory outside any other
web server's document root. The embedded server caches the completed snapshot in
memory (64 MiB maximum) and supports ETags. Browsers check for updates every
30 seconds without resetting their map position; a serving-only process detects
new snapshots within five seconds. Assets and scripts are bundled. Map tiles
remain external/configurable and follow their provider's availability and usage
policy; no offline tile bundle, tile proxy, bulk downloader or LOS/elevation tool
is included. Existing relative local-tile paths require a separately hosted tile
service URL, because the embedded server does not expose arbitrary directories.

The familiar full-screen map retains the band colors, node popups, layer filters,
map-tile choice and node reports. Search by node name or IP, use Fit network or
Map origin, and inspect co-located nodes from the popup's location group. Node
and link reports have filtering, sorting, 100-row pages, and CSV downloads.
Nodes without coordinates remain in reports. Link lines require coordinates for
both endpoints; observed directions and parallel trackers remain separate, with
an offset for visibility. Tunnel lines are dashed and DtD lines dotted.

Link coloring defaults to reported TX radio rate, from red through yellow to
green. Select 30, 50 or 100 Mbps as the green endpoint, or switch to RX rate,
SNR, Hello/TX/reverse quality, RTT, directional Babel costs, route summaries or
legacy OLSR cost/quality. Unreported metrics are gray, never invented from route
costs. For latency/cost, lower values are green. Each link popup shows its source,
reported metrics, counters and cached-measurement age. PHY rates are not measured
application throughput; route metric summaries are not direct-link ETX.

For continuous operation, see the systemd example in `deploy/meshmap.service`.

# Update to 0.2.0: LQM telemetry

Replace the executable and retain your configuration and database. `./meshmap
--version` identifies this build. No node-side package, exporter, or firmware
change is required. Stop the previous polling process before replacing it.

On first database access, the backend automatically adds a nullable `lqm` column
when missing. On MariaDB it also widens `link_info` to LONGTEXT, because richer
link data can exceed the old 64 KiB TEXT limit. Existing records are retained;
the MariaDB account needs ALTER permission. Back up an existing database before
upgrading. **Do not reimport schema.sql into an existing installation.**

Polling now requests `lqm=1` in the same sysinfo request. Old nodes that omit the
section continue to work; nodes that reject the optional query are retried with
the previous query. Network failures do not cause that extra retry. No active
speed tests or changes to routing settings are performed.

All returned LQM fields are retained, including unknown future fields, nulls,
zero counters, reverse observations, configuration and sampling clocks. Depending
on firmware/hardware, these include:

- TX/RX physical radio bitrates, signal, SNR, retries and failures.
- Hello receive quality (`lq`, `avg_lq`, `rev_lq`), TX quality, ping quality,
  combined quality and reverse measurements.
- `rxcost`, `txcost`, RTT, route counts, route metric summaries and interface costs.
- Tracker interface, MAC, IPv6 link address, canonical IPv4, state, last-seen and
  uptime fields, plus the source `info.now`/`start` clocks when reported.

Data is stored as JSON in `node_info.lqm`. The full source object is available
in `link_telemetry.json` for every stored node, including nodes without map
coordinates. Matched trackers also appear at
`link_info[peer_ip].lqm_trackers[tracker_id]`, retaining separate trackers for
parallel interfaces. Matching prefers an existing canonical/link IP; a unique
hostname/interface match is used when IP association is unavailable. Unmatched,
blocked or stale source trackers remain in `lqm.info.trackers`, and never create
new discovery targets on their own.

`link_telemetry.json` has this structure:

```json
{
  "schema_version": 1,
  "data": [{
    "node": "CALLSIGN-NODE",
    "wlan_ip": "10.1.1.1",
    "last_seen": "2026-10-29 12:00:00",
    "lqm": {"enabled": true, "info": {"trackers": {}}},
    "link_info": {}
  }]
}
```

The LQM and tracker values retain the firmware's field names and types. For
example, some versions call route summaries `metric`/`route_count` and others
`babel_metric`/`babel_route_count`; both are retained without conflating them.
Radio bitrate is not measured application throughput. `lq` is Hello reception
quality; the composite `quality` and retry-based `tx_quality` are different
measurements. Route-summary `metric` is not automatically a direct link cost.
No synthetic ETX or conversions into legacy OLSR cost fields are added.

`last_seen` timestamps the last successful node poll in UTC; LQM itself contains
cached measurements. Use its sampling clocks and tracker/reverse last-seen values
when assessing age; reverse observations can be older than local observations.
A failed poll retains earlier node LQM with its old `last_seen`. A successful
poll with no LQM clears earlier LQM instead of presenting it as current.

The existing `map_data.js` and node-report formats exclude the added telemetry
to keep the old webpage's downloads small. The full measurements are in the
new file and `node_observations.json`; data-files-only regenerates the new file
from the database.

# Initial port from PHP (0.0.1)
## Old OLSR, modern Babel, and mixed networks

Discovery tries `/a/sysinfo?nodes=1&topology=1`, historical
`/cgi-bin/sysinfo.json` on ports 80 and 8080, and finally
`:8080/cgi-bin/api?mesh=topology`. Both top-level OLSR topology and the old
`pages.mesh.topology.topology` wrapper are supported. Node polling tries current
and historical sysinfo endpoints on both ports and parses old named interfaces,
new interface arrays, old flat fields and nested node details.

Modern API 2.0 discovery uses `nodes=1`, which advertises canonical mesh nodes.
It does **not** poll the `hosts=1` list of LAN devices and service hosts. Each
polled node contributes its direct `link_info` neighbors, and older nodes can
contribute additional OLSR topology. Newly discovered addresses are polled once
per run, using the same bounded worker pool. `max_nodes` (default 10000) limits
discovery; exceeding it aborts before replacing database polling results.

For networks spanning separate visibility domains, set extra origins:

```ini
discovery_seeds = "old-origin.local.mesh,new-origin.local.mesh"
```

An origin cannot discover networks it cannot see. Additional seeds must be
reachable from the polling machine. A failed configured seed aborts the run
rather than silently publishing partial discovery.

OLSR path costs/hops are retained when provided. Babel is not treated as an
OLSR topology export: the map uses reported neighbor links, not inferred routing
paths. Babel does not expose hop counts through this API; those remain NULL in
the database and display as `unknown` in map data. `highestHops` summarizes only
available OLSR hop information (zero when none was supplied). Link metrics are
retained under the API's own names; absent OLSR cost/quality or throughput values
are not invented. Some older frontend metric layers therefore have no useful
values for current firmware; use available signal/noise and neighbor links.
RF distance/bearing is calculated only when both endpoints have usable locations.

The old frontend has one radio per node. It receives the primary radio; additional
radios, new fields, and all successful API observations are retained in
`node_observations.json`. That companion file is a snapshot of the current run,
not historical storage. Support is based on observed API shapes, not a guarantee
that every historical or future firmware release has been tested.

## MariaDB / MySQL

Set these in the same configuration file:

```ini
SQL_TYPE = "mysql"
sql_server = "localhost:3306"
sql_user = "mesh-map"
sql_passwd = "your password"
sql_db = "node_map"
sql_db_tbl = "node_info"
```

`SQL_TYPE = "mariadb"` is an alias. `sql_server` can also be a Unix socket path.
For a **new empty database only**, create a database/user and import `schema.sql`
with the MariaDB client. **That import drops existing tables.** For an existing
PHP database, back it up and retain its existing schema instead. The `hardware`
column from the original project's update must be present. PHP-serialized array
columns are retained for compatibility with existing database consumers.

Switching SQL_TYPE selects another database; it does not migrate history between
SQLite and MariaDB. Stop cron before copying a database. SQLite backups should
use an SQLite backup tool or copy the database after all polling processes stop
(including any WAL files if present).

PostgreSQL is not included in this version: supporting and validating a third
schema/migration path would expand the port beyond the two requested backends.

## Outputs and modes

- `map_data.js`: `arednVersions`, `mapInfo`, `pollingInfo`, `allDevices`.
- `node_report_data.json`, `error_report.json`, `no_location.json`: existing report shapes.
- `link_telemetry.json`: full persisted LQM and per-link tracker associations.
- `node_observations.json`: successful modern/legacy observations, including extra radios.
- `logs/`: polling summaries, failures, and missing locations.

Output files are replaced atomically one at a time, not as a single generation.
Nodes failing polling retain their prior last-seen time; stale links are cleared.
Expiry is based on last successful polling and occurs only when enabled.
The PHP test mode's accidental database accesses are removed:

```sh
./meshmap --test-mode-no-sql
./meshmap --data-files-only
```

No-SQL mode still writes logs/output files, so use a separate output directory
for testing. Data-files-only regenerates from the selected database without
network access and retains previous reports/origin when present. It leaves the
observations snapshot untouched. Optional SSH publishing uses `scp` and `ssh` with key authentication to publish
`public_map.json`. It is disabled by default and skipped in no-SQL mode.


# Provenance

Port of Eric Satterlee / KG6WXC's KG6WXC-meshmap (2016–2025), upstream commit
`56c861d08d7f5bafc90c605af8aadb1d749f9114`. Hardware names and default settings
are derived from that project. Licensed GPL-3.0-or-later; see LICENSE.

The embedded frontend retains layout/branding from KG6WXC-meshmap-webpage, commit
`0c466a043cb2f6af7a918b9257fcb7bd09b0227b`, with new safe DOM rendering and telemetry
controls. Bundled Leaflet and PolylineOffset licenses are in web/vendor; see
web/NOTICE.txt.

Current API behavior was researched against the AREDN Project's sysinfo and mesh
API implementations at commit `d3662953a86be92a6f3b1e7995ac09168cee2d30`:
https://github.com/aredn/aredn/blob/d3662953a86be92a6f3b1e7995ac09168cee2d30/files/app/main/sysinfo.ut
and the integrator documentation:
https://docs.arednmesh.org/en/latest/arednHow-toGuides/devtools.html
This is an independent backend port, not AREDN firmware.
