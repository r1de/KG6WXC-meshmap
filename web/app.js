/* SPDX-License-Identifier: GPL-3.0-or-later
 * Modernized from the KG6WXC MeshMap webpage (Eric Satterlee, 2016–2025).
 */
"use strict";
const M = MeshMetrics,
  $ = (id) => document.getElementById(id);
const bands = {
  900: ["900 MHz", "#b82ba7"],
  "2ghz": ["2.4 GHz", "#8045be"],
  "3ghz": ["3.4 GHz", "#287bc6"],
  "5ghz": ["5.8 GHz", "#c79b18"],
  noRF: ["No RF", "#7c8991"],
  supernode: ["Supernode", "#d75050"],
  other: ["Other RF", "#399a87"],
};
const types = { RF: "RF", DTD: "DtD", TUN: "Tunnels", XLINK: "XLink", OTHER: "Other" };
const linkColors = { TUN: "#3366cc", DTD: "#9933cc", XLINK: "#00aaaa" };

let data = null,
  etag = "",
  initialized = false,
  baseLayer = null,
  markers = new Map(),
  edges = [],
  allEdges = [],
  reportPage = 0,
  sortColumn = 0,
  sortAscending = true,
  loading = false;
const visibleBands = new Set(Object.keys(bands)),
  visibleTypes = new Set(Object.keys(types));
const map = L.map("map", { preferCanvas: true }).setView([34.35, -119.6], 9),
  stationLayer = L.layerGroup().addTo(map),
  linkLayer = L.layerGroup().addTo(map);
  map.zoomControl.setPosition("bottomright");

//Cursor Coordinates
const cursorCoordinates = $("cursor-coordinates");
  
  map.on("mousemove", (e) => {
    const position = e.latlng.wrap();
    
    cursorCoordinates.textContent =
    "Pointer Coordinates: Lat: " + position.lat.toFixed(5) +
    " · Lon: " + position.lng.toFixed(5);
  });
  
  map.getContainer().addEventListener("mouseleave", () => {
    cursorCoordinates.textContent = "Pointer Coordinates: Lat: — · Lon: —";
  });

  const nodePane = map.createPane("nodeMarkers");
  nodePane.style.zIndex = "450";

//
// L.control.scale().addTo(map);
//
let distanceScale = null;
function el(tag, text, className) {
  const e = document.createElement(tag);
  if (text !== undefined && text !== null) e.textContent = String(text);
  if (className) e.className = className;
  return e;
}
function format(v, unit = "") {
  const n = M.number(v);
  return n === null
    ? "—"
    : n.toLocaleString(undefined, { maximumFractionDigits: 2 }) +
        (unit ? " " + unit : "");
}
function distanceUnit() {
  return data?.mapInfo?.kilometers === true ? "km" : "mi";
}

function linkDistance(link) {
  return M.number(
    distanceUnit() === "km"
    ? link.distanceKM
    : link.distanceMiles
  );
}
function utc(v) {
  if (!v) return "Unknown";
  const d = new Date(
    String(v).includes("T") ? v : String(v).replace(" ", "T") + "Z",
  );
  return Number.isFinite(d.getTime()) ? d.toLocaleString() : String(v);
}
function located(n) {
  const lat = M.number(n.lat),
    lon = M.number(n.lon);
  return (
    lat !== null &&
    lon !== null &&
    !(lat === 0 && lon === 0) &&
    Math.abs(lat) <= 90 &&
    Math.abs(lon) <= 180
  );
}
function linkType(l, t) {
  const type = String(t?.type || l.linkType || "").toUpperCase();
  if (type === "RF" || type === "DTD" || type === "XLINK") return type;
  if (["TUN", "WIREGUARD", "TUNNEL"].includes(type)) return "TUN";
  return "OTHER";
}

//Remove null values in the node popups
/*
function addRows(parent, rows) {
  const dl = el("dl", null, "metrics");
  for (const [k, v] of rows) {
    dl.append(
      el("dt", k),
      el("dd", v === null || v === undefined || v === "" ? "—" : v),
    );
  }
  parent.append(dl);
}
*/
function addRows(parent, rows) {
  const dl = el("dl", null, "metrics");
  
  for (const [k, v] of rows) {
    if (
      v === null ||
      v === undefined ||
      v === "" ||
      v === "—" ||
      v === "-"
    ) {
      continue;
    }
    
    dl.append(el("dt", k), el("dd", v));
  }
  
  if (dl.children.length) parent.append(dl);
}

