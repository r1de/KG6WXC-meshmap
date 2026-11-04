// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Store struct {
	DB     *sql.DB
	Table  string
	Driver string
}

func openStore(ctx context.Context, c Config) (*Store, error) {
	driver := c.Driver
	if driver == "" {
		driver = "mysql"
	}
	if driver == "sqlite" {
		if e := os.MkdirAll(filepath.Dir(c.DSN), 0700); e != nil {
			return nil, e
		}
	}
	db, e := sql.Open(driver, c.DSN)
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(5 * time.Minute)
	if e = db.PingContext(ctx); e != nil {
		db.Close()
		return nil, fmt.Errorf("database unavailable: %w", e)
	}
	s := &Store{DB: db, Table: c.Table, Driver: driver}
	if driver == "sqlite" {
		db.SetMaxOpenConns(1)
		if _, e = db.ExecContext(ctx, "PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000; PRAGMA foreign_keys=ON;"); e != nil {
			db.Close()
			return nil, e
		}
		if e = s.initialize(ctx); e != nil {
			db.Close()
			return nil, e
		}
	}
	if e = s.migrateTelemetry(ctx); e != nil {
		db.Close()
		return nil, fmt.Errorf("initialize LQM storage: %w", e)
	}
	return s, nil
}
func (s *Store) load(ctx context.Context) ([]Row, error) {
	rows, e := s.DB.QueryContext(ctx, "SELECT * FROM `"+s.Table+"` ORDER BY wlan_ip")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	cols, e := rows.Columns()
	if e != nil {
		return nil, e
	}
	out := []Row{}
	for rows.Next() {
		values := make([]any, len(cols))
		ptr := make([]any, len(cols))
		for i := range values {
			ptr[i] = &values[i]
		}
		if e = rows.Scan(ptr...); e != nil {
			return nil, e
		}
		r := Row{}
		for i, k := range cols {
			v := values[i]
			if b, ok := v.([]byte); ok {
				v = string(b)
			}
			if v != nil {
				switch k {
				case "services", "loadavg", "link_info":
					if decoded, e := unphp(str(v)); e == nil {
						v = decoded
					}
				case "lqm":
					var decoded any
					if err := json.Unmarshal([]byte(str(v)), &decoded); err != nil {
						return nil, fmt.Errorf("decode LQM for %s: %w", r["wlan_ip"], err)
					}
					v = decoded
				case "lat", "lon":
					v = num(v)
				}
			}
			r[k] = v
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
func (s *Store) upsert(ctx context.Context, tx *sql.Tx, table string, r Row, seen bool) error {
	keys := []string{}
	for k := range r {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	columns, marks, updates := []string{}, []string{}, []string{}
	args := []any{}
	for _, k := range keys {
		columns = append(columns, "`"+k+"`")
		marks = append(marks, "?")
		if k != "wlan_ip" && k != "id" {
			updates = append(updates, "`"+k+"`=VALUES(`"+k+"`)")
		}
		v := r[k]
		switch k {
		case "lqm":
			if v != nil {
				b, e := json.Marshal(v)
				if e != nil {
					return fmt.Errorf("encode LQM: %w", e)
				}
				v = string(b)
			}
		case "loadavg", "services", "link_info":
			switch v.(type) {
			case Row, map[string]any, []any:
				v = php(v)
			}
		}
		args = append(args, v)
	}
	if seen {
		columns = append(columns, "last_seen")
		marks = append(marks, "?")
		updates = append(updates, "last_seen=VALUES(last_seen)")
		args = append(args, time.Now().UTC().Format("2006-01-02 15:04:05"))
	}
	suffix := " ON DUPLICATE KEY UPDATE " + strings.Join(updates, ",")
	if s.Driver == "sqlite" {
		key := "id"
		if table == s.Table {
			key = "wlan_ip"
		}
		for i, u := range updates {
			updates[i] = regexpValues.ReplaceAllString(u, "excluded.$1")
		}
		suffix = " ON CONFLICT(`" + key + "`) DO UPDATE SET " + strings.Join(updates, ",")
	}
	_, e := tx.ExecContext(ctx, "INSERT INTO `"+table+"` ("+strings.Join(columns, ",")+") VALUES ("+strings.Join(marks, ",")+")"+suffix, args...)
	return e
}
func (s *Store) savePoll(ctx context.Context, c Config, targets []Target, results []Result) error {
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if c.b("expire_old_nodes") {
		if _, e = tx.ExecContext(ctx, "DELETE FROM `"+s.Table+"` WHERE last_seen <= ?", time.Now().UTC().AddDate(0, 0, -c.ExpireDays).Format("2006-01-02 15:04:05")); e != nil {
			return e
		}
	}
	if _, e = tx.ExecContext(ctx, "UPDATE `"+s.Table+"` SET link_info=?", php(Row{})); e != nil {
		return e
	}
	for _, t := range targets {
		if e = s.upsert(ctx, tx, s.Table, Row{"wlan_ip": t.IP, "hopsAway": t.Hops, "link_info": t.Links}, false); e != nil {
			return e
		}
	}
	for _, r := range results {
		if r.Err == nil {
			if e = s.upsert(ctx, tx, s.Table, r.Node, true); e != nil {
				return e
			}
		}
	}
	return tx.Commit()
}
func (s *Store) saveMeta(ctx context.Context, nodes []Row, stats, versions Row) error {
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	for _, n := range nodes {
		if _, e = tx.ExecContext(ctx, "UPDATE `"+s.Table+"` SET link_info=? WHERE wlan_ip=?", php(n["link_info"]), n["wlan_ip"]); e != nil {
			return e
		}
	}
	r := Row{"id": "POLLINFO"}
	for k, v := range stats {
		r[k] = v
	}
	if e = s.upsert(ctx, tx, "map_info", r, false); e != nil {
		return e
	}
	v := Row{"id": "AREDNINFO", "current_stable_version": versions["AREDN_STABLE_VERSION"], "current_nightly_version": versions["AREDN_NIGHTLY_VERSION"]}
	if e = s.upsert(ctx, tx, "aredn_info", v, false); e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Store) metadata(ctx context.Context) (Row, Row, error) {
	stats := Row{}
	var last sql.NullString
	var duration sql.NullFloat64
	var nt, garbage, hops, total, workers, noloc, links, nodes sql.NullInt64
	e := s.DB.QueryRowContext(ctx, "SELECT lastPollingRun,pollingTimeSec,nodeTotal,garbageReturned,highestHops,totalPolled,numParallelThreads,noLocation,mappableLinks,mappableNodes FROM map_info WHERE id='POLLINFO'").Scan(&last, &duration, &nt, &garbage, &hops, &total, &workers, &noloc, &links, &nodes)
	if e != nil && e != sql.ErrNoRows {
		return nil, nil, e
	}
	stats["lastPollingRun"] = last.String
	stats["pollingTimeSec"] = duration.Float64
	for k, v := range map[string]sql.NullInt64{"nodeTotal": nt, "garbageReturned": garbage, "highestHops": hops, "totalPolled": total, "numParallelThreads": workers, "noLocation": noloc, "mappableLinks": links, "mappableNodes": nodes} {
		stats[k] = v.Int64
	}
	versions := Row{"AREDN_STABLE_VERSION": "", "AREDN_NIGHTLY_VERSION": ""}
	var a, b sql.NullString
	e = s.DB.QueryRowContext(ctx, "SELECT current_stable_version,current_nightly_version FROM aredn_info WHERE id='AREDNINFO'").Scan(&a, &b)
	if e != nil && e != sql.ErrNoRows {
		return nil, nil, e
	}
	versions["AREDN_STABLE_VERSION"] = a.String
	versions["AREDN_NIGHTLY_VERSION"] = b.String
	return stats, versions, nil
}

var regexpValues = regexp.MustCompile("VALUES\\(([^)]+)\\)")

// SQLite initializes a fresh database without external services or schema imports.
func (s *Store) initialize(ctx context.Context) error {
	r := parseNode(Row{}, "")
	r["last_seen"] = ""
	r["location_fix"] = 0
	columns := []string{}
	for k := range r {
		typ := "TEXT"
		if k == "lat" || k == "lon" {
			typ = "REAL"
		}
		if k == "wlan_ip" {
			typ += " PRIMARY KEY"
		}
		columns = append(columns, "`"+k+"` "+typ)
	}
	sort.Strings(columns)
	_, e := s.DB.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS `"+s.Table+"` ("+strings.Join(columns, ",")+"); CREATE TABLE IF NOT EXISTS map_info (id TEXT PRIMARY KEY,lastPollingRun TEXT,pollingTimeSec REAL,nodeTotal INTEGER,garbageReturned INTEGER,highestHops INTEGER,totalPolled INTEGER,numParallelThreads INTEGER,noLocation INTEGER,mappableLinks INTEGER,mappableNodes INTEGER); CREATE TABLE IF NOT EXISTS aredn_info (id TEXT PRIMARY KEY,current_stable_version TEXT,current_nightly_version TEXT);")
	return e
}

// Additive migration works with both the PHP schema and earlier Go databases.
func (s *Store) migrateTelemetry(ctx context.Context) error {
	rows, e := s.DB.QueryContext(ctx, "SELECT * FROM `"+s.Table+"` LIMIT 0")
	if e != nil {
		return e
	}
	columns, e := rows.Columns()
	rows.Close()
	if e != nil {
		return e
	}
	found := false
	for _, c := range columns {
		if c == "lqm" {
			found = true
		}
	}
	if !found {
		typ := "TEXT"
		if s.Driver != "sqlite" {
			typ = "LONGTEXT"
		}
		if _, e = s.DB.ExecContext(ctx, "ALTER TABLE `"+s.Table+"` ADD COLUMN lqm "+typ+" NULL"); e != nil {
			return e
		}
	}
	if s.Driver != "sqlite" {
		var typ string
		if e = s.DB.QueryRowContext(ctx, "SELECT DATA_TYPE FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=? AND COLUMN_NAME='link_info'", s.Table).Scan(&typ); e != nil {
			return e
		}
		if strings.ToLower(typ) != "longtext" {
			if _, e = s.DB.ExecContext(ctx, "ALTER TABLE `"+s.Table+"` MODIFY COLUMN link_info LONGTEXT NULL"); e != nil {
				return e
			}
		}
	}
	return nil
}
