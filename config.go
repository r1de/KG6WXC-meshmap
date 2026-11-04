// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	_ "embed"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"gopkg.in/ini.v1"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

//go:embed settings/user-settings.ini-default
var defaultSettings []byte

type Config struct {
	Values                               map[string]string
	Arrays                               map[string]map[string]string
	Workers, ExpireDays                  int
	Timeout                              time.Duration
	Table, Output, Logs, DSN             string
	Driver                               string
	Mode, ListenAddress, TLSCert, TLSKey string
	PollInterval                         time.Duration
	Debug                                bool
	LogMaxBytes                          int64
	LogBackups                           int
	DebugHeartbeat                       time.Duration
}

func (c Config) s(k string) string { return c.Values[k] }
func (c Config) b(k string) bool   { return truth(c.s(k)) }
func loadConfig(path string, timeout time.Duration) (Config, error) {
	return loadConfigOverrides(path, timeout, nil)
}
func loadConfigOverrides(path string, timeout time.Duration, overrides map[string]string) (Config, error) {
	c := Config{Values: map[string]string{}, Arrays: map[string]map[string]string{}, Timeout: timeout}
	if e := c.readSource(defaultSettings); e != nil {
		return c, e
	}
	defaults := filepath.Join(filepath.Dir(path), "user-settings.ini-default")
	if _, e := os.Stat(defaults); e == nil {
		if e = c.read(defaults); e != nil {
			return c, e
		}
	}
	defaultsToAdd, e := configDefaults(path)
	if e != nil {
		return c, e
	}
	userSource, e := upgradeConfig(path, defaultsToAdd)
	if e != nil {
		return c, fmt.Errorf("load/update user settings: %w", e)
	}
	if e = c.readSource(userSource); e != nil {
		return c, e
	}
	for key, value := range overrides {
		c.Values[key] = value
	}
	if timeout > 0 {
		c.Values["http_timeout"] = timeout.String()
	}
	c.Timeout, e = time.ParseDuration(c.s("http_timeout"))
	if e != nil || c.Timeout <= 0 {
		return c, fmt.Errorf("http_timeout must be a positive duration, such as 30s")
	}
	c.PollInterval, e = time.ParseDuration(c.s("poll_interval"))
	if e != nil || c.PollInterval <= 0 {
		return c, fmt.Errorf("poll_interval must be a positive duration, such as 10m")
	}
	c.Mode = c.s("run_mode")
	if c.Mode != "poll" && c.Mode != "serve" && c.Mode != "serve-only" {
		return c, fmt.Errorf("run_mode must be poll, serve, or serve-only")
	}
	c.ListenAddress = c.s("listen_address")
	c.TLSCert = c.s("tls_cert")
	c.TLSKey = c.s("tls_key")
	c.Workers, e = strconv.Atoi(c.s("numParallelThreads"))
	if e != nil || c.Workers < 1 || c.Workers > 1024 {
		return c, fmt.Errorf("numParallelThreads must be 1..1024")
	}
	c.ExpireDays, e = strconv.Atoi(c.s("node_expire_interval"))
	if e != nil || c.ExpireDays < 1 {
		return c, fmt.Errorf("node_expire_interval must be positive")
	}
	c.Driver = strings.ToLower(c.s("SQL_TYPE"))
	if c.Driver == "" {
		c.Driver = "sqlite"
	}
	if c.Driver == "mariadb" {
		c.Driver = "mysql"
	}
	if c.Driver != "sqlite" && c.Driver != "mysql" {
		return c, fmt.Errorf("SQL_TYPE must be sqlite, mysql or mariadb")
	}

	c.Table = c.s("sql_db_tbl")
	if !regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(c.Table) {
		return c, fmt.Errorf("invalid sql_db_tbl")
	}
	c.Output = c.s("webpageDataDir")
	if c.Output == "" {
		return c, fmt.Errorf("webpageDataDir is required")
	}
	c.Logs = filepath.Join(filepath.Dir(filepath.Dir(path)), "logs")
	if dir := c.s("log_directory"); dir != "" {
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(filepath.Dir(filepath.Dir(path)), dir)
		}
		c.Logs = dir
	}
	c.Debug, e = strconv.ParseBool(c.s("debug_mode"))
	if e != nil {
		return c, fmt.Errorf("debug_mode must be true or false")
	}
	maxMB, e := strconv.ParseInt(c.s("log_max_size_mb"), 10, 64)
	if e != nil || maxMB < 1 || maxMB > 1024 {
		return c, fmt.Errorf("log_max_size_mb must be 1..1024")
	}
	c.LogMaxBytes = maxMB * 1024 * 1024
	c.LogBackups, e = strconv.Atoi(c.s("log_backups"))
	if e != nil || c.LogBackups < 0 || c.LogBackups > 20 {
		return c, fmt.Errorf("log_backups must be 0..20")
	}
	c.DebugHeartbeat, e = time.ParseDuration(c.s("debug_heartbeat_interval"))
	if e != nil || c.DebugHeartbeat < time.Second {
		return c, fmt.Errorf("debug_heartbeat_interval must be at least 1s")
	}
	mc := mysql.NewConfig()
	mc.User = c.s("sql_user")
	mc.Passwd = c.s("sql_passwd")
	mc.DBName = c.s("sql_db")
	mc.Net = "tcp"
	mc.Addr = c.s("sql_server")
	if strings.HasPrefix(mc.Addr, "/") {
		mc.Net = "unix"
	} else if _, _, e := net.SplitHostPort(mc.Addr); e != nil {
		mc.Addr = net.JoinHostPort(mc.Addr, "3306")
	}
	mc.Timeout = c.Timeout
	mc.ReadTimeout = c.Timeout
	mc.WriteTimeout = c.Timeout
	mc.Params = map[string]string{"charset": "utf8mb4"}
	c.DSN = mc.FormatDSN()
	if c.Driver == "sqlite" {
		name := c.s("sqlite_path")
		if name == "" {
			name = "../data/meshmap.sqlite"
		}
		if !filepath.IsAbs(name) {
			name = filepath.Join(filepath.Dir(path), name)
		}
		c.DSN = name
	}
	if c.s("localnode") == "" {
		return c, fmt.Errorf("localnode is required")
	}
	return c, nil
}
func (c *Config) read(path string) error { return c.readSource(path) }
func (c *Config) readSource(source any) error {
	f, e := ini.LoadSources(ini.LoadOptions{IgnoreInlineComment: true}, source)
	if e != nil {
		return e
	}
	re := regexp.MustCompile(`^([^\[]+)\[['"]?([^\]'"\[]+)['"]?\]$`)
	for _, s := range f.Sections() {
		for _, k := range s.Keys() {
			v := k.String()
			if m := re.FindStringSubmatch(k.Name()); m != nil {
				if c.Arrays[m[1]] == nil {
					c.Arrays[m[1]] = map[string]string{}
				}
				c.Arrays[m[1]][m[2]] = v
			} else {
				c.Values[k.Name()] = v
			}
		}
	}
	return nil
}
