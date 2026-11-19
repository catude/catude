package claudecfg

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// A tiny order-preserving JSON tree. encoding/json loses key order when decoding into maps, and writing
// a settings file back with its keys shuffled would make every edit a noisy diff — so files are parsed
// into this tree, changed in place and written back in the original order.

type node struct {
	kind byte // 'o' object, 'a' array, 's' string, 'n' number, 'b' bool, 'z' null
	keys []string
	vals []*node // object values (same order as keys) or array items
	str  string
	num  json.Number
	b    bool
}

func parse(data []byte) (*node, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	n, err := readValue(dec, 0)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("unexpected content after the JSON value")
	}
	return n, nil
}

// maxDepth bounds nesting: a file made of millions of "[" must not exhaust the stack.
const maxDepth = 64

func readValue(dec *json.Decoder, depth int) (*node, error) {
	if depth > maxDepth {
		return nil, errors.New("nested too deeply")
	}
	t, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch v := t.(type) {
	case json.Delim:
		switch v {
		case '{':
			n := &node{kind: 'o'}
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return nil, err
				}
				k, ok := kt.(string)
				if !ok {
					return nil, fmt.Errorf("object key is not a string: %v", kt)
				}
				val, err := readValue(dec, depth+1)
				if err != nil {
					return nil, err
				}
				n.keys, n.vals = append(n.keys, k), append(n.vals, val)
			}
			_, err := dec.Token()
			return n, err
		case '[':
			n := &node{kind: 'a'}
			for dec.More() {
				val, err := readValue(dec, depth+1)
				if err != nil {
					return nil, err
				}
				n.vals = append(n.vals, val)
			}
			_, err := dec.Token()
			return n, err
		}
		return nil, fmt.Errorf("unexpected %v", v)
	case string:
		return &node{kind: 's', str: v}, nil
	case json.Number:
		return &node{kind: 'n', num: v}, nil
	case bool:
		return &node{kind: 'b', b: v}, nil
	case nil:
		return &node{kind: 'z'}, nil
	}
	return nil, fmt.Errorf("unexpected token %v", t)
}

func (n *node) get(key string) *node {
	if n == nil || n.kind != 'o' {
		return nil
	}
	for i, k := range n.keys {
		if k == key {
			return n.vals[i]
		}
	}
	return nil
}

// at walks a path of object keys.
func (n *node) at(path []string) *node {
	cur := n
	for _, p := range path {
		cur = cur.get(p)
		if cur == nil {
			return nil
		}
	}
	return cur
}

// set stores a value at a path, creating intermediate objects (new keys go at the end).
func (n *node) set(path []string, v *node) error {
	if n.kind != 'o' {
		return errors.New("the file is not a JSON object")
	}
	cur := n
	for i, p := range path {
		last := i == len(path)-1
		idx := -1
		for j, k := range cur.keys {
			if k == p {
				idx = j
			}
		}
		if last {
			if idx >= 0 {
				cur.vals[idx] = v
			} else {
				cur.keys, cur.vals = append(cur.keys, p), append(cur.vals, v)
			}
			return nil
		}
		if idx < 0 {
			child := &node{kind: 'o'}
			cur.keys, cur.vals = append(cur.keys, p), append(cur.vals, child)
			cur = child
			continue
		}
		if cur.vals[idx].kind != 'o' {
			return fmt.Errorf("%s is not an object", strings.Join(path[:i+1], "."))
		}
		cur = cur.vals[idx]
	}
	return nil
}

func (n *node) encode() []byte {
	var b bytes.Buffer
	n.write(&b, 0)
	b.WriteByte('\n')
	return b.Bytes()
}

func jsonString(s string) string {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.Encode(s)
	return strings.TrimRight(b.String(), "\n")
}

func (n *node) write(b *bytes.Buffer, depth int) {
	ind := strings.Repeat("  ", depth+1)
	switch n.kind {
	case 'o':
		if len(n.keys) == 0 {
			b.WriteString("{}")
			return
		}
		b.WriteString("{\n")
		for i, k := range n.keys {
			b.WriteString(ind + jsonString(k) + ": ")
			n.vals[i].write(b, depth+1)
			if i < len(n.keys)-1 {
				b.WriteByte(',')
			}
			b.WriteByte('\n')
		}
		b.WriteString(strings.Repeat("  ", depth) + "}")
	case 'a':
		if len(n.vals) == 0 {
			b.WriteString("[]")
			return
		}
		b.WriteString("[\n")
		for i, v := range n.vals {
			b.WriteString(ind)
			v.write(b, depth+1)
			if i < len(n.vals)-1 {
				b.WriteByte(',')
			}
			b.WriteByte('\n')
		}
		b.WriteString(strings.Repeat("  ", depth) + "]")
	case 's':
		b.WriteString(jsonString(n.str))
	case 'n':
		b.WriteString(n.num.String())
	case 'b':
		if n.b {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	default:
		b.WriteString("null")
	}
}

// text renders a scalar or a short summary of a container, for display.
func (n *node) text() string {
	switch n.kind {
	case 's':
		return n.str
	case 'n':
		return n.num.String()
	case 'b':
		if n.b {
			return "true"
		}
		return "false"
	case 'z':
		return "null"
	case 'a':
		var parts []string
		for _, v := range n.vals {
			if v.kind == 'o' || v.kind == 'a' {
				return fmt.Sprintf("[%d items]", len(n.vals))
			}
			parts = append(parts, v.text())
		}
		return "[" + strings.Join(parts, ", ") + "]"
	}
	return fmt.Sprintf("{%d keys}", len(n.keys))
}

func (n *node) typeName() string {
	return map[byte]string{'o': "object", 'a': "list", 's': "text", 'n': "number", 'b': "yes/no", 'z': "empty"}[n.kind]
}

// del removes the value at a path; empty parent objects are removed too. Missing paths are not an error.
func (n *node) del(path []string) {
	if n == nil || n.kind != 'o' || len(path) == 0 {
		return
	}
	for i, k := range n.keys {
		if k != path[0] {
			continue
		}
		if len(path) == 1 {
			n.keys = append(n.keys[:i:i], n.keys[i+1:]...)
			n.vals = append(n.vals[:i:i], n.vals[i+1:]...)
			return
		}
		child := n.vals[i]
		child.del(path[1:])
		if child.kind == 'o' && len(child.keys) == 0 {
			n.keys = append(n.keys[:i:i], n.keys[i+1:]...)
			n.vals = append(n.vals[:i:i], n.vals[i+1:]...)
		}
		return
	}
}
