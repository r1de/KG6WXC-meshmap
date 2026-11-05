// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/ini.v1"
)

var arraySetting = regexp.MustCompile(`^([^\[]+)\[['"]?([^\]'"\[]+)['"]?\]$`)

func settingID(name string) string {
	if m := arraySetting.FindStringSubmatch(name); m != nil {
		return strings.TrimSpace(m[1]) + "[" + strings.TrimSpace(m[2]) + "]"
	}
	return name
}

type settingDefault struct{ name, value, comment string }

func configDefaults(path string) ([]settingDefault, error) {
	sources := [][]byte{defaultSettings}
	sibling := filepath.Join(filepath.Dir(path), "user-settings.ini-default")
	if b, e := os.ReadFile(sibling); e == nil {
		sources = append(sources, b)
	} else if !os.IsNotExist(e) {
		return nil, e
	}
	entries := []settingDefault{}
	index := map[string]int{}
	for _, source := range sources {
		f, e := ini.LoadSources(ini.LoadOptions{IgnoreInlineComment: true}, source)
		if e != nil {
			return nil, e
		}
		for _, section := range f.Sections() {
			for _, key := range section.Keys() {
				entry := settingDefault{key.Name(), key.Value(), key.Comment}
				id := settingID(key.Name())
				if i, ok := index[id]; ok {
					if entry.comment == "" {
						entry.comment = entries[i].comment
					}
					entries[i] = entry
				} else {
					index[id] = len(entries)
					entries = append(entries, entry)
				}
			}
		}
	}
	return entries, nil
}
func missingSettings(original []byte, defaults []settingDefault) ([]byte, error) {
	f, e := ini.LoadSources(ini.LoadOptions{IgnoreInlineComment: true}, original)
	if e != nil {
		return nil, e
	}
	present := map[string]bool{}
	for _, s := range f.Sections() {
		for _, k := range s.Keys() {
			present[settingID(k.Name())] = true
		}
	}
	additions := ini.Empty()
	section := additions.Section("user-settings")
	count := 0
	for _, entry := range defaults {
		if present[settingID(entry.name)] {
			continue
		}
		key, e := section.NewKey(entry.name, entry.value)
		if e != nil {
			return nil, e
		}
		key.Comment = entry.comment
		count++
	}
	if count == 0 {
		return nil, nil
	}
	var block bytes.Buffer
	if _, e = additions.WriteTo(&block); e != nil {
		return nil, e
	}
	newline := "\n"
	if bytes.Contains(original, []byte("\r\n")) {
		newline = "\r\n"
	}
	text := strings.ReplaceAll(strings.ReplaceAll(block.String(), "\r\n", "\n"), "\n", newline)
	// The original file remains an exact byte-for-byte prefix. Repeating the
	// section header is intentional: the user's last section may be unrelated.
	return append(append([]byte{}, original...), []byte(newline+newline+"; MeshMap: added missing default settings. Existing settings are unchanged."+newline+text)...), nil
}

// Append only missing keys, never regenerate the user's file. The sidecar lock
// serializes cooperating processes and atomic replacement prevents partial INI
// writes. A complete read-only configuration needs no writable directory/lock.
func upgradeConfig(path string, defaults []settingDefault) ([]byte, error) {
	target, e := filepath.Abs(path)
	if e != nil {
		return nil, e
	}
	target, e = filepath.EvalSymlinks(target)
	if e != nil {
		return nil, e
	}
	before, e := os.ReadFile(target)
	if e != nil {
		return nil, e
	}
	updated, e := missingSettings(before, defaults)
	if e != nil {
		return nil, e
	}
	if updated == nil {
		return before, nil
	}
	lock, e := os.OpenFile(target+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return nil, fmt.Errorf("cannot lock settings for adding new defaults: %w", e)
	}
	defer lock.Close()
	if e = lockFile(lock, true); e != nil {
		return nil, e
	}
	defer unlockFile(lock)
	before, e = os.ReadFile(target)
	if e != nil {
		return nil, e
	}
	updated, e = missingSettings(before, defaults)
	if e != nil {
		return nil, e
	}
	if updated == nil {
		return before, nil
	}
	info, e := os.Stat(target)
	if e != nil {
		return nil, e
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("settings must be a regular file")
	}
	f, e := os.CreateTemp(filepath.Dir(target), ".meshmap-settings-*")
	if e != nil {
		return nil, fmt.Errorf("cannot add new settings defaults: %w", e)
	}
	defer os.Remove(f.Name())
	fail := func(e error) ([]byte, error) { f.Close(); return nil, e }
	if e = preserveFileOwner(f, info); e != nil {
		return fail(e)
	}
	if e = f.Chmod(info.Mode().Perm()); e != nil {
		return fail(e)
	}
	if _, e = f.Write(updated); e != nil {
		return fail(e)
	}
	if e = f.Sync(); e != nil {
		return fail(e)
	}
	if e = f.Close(); e != nil {
		return nil, e
	}
	current, e := os.ReadFile(target)
	if e != nil {
		return nil, e
	}
	if !bytes.Equal(current, before) {
		return nil, fmt.Errorf("settings changed while adding defaults; restart to retry")
	}
	if e = os.Rename(f.Name(), target); e != nil {
		return nil, e
	}
	return updated, nil
}
