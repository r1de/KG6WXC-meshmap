/* SPDX-License-Identifier: GPL-3.0-or-later */
(function (root) {
  "use strict";
  const specs = {
    tx: ["TX radio speed", "Mbps", 0, 100, false],
    rx: ["RX radio speed", "Mbps", 0, 100, false],
    snr: ["Signal to noise", "dB", 0, 40, false],
    hello: ["Hello receive quality", "%", 0, 100, false],
    txquality: ["TX delivery quality", "%", 0, 100, false],
    reversequality: ["Reverse composite quality", "%", 0, 100, false],
    rtt: ["Babel RTT", "ms", 0, 100, true],
    rxcost: ["Babel RX cost", "raw", 256, 1024, true],
    txcost: ["Babel TX cost", "raw", 256, 1024, true],
    route: ["Route metric summary", "raw", 256, 2048, true],
    olsrCost: ["OLSR link cost", "raw", 1, 4, true],
    olsrQuality: ["OLSR receive quality", "%", 0, 100, false],
  };
  function number(v) {
    if (v === null || v === undefined || v === "" || typeof v === "boolean")
      return null;
    const n = Number(v);
    return Number.isFinite(n) ? n : null;
  }
  function first(...values) {
    for (const v of values) {
      const n = number(v);
      if (n !== null) return n;
    }
    return null;
  }
  function value(kind, link, tracker) {
    const t = tracker || {},
      l = link || {};
    const hasAmbiguousLegacyRate =
      tracker &&
      Object.keys(l.lqm_trackers || {}).length > 1 &&
      String(l.interface || l.olsrInterface || "") !== String(t.device || "");
    const legacyTx = hasAmbiguousLegacyRate ? null : l.tx_rate,
      legacyRx = hasAmbiguousLegacyRate ? null : l.rx_rate;
    switch (kind) {
      case "tx":
        return first(t.tx_bitrate, legacyTx);
      case "rx":
        return first(t.rx_bitrate, legacyRx);
      case "snr": {
        const v = number(t.snr);
        if (v !== null) return v;
        const s = first(t.signal, l.signal),
          n = first(t.noise, l.noise);
        return s !== null && n !== null ? s - n : null;
      }
      case "hello":
        return number(t.lq);
      case "txquality":
        return number(t.tx_quality);
      case "reversequality":
        return number(t.rev_quality);
      case "rtt":
        return number(t.rtt);
      case "rxcost":
        return number(t.rxcost);
      case "txcost":
        return number(t.txcost);
      case "route":
        return first(t.metric, t.babel_metric);
      case "olsrCost":
        return number(l.linkCost);
      case "olsrQuality": {
        const n = number(l.linkQuality);
        return n === null ? null : n * 100;
      }
      default:
        return null;
    }
  }
  function color(kind, value, cap) {
    if (number(value) === null) return "#89969e";
    const spec = specs[kind],
      max = kind === "tx" || kind === "rx" ? number(cap) || 100 : spec[3];
    let x = Math.max(0, Math.min(1, (value - spec[2]) / (max - spec[2])));
    if (spec[4]) x = 1 - x;
    const stops = [
        [214, 56, 53],
        [229, 166, 44],
        [98, 168, 62],
        [24, 132, 75],
      ],
      i = Math.min(2, Math.floor(x * 3)),
      f = x * 3 - i;
    return (
      "#" +
      stops[i]
        .map((c, j) =>
          Math.round(c + (stops[i + 1][j] - c) * f)
            .toString(16)
            .padStart(2, "0"),
        )
        .join("")
    );
  }
  function safeURL(v) {
    if (typeof v !== "string") return null;
    try {
      const u = new URL(v);
      return u.protocol === "http:" || u.protocol === "https:" ? u.href : null;
    } catch {
      return null;
    }
  }
  function canonicalName(v) {
    return String(v || "")
      .toLowerCase()
      .replace(/\.local\.mesh$/, "");
  }
  const api = { specs, number, first, value, color, safeURL, canonicalName };
  root.MeshMetrics = api;
  if (typeof module !== "undefined" && module.exports) module.exports = api;
})(typeof globalThis !== "undefined" ? globalThis : this);
