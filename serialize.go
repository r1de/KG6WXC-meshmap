// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// PHP array/scalar serialization is retained for existing database consumers.
func php(v any) string {
	switch x := v.(type) {
	case nil:
		return "N;"
	case bool:
		if x {
			return "b:1;"
		}
		return "b:0;"
	case int:
		return fmt.Sprintf("i:%d;", x)
	case float64:
		return "d:" + strconv.FormatFloat(x, 'g', -1, 64) + ";"
	case jsonNumber:
		return "d:" + string(x) + ";"
	case Row:
		return php(map[string]any(x))
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var s strings.Builder
		fmt.Fprintf(&s, "a:%d:{", len(keys))
		for _, k := range keys {
			s.WriteString(php(k))
			s.WriteString(php(x[k]))
		}
		s.WriteByte('}')
		return s.String()
	case []any:
		var s strings.Builder
		fmt.Fprintf(&s, "a:%d:{", len(x))
		for i, v := range x {
			s.WriteString(php(i))
			s.WriteString(php(v))
		}
		s.WriteByte('}')
		return s.String()
	default:
		s := str(v)
		return fmt.Sprintf("s:%d:\"%s\";", len(s), s)
	}
}

type jsonNumber string

func unphp(s string) (any, error) {
	p := decoder{s: s}
	v, e := p.value(0)
	if e == nil && p.i != len(s) {
		e = fmt.Errorf("trailing serialized data")
	}
	return v, e
}

type decoder struct {
	s string
	i int
}

func (p *decoder) take(s string) error {
	if !strings.HasPrefix(p.s[p.i:], s) {
		return fmt.Errorf("invalid serialized data at %d", p.i)
	}
	p.i += len(s)
	return nil
}
func (p *decoder) until(c byte) (string, error) {
	j := strings.IndexByte(p.s[p.i:], c)
	if j < 0 {
		return "", fmt.Errorf("truncated serialized data")
	}
	s := p.s[p.i : p.i+j]
	p.i += j + 1
	return s, nil
}
func (p *decoder) value(depth int) (any, error) {
	if depth > 64 || p.i >= len(p.s) {
		return nil, fmt.Errorf("invalid serialization depth/length")
	}
	t := p.s[p.i]
	p.i++
	if t == 'N' {
		return nil, p.take(";")
	}
	if e := p.take(":"); e != nil {
		return nil, e
	}
	switch t {
	case 's':
		n, e := p.until(':')
		if e != nil {
			return nil, e
		}
		size, e := strconv.Atoi(n)
		if e != nil || size < 0 || size > len(p.s)-p.i-3 {
			return nil, fmt.Errorf("invalid string size")
		}
		if e = p.take("\""); e != nil {
			return nil, e
		}
		v := p.s[p.i : p.i+size]
		p.i += size
		return v, p.take("\";")
	case 'i', 'd', 'b':
		v, e := p.until(';')
		if e != nil {
			return nil, e
		}
		if t == 'b' {
			return v == "1", nil
		}
		if t == 'i' {
			return strconv.Atoi(v)
		}
		return strconv.ParseFloat(v, 64)
	case 'a':
		s, e := p.until(':')
		if e != nil {
			return nil, e
		}
		n, e := strconv.Atoi(s)
		if e != nil || n < 0 || n > len(p.s) {
			return nil, fmt.Errorf("invalid array size")
		}
		if e = p.take("{"); e != nil {
			return nil, e
		}
		m := Row{}
		a := make([]any, 0)
		list := true
		for i := 0; i < n; i++ {
			k, e := p.value(depth + 1)
			if e != nil {
				return nil, e
			}
			v, e := p.value(depth + 1)
			if e != nil {
				return nil, e
			}
			m[str(k)] = v
			if ki, ok := k.(int); !ok || ki != i {
				list = false
			}
			a = append(a, v)
		}
		if e = p.take("}"); e != nil {
			return nil, e
		}
		if list && n > 0 {
			return a, nil
		}
		return m, nil
	}
	return nil, fmt.Errorf("unsupported PHP type %q", t)
}