function nodeURL(n) {
  const host = String(n.node || "");
  if (/^[A-Za-z0-9-]+$/.test(host))
    return M.safeURL("http://" + host + ".local.mesh/");
  return null;
}
function serviceList(n) {
  const content = el("div");
  if (!n.services?.length) {
    content.append(el("p", "No published services."));
    return content;
  }
  for (const s of n.services) {
    const p = el("p"),
      url = M.safeURL(s.link);
    if (url) {
      const a = el("a", s.name || url);
      a.href = url;
      a.target = "_blank";
      a.rel = "noopener noreferrer";
      p.append(a);
    } else p.append(el("span", s.name || "Unnamed service"));
    content.append(p);
  }
  return content;
}
function telemetryDetails(parent, node, link, tracker, id) {
  const t = tracker || {};
  parent.append(
    el(
      "h3",
      t.device || link.interface || link.olsrInterface || "Reported link",
    ),
  );
  const now = M.number(node.lqm_clock?.now),
    seen = M.number(t.lastseen),
    reverseSeen = M.number(t.rev_lastseen);
  addRows(parent, [
    ["Tracker", id || "Legacy link info"],
    ["Link type", t.type || link.linkType],
    [
      "Routable",
      typeof t.routable === "boolean" ? (t.routable ? "Yes" : "No") : "—",
    ],
    ["IPv6 link address", t.ipv6ll],
    ["TX radio rate", format(M.value("tx", link, t), "Mbps")],
    ["RX radio rate", format(M.value("rx", link, t), "Mbps")],
    ["Average TX rate", format(t.avg_tx_bitrate, "Mbps")],
    ["Average RX rate", format(t.avg_rx_bitrate, "Mbps")],
    ["Expected throughput (reported)", format(link.expected_throughput)],
    ["Signal", format(M.first(t.signal, link.signal), "dBm")],
    ["SNR", format(M.value("snr", link, t), "dB")],
    ["Reverse SNR", format(t.rev_snr, "dB")],
    ["Noise", format(M.first(t.noise, link.noise), "dBm")],
    ["Hello RX quality", format(t.lq, "%")],
    ["Average Hello RX quality", format(t.avg_lq, "%")],
    ["Reverse Hello RX quality", format(t.rev_lq, "%")],
    ["TX delivery quality", format(t.tx_quality, "%")],
    ["Composite quality", format(t.quality, "%")],
    ["Reverse composite quality", format(t.rev_quality, "%")],
    ["Ping success quality", format(t.ping_quality, "%")],
    ["Reverse ping quality", format(t.rev_ping_quality, "%")],
    [
      "Ping time",
      format(
        M.number(t.ping_success_time) === null
          ? null
          : t.ping_success_time * 1000,
        "ms",
      ),
    ],
    [
      "Reverse ping time",
      format(
        M.number(t.rev_ping_success_time) === null
          ? null
          : t.rev_ping_success_time * 1000,
        "ms",
      ),
    ],
    ["Babel RTT", format(t.rtt, "ms")],
    ["Babel RX cost", format(t.rxcost)],
    ["Babel TX cost", format(t.txcost)],
    ["Route metric summary", format(M.first(t.metric, t.babel_metric))],
    [
      "Routes via neighbor",
      format(M.first(t.route_count, t.babel_route_count)),
    ],
    ["TX packet counter", format(t.tx_packets)],
    ["TX retry counter", format(t.tx_retries)],
    ["TX failure counter", format(t.tx_fail)],
    ["Average TX packets (reported)", format(t.avg_tx_packets)],
    ["Average TX retries (reported)", format(t.avg_tx_retries)],
    ["Average TX failures (reported)", format(t.avg_tx_fail)],
    ["OLSR link cost", format(link.linkCost)],
    ["OLSR path cost", format(link.pathCost)],
    ["OLSR receive quality", format(M.value("olsrQuality", link, t), "%")],
    [
      "OLSR neighbor quality",
      format(
        M.number(link.neighborLinkQuality) === null
          ? null
          : link.neighborLinkQuality * 100,
        "%",
      ),
    ],
    ["Distance", format(linkDistance(link), distanceUnit())],
    ["Bearing", format(link.bearing, "°")],
    [
      "Local sample age at poll",
      format(
        now === null || seen === null ? null : Math.max(0, now - seen),
        "s",
      ),
    ],
    [
      "Reverse sample age at poll",
      format(
        now === null || reverseSeen === null
          ? null
          : Math.max(0, now - reverseSeen),
        "s",
      ),
    ],
    ["Hello interval (reported)", format(t.hello_interval)],
    ["Update interval (reported)", format(t.update_interval)],
  ]);
  parent.append(
    el(
      "p",
      "Radio rates are not measured throughput. Route summaries are not direct-link ETX. Cached reverse measurements may be older.",
      "node-subtitle",
    ),
  );
}
function nodeDetails(n) {
  const root = el("div");
  root.append(
    el("h2", n.node),
    el("p", n.wlan_ip + " · Last seen " + utc(n.last_seen), "node-subtitle"),
  );
  const tabs = el("div", null, "tabs");
  tabs.setAttribute("role", "tablist");
  const area = el("div");
  const panels = {
    Node: () => {
      const p = el("div");
      const antenna = String(n.antDesc ?? "").trim();
      const hasAntenna = antenna !== "" && antenna.toLowerCase() !== "not available";
      if (n.description) p.append(el("p", n.description));
      addRows(p, [
        ["Model", n.model],
        ["Firmware", n.firmware_version],
        ["API", n.api_version],
        ["Location", located(n) ? n.lat + ", " + n.lon : "Not provided"],
        ["Grid", n.grid_square],
        ...(hasAntenna
          ? [
              ["SSID", n.ssid],
              [
                "Channel / width",
                  String(n.channel ?? "—") + " / " + String(n.chanbw ?? "—") + " MHz",
              ],
              ["Frequency", format(n.freq, "MHz")],
              ["Antenna", n.antDesc],
              [
                "Gain / beam",
                  format(n.antGain, "dBi") + " / " + format(n.antBeam, "°"),
              ],
            ]
          : []),
        ["Uptime", n.uptime],
        ["Hops (when reported)", n.hopsAway ?? "Unknown"],
      ]);
      const url = nodeURL(n);
      if (url) {
        const a = el("a", "Open node on the mesh ↗");
        a.href = url;
        a.target = "_blank";
        a.rel = "noopener noreferrer";
        p.append(a);
      }
      return p;
    },
    Services: () => serviceList(n),
    Links: () => {
      const p = el("div"),
        links = Object.entries(n.link_info || {});
      if (!links.length) p.append(el("p", "No reported neighbors."));
      for (const [ip, link] of links.slice(0, 100)) {
        p.append(
          el(
            "p",
            (link.hostname || ip) + " · " + (link.linkType || "Unknown"),
            "peer-heading",
          ),
        );
        const trackers = Object.entries(link.lqm_trackers || {});
        if (!trackers.length) telemetryDetails(p, n, link, null, "");
        else
          for (const [id, t] of trackers) telemetryDetails(p, n, link, t, id);
      }
      if (links.length > 100)
        p.append(
          el(
            "p",
            "Showing the first 100 neighbors. The link report includes all reported links.",
          ),
        );
      return p;
    },
  };
  for (const [name, build] of Object.entries(panels)) {
    const button = el("button", name);
    button.setAttribute("role", "tab");
    button.setAttribute("aria-selected", "false");
    button.addEventListener("click", () => {
      for (const b of tabs.children)
        b.setAttribute("aria-selected", String(b === button));
      area.replaceChildren(build());
    });
    tabs.append(button);
  }
  root.append(tabs, area);
  tabs.firstChild.click();
  const neighbors = (data?.nodes || []).filter(
    (other) =>
      other.wlan_ip !== n.wlan_ip &&
      located(other) &&
      located(n) &&
      Math.abs(Number(other.lat) - Number(n.lat)) < 0.00001 &&
      Math.abs(Number(other.lon) - Number(n.lon)) < 0.00001,
  );
  if (neighbors.length) {
    const group = el("details");
    group.append(
      el("summary", "Also at this location (" + neighbors.length + ")"),
    );
    for (const peer of neighbors) {
      const button = el("button", peer.node, "detail-link");
      button.addEventListener("click", () => openNode(peer));
      const line = el("p");
      line.append(button);
      group.append(line);
    }
    root.append(group);
  }
  return root;
}
function openNode(n) {
  if (located(n)) {
    const marker = markers.get(n.wlan_ip);
    if (marker) {
      map.setView([n.lat, n.lon], Math.max(map.getZoom(), 12));
      marker.openPopup();
      return;
    }
  }
  $("detail-content").replaceChildren(nodeDetails(n));
  $("detail-dialog").showModal();
}
function edgePopup(e) {
  const root = el("div");
  root.append(
    el("h2", e.from.node + " → " + (e.to?.node || e.link.hostname || e.ip)),
    el("p", "Source polled " + utc(e.from.last_seen), "node-subtitle"),
  );
  telemetryDetails(root, e.from, e.link, e.tracker, e.id);
  return root;
}
function buildEdges() {
  allEdges = [];
  const byIP = new Map(data.nodes.map((n) => [n.wlan_ip, n])),
    byName = new Map(data.nodes.map((n) => [M.canonicalName(n.node), n]));
  for (const from of data.nodes) {
    for (const [ip, link] of Object.entries(from.link_info || {})) {
      const to = byIP.get(ip) || byName.get(M.canonicalName(link.hostname));
      const trackers = Object.entries(link.lqm_trackers || {});
      for (const [id, tracker] of trackers.length ? trackers : [["", null]]) {
        allEdges.push({
          from,
          to,
          ip,
          link,
          tracker,
          id,
          type: linkType(link, tracker),
        });
      }
    }
  }
}
function renderMap() {
  stationLayer.clearLayers();
  linkLayer.clearLayers();
  markers.clear();
  edges = [];
  const kind = $("metric").value,
    cap = Number($("scale").value);
  for (const n of data.nodes) {
    if (!visibleBands.has(n.band) || !located(n)) continue;
    const marker = L.circleMarker([n.lat, n.lon], {
      pane: "nodeMarkers",
      radius: n.band === "supernode" ? 8 : 5,
      color: n.node === data.mapInfo.localnode ? "#142b35" : "#ffffff",
      weight: n.node === data.mapInfo.localnode ? 3 : 1.5,
      fillColor: (bands[n.band] || bands.other)[1],
      fillOpacity: 0.95,
    });
    marker.bindTooltip(() => el("span", n.node));
    marker.bindPopup(() => nodeDetails(n), { maxWidth: 410, minWidth: 300 });
    marker.addTo(stationLayer);
    markers.set(n.wlan_ip, marker);
  }
  const offsets = new Map();
  for (const e of allEdges) {
    if (
      !visibleBands.has(e.from.band) ||
      !visibleTypes.has(e.type) ||
      !located(e.from)
    )
      continue;
    let target =
      e.to && located(e.to)
        ? [e.to.lat, e.to.lon]
        : [M.number(e.link.linkLat), M.number(e.link.linkLon)];
    if (
      target.some((v) => v === null) ||
      !Number.isFinite(Number(target[0])) ||
      !Number.isFinite(Number(target[1]))
    )
      continue;
    const key = e.from.wlan_ip + ">" + e.ip,
      index = offsets.get(key) || 0;
    offsets.set(key, index + 1);
    const value = M.value(kind, e.link, e.tracker);
    const line = L.polyline([[e.from.lat, e.from.lon], target], {
/*    color:
        e.type === "TUN" ? "#3366cc" :
        e.type === "DTD" ? "#9933cc" :
        e.type === "XLINK" ? "#00aaaa" :
        M.color(kind, value, cap),
*/
      color: linkColors[e.type] || M.color(kind, value, cap),
      weight: 2.5,
      opacity:
        e.type === "TUN" ? 0.3 :
        e.type === "DTD" ? 0.3 :
        e.type === "XLINK" ? 0.3 :
        0.8,
      dashArray:
        e.type === "TUN" ? "7 10" :
        e.type === "DTD" ? "2 6" :
        e.type === "XLINK" ? "10 4 2 4" :
        undefined,
      offset: 3 + index * 3,
    });
    line.bindTooltip(() =>
      el(
        "span",
        e.from.node +
          " → " +
          (e.to?.node || e.link.hostname || e.ip) +
          " · " +
          format(value, M.specs[kind][1]),
      ),
    );
    line.bindPopup(() => edgePopup(e), { maxWidth: 430, minWidth: 320 });
    line.addTo(linkLayer);
    edges.push(e);
  }
  updateLegend();
  const mapped = data.nodes.filter(located).length;
  $("summary").textContent =
    data.nodes.length.toLocaleString() +
    " nodes · " +
    mapped.toLocaleString() +
    " located · " +
    edges.length.toLocaleString() +
    " directed links shown";
}
function updateLegend() {
  const kind = $("metric").value,
    s = M.specs[kind],
    speed = kind === "tx" || kind === "rx";
  $("legend-title").textContent = s[0] + " · " + s[1];
  $("scale-label").hidden = !speed;
  $("range-start").textContent = s[4] ? s[3] + "+" : s[2];
  $("range-end").textContent = s[4]
    ? s[2]
    : speed
      ? $("scale").value + "+"
      : s[3] + "+";
  $("metric-note").textContent = speed
    ? "Reported radio rate for RF links; not measured throughput."
    : kind === "route"
      ? "Route summary; not direct-link ETX."
      : kind === "rxcost" || kind === "txcost"
        ? "Raw Babel cost; lower is better."
        : kind === "reversequality"
          ? "Cached reverse composite quality; not Hello reception."
          : "Reported metric; missing values stay gray.";
}
function fitNetwork() {
  const points =
    data?.nodes
      .filter((n) => visibleBands.has(n.band) && located(n))
      .map((n) => [n.lat, n.lon]) || [];
  if (points.length) map.fitBounds(points, { padding: [70, 70], maxZoom: 14 });
}
function buildTiles(info) {
  const select = $("tiles");
  select.replaceChildren();
  const providers = Object.entries(info.mapTileServers || {});
  let selected =
    providers.find(
      ([k, v]) =>
        location.hostname.endsWith(".local.mesh") &&
        String(v).includes(".local.mesh"),
    ) ||
    providers.find(([k]) => k.endsWith("-default")) ||
    providers[0];
  if (!selected)
    selected = [
      "Street-default",
      "https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png",
    ];
  for (const [k, v] of providers.length ? providers : [selected]) {
    const option = el("option", k.replace(/-default$/, ""));
    option.value = String(v);
    select.append(option);
  }
  select.value = String(selected[1]);
  chooseTiles();
}
function chooseTiles() {
  let url = $("tiles").value;
  if (baseLayer) {
    map.removeLayer(baseLayer);
    baseLayer = null;
  }
  if (url.startsWith("//")) url = location.protocol + url;
  url = url.replace(
    /^https?:\/\/(?:\{s\}\.|[abc]\.)?tile\.openstreetmap\.org\//,
    "https://tile.openstreetmap.org/",
  );
  if (!/^https?:\/\//i.test(url) && !url.startsWith("/")) return;
  if (location.protocol === "https:" && url.startsWith("http:"))
    url = "https:" + url.slice(5);
  let attribution =
    "Map tiles: " +
    $("tiles").selectedOptions[0].textContent.replace(/[<>&"']/g, "");
  if (url.includes("openstreetmap.org"))
    attribution =
      '© <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors';
  else if (url.includes("opentopomap.org"))
    attribution =
      '© OpenStreetMap contributors, SRTM · <a href="https://opentopomap.org">OpenTopoMap</a>';
  else if (url.includes("nationalmap.gov"))
    attribution = "USGS The National Map";
  baseLayer = L.tileLayer(url, { maxZoom: 19, attribution }).addTo(map);
  baseLayer.bringToBack();
}
function restoreView(info, useURL = true) {
  const parts = location.hash.replace(/^#/, "").split("/").map(Number);
  if (
    useURL &&
    parts.length === 3 &&
    parts.every(Number.isFinite) &&
    parts[0] >= 0 &&
    parts[0] <= 19 &&
    Math.abs(parts[1]) <= 90 &&
    Math.abs(parts[2]) <= 180
  ) {
    map.setView([parts[1], parts[2]], parts[0]);
  } else {
    const coords = info.mapCenterCoords;
    if (
      Array.isArray(coords) &&
      coords.length === 2 &&
      coords.every((v) => M.number(v) !== null)
    ) {
      map.setView(coords, M.number(info.mapInitialZoom) ?? 9);
    }
  }
}
async function refresh() {
  if (loading) return;
  loading = true;
  try {
    const response = await fetch("/api/map", {
      headers: etag ? { "If-None-Match": etag } : {},
      cache: "no-cache",
    });
    if (response.status === 304) return;
    if (!response.ok) {
      throw Error(
        response.status === 503
          ? "Waiting for the first completed poll. The map will retry automatically."
          : "Data unavailable; retaining the last loaded map.",
      );
    }
    const next = await response.json();
    if (!Array.isArray(next.nodes)) throw Error("Unexpected map data format.");
    data = next;
    if (distanceScale) distanceScale.remove();
    
    const metric = distanceUnit() === "km";
    
    distanceScale = L.control.scale({
      position: "bottomright",
      metric: metric,
      imperial: !metric,
    }).addTo(map);
    
    map.zoomControl.addTo(map);
    
    etag = response.headers.get("ETag") || "";
    document.title = data.mapInfo.title || "KG6WXC MeshMap";
    $("title").textContent = document.title;
    $("credit").textContent =
      (data.mapInfo.attribution || "KG6WXC MeshMap") + " · " + data.mapInfo.version + " · Go";
    $("timestamp").textContent =
      "Last completed poll: " +
      utc(data.pollingInfo.lastPollingRun) +
      " · " +
      format(data.pollingInfo.pollingTimeSec / 60, "min");
    if (!initialized) {
      buildTiles(data.mapInfo);
      restoreView(data.mapInfo);
      initialized = true;
    }
    buildEdges();
    renderMap();
    const options = document.createDocumentFragment();
    for (const n of data.nodes) {
      const option = el("option");
      option.value = n.node;
      options.append(option);
    }
    $("node-options").replaceChildren(options);
    $("notice").hidden = data.nodes.length > 0;
    if (!data.nodes.length)
      $("notice").textContent =
        "The snapshot has no successfully polled nodes yet.";
    if ($("report-dialog").open) renderReport();
  } catch (error) {
    $("notice").textContent = error.message;
    $("notice").hidden = false;
  } finally {
    loading = false;
  }
}
function find() {
  if (!data) return;
  const query = $("search").value.toLowerCase().trim();
  if (!query) return;
  const n =
    data.nodes.find(
      (n) => n.node.toLowerCase() === query || n.wlan_ip === query,
    ) || data.nodes.find((n) => n.node.toLowerCase().includes(query));
  if (n) openNode(n);
  else {
    $("notice").textContent = "No node matches “" + query + "”.";
    $("notice").hidden = false;
  }
}
function reportDefinition() {
  const kind = $("report-kind").value;
  if (kind === "links")
    return {
      columns: [
        "From",
        "To",
        "Type",
        "Interface",
        "Distance (" + distanceUnit() + ")",
        "TX Mbps",
        "RX Mbps",
        "SNR dB",
        "Hello RX %",
        "TX quality %",
        "Babel RX cost",
        "Babel TX cost",
        "RTT ms",
        "Route metric",
      ],
      rows: allEdges.map((e) => ({
        node: e.from,
        edge: e,
        values: [
          e.from.node,
          e.to?.node || e.link.hostname || e.ip,
          e.tracker?.type || e.link.linkType,
          e.tracker?.device || e.link.interface || e.link.olsrInterface,
          linkDistance(e.link),
          M.value("tx", e.link, e.tracker),
          M.value("rx", e.link, e.tracker),
          M.value("snr", e.link, e.tracker),
          M.value("hello", e.link, e.tracker),
          M.value("txquality", e.link, e.tracker),
          M.value("rxcost", e.link, e.tracker),
          M.value("txcost", e.link, e.tracker),
          M.value("rtt", e.link, e.tracker),
          M.value("route", e.link, e.tracker),
        ],
      })),
    };
  return {
    columns: [
      "Node",
      "Uptime",
      "IP",
      "Band",
      "Model",
      "Firmware",
      "Last seen (UTC)",
      "Location",
    ],
    rows: (data?.nodes || [])
      .filter((n) => kind !== "missing" || !located(n))
      .map((n) => ({
        node: n,
        values: [
          n.node,
          n.uptime,
          n.wlan_ip,
          (bands[n.band] || bands.other)[0],
          n.model,
          n.firmware_version,
          n.last_seen,
          located(n) ? n.lat + ", " + n.lon : "Not provided",
        ],
      })),
  };
}
function filteredReport() {
  const def = reportDefinition(),
    query = $("report-filter").value.toLowerCase();
  def.rows = def.rows.filter((r) =>
    r.values.some((v) =>
      String(v ?? "")
        .toLowerCase()
        .includes(query),
    ),
  );
  const column = Math.min(sortColumn, def.columns.length - 1);
  def.rows.sort((a, b) => {
    const x = a.values[column],
      y = b.values[column];
    return (
      (typeof x === "number" && typeof y === "number"
        ? x - y
        : String(x ?? "").localeCompare(String(y ?? ""))) *
      (sortAscending ? 1 : -1)
    );
  });
  return def;
}
function renderReport() {
  const def = filteredReport(),
    table = $("report-table"),
    head = el("thead"),
    tr = el("tr");
  def.columns.forEach((name, i) => {
    const th = el("th"),
      button = el(
        "button",
        name + (sortColumn === i ? (sortAscending ? " ↑" : " ↓") : ""),
      );
    button.addEventListener("click", () => {
      sortAscending = sortColumn === i ? !sortAscending : true;
      sortColumn = i;
      renderReport();
    });
    th.append(button);
    tr.append(th);
  });
  head.append(tr);
  const body = el("tbody");
  const pages = Math.max(1, Math.ceil(def.rows.length / 100));
  reportPage = Math.min(reportPage, pages - 1);
  for (const row of def.rows.slice(reportPage * 100, (reportPage + 1) * 100)) {
    const tr = el("tr");
    row.values.forEach((value, i) => {
      const td = el("td");
      if (i === 0) {
        const button = el("button", value);
        button.addEventListener("click", () => {
          $("report-dialog").close();
          if (row.edge && located(row.node)) {
            map.setView(
              [row.node.lat, row.node.lon],
              Math.max(map.getZoom(), 12),
            );
            L.popup({ maxWidth: 430 })
              .setLatLng([row.node.lat, row.node.lon])
              .setContent(edgePopup(row.edge))
              .openOn(map);
          } else openNode(row.node);
        });
        td.append(button);
      } else
        td.textContent =
          value === null || value === undefined
            ? "—"
            : typeof value === "number"
              ? format(value)
              : String(value);
      tr.append(td);
    });
    body.append(tr);
  }
  table.replaceChildren(head, body);
  $("report-summary").textContent =
    def.rows.length.toLocaleString() +
    " matching rows · click a name to inspect";
  $("page-label").textContent = "Page " + (reportPage + 1) + " of " + pages;
  $("previous").disabled = reportPage === 0;
  $("next").disabled = reportPage >= pages - 1;
}
function downloadCSV() {
  const def = filteredReport();
  const quote = (v) => {
    let s = String(v ?? "");
    if (/^[=+\-@\t\r]/.test(s)) s = "'" + s;
    return '"' + s.replace(/"/g, '""') + '"';
  };
  const csv = [def.columns, ...def.rows.map((r) => r.values)]
    .map((row) => row.map(quote).join(","))
    .join("\r\n");
  const url = URL.createObjectURL(
    new Blob([csv], { type: "text/csv;charset=utf-8" }),
  );
  const a = el("a");
  a.href = url;
  a.download = "meshmap-" + $("report-kind").value + ".csv";
  a.click();
  URL.revokeObjectURL(url);
}
for (const [id, choices, set] of [
  ["bands", bands, visibleBands],
  [
    "types",
    Object.fromEntries(Object.entries(types).map(([k, v]) => [k, [v, null]])),
    visibleTypes,
  ],
]) {
  for (const [key, [name, color]] of Object.entries(choices)) {
    const label = el("label"),
      input = el("input");
    input.type = "checkbox";
    input.checked = true;
    input.addEventListener("change", () => {
      input.checked ? set.add(key) : set.delete(key);
      if (data) renderMap();
    });
    label.append(input);
    if (color) {
      const dot = el("span", null, "dot");
      dot.style.setProperty("--band", color);
      label.append(dot);
    }
//    label.append(el("span", name));
    const text = el("span", name);
    if (id === "types" && linkColors[key]) {
      text.style.color = linkColors[key];
    }
    label.append(text);
    $(id).append(label);
  }
}
$("metric").addEventListener("change", () =>
  data ? renderMap() : updateLegend(),
);
$("scale").addEventListener("change", () => data && renderMap());
$("tiles").addEventListener("change", chooseTiles);
$("fit").addEventListener("click", fitNetwork);
$("recenter").addEventListener("click", () => {
  if (data?.mapInfo) restoreView(data.mapInfo, false);
});
$("origin").addEventListener("click", () => {
  const n = data?.nodes.find((n) => n.node === data.mapInfo.localnode);
  if (n) openNode(n);
});
$("find").addEventListener("click", find);
$("search").addEventListener("keydown", (event) => {
  if (event.key === "Enter") find();
});
$("reports").addEventListener("click", () => {
  renderReport();
  $("report-dialog").showModal();
});
$("close-report").addEventListener("click", () => $("report-dialog").close());
$("close-detail").addEventListener("click", () => $("detail-dialog").close());
$("report-kind").addEventListener("change", () => {
  reportPage = 0;
  sortColumn = 0;
  renderReport();
});
$("report-filter").addEventListener("input", () => {
  reportPage = 0;
  renderReport();
});
$("previous").addEventListener("click", () => {
  reportPage--;
  renderReport();
});
$("next").addEventListener("click", () => {
  reportPage++;
  renderReport();
});
$("download").addEventListener("click", downloadCSV);
//JS part of the "Refresh Data" button
//$("refresh").addEventListener("click", refresh);
map.on("moveend", () => {
  if (initialized) {
    const p = map.getCenter();
    history.replaceState(
      null,
      "",
      "#" + map.getZoom() + "/" + p.lat.toFixed(5) + "/" + p.lng.toFixed(5),
    );
  }
});
window.addEventListener("hashchange", () => {
  if (initialized) {
    restoreView(data.mapInfo);
  }
});
if (matchMedia("(max-width:720px)").matches)
  document.querySelector(".controls details").open = false;

// Polling Indicator
let checkingPollStatus = false;

async function refreshPollStatus() {
  if (checkingPollStatus) return;
  checkingPollStatus = true;
  
  const indicator = $("poll-status");
  const text = $("poll-status-text");
  
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 4000);
  
  try {
    const response = await fetch("/api/status", {
      cache: "no-store",
//      signal: AbortSignal.timeout(4000),
      signal: controller.signal,
    });
    
    if (!response.ok) throw new Error("Status unavailable");
    
    const status = await response.json();
    const labels = {
      polling: "Currently polling",
      waiting: "Waiting for next poll",
      "serve-only": "No Polling · Serve-only mode",
    };
    
    indicator.dataset.state = labels[status.state]
    ? status.state
    : "unknown";
    
    text.textContent = labels[status.state] || "Status unavailable";
    
    if (
      status.state === "waiting" &&
      Number.isFinite(status.next_poll_in_seconds)
    ) {
      const totalMinutes = Math.ceil(status.next_poll_in_seconds / 60);
      
      if (totalMinutes === 0) {
        text.textContent = "Next poll due";
      } else {
        const hours = Math.floor(totalMinutes / 60);
        const minutes = totalMinutes % 60;
        const remaining = [];
        
        if (hours > 0) remaining.push(`${hours}h`);
        if (minutes > 0) remaining.push(`${minutes}m`);
        
        text.textContent = `Waiting for next poll in: ${remaining.join(" ")}`;
      }
    }
    
    if (
      status.state === "polling" &&
      status.phase === "node polling" &&
      status.targets > 0
    ) {
      text.textContent +=
      " · " + status.completed + "/" + status.targets;
    }
    
    indicator.title =
    status.state === "polling"
    ? "Phase: " + (status.phase || "starting") +
    " · Elapsed: " + (status.elapsed_seconds || 0) + " seconds"
    : "";
  } catch {
    indicator.dataset.state = "unknown";
    indicator.title = "Cannot obtain live polling status.";
    text.textContent = "Status unavailable";
  } finally {
    clearTimeout(timeout);
    checkingPollStatus = false;
  }
}

refreshPollStatus();
setInterval(refreshPollStatus, 5000);

refresh();
setInterval(refresh, 30000);
