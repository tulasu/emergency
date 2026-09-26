package models

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"sort"
)

func BankDigest(slots map[string]string, questions map[string][]string) string {
	canonQ := map[string][]string{}
	for k, v := range questions {
		cp := append([]string{}, v...)
		sort.Strings(cp)
		canonQ[k] = cp
	}
	var buf bytes.Buffer
	buf.WriteString(`{"questions": {`)
	writeStrMapList(&buf, canonQ)
	buf.WriteString(`}, "slots": {`)
	writeStrMap(&buf, slots)
	buf.WriteString(`}}`)
	sum := sha256.Sum256(buf.Bytes())
	return fmt.Sprintf("%x", sum)[:16]
}

func writeStrMap(buf *bytes.Buffer, m map[string]string) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i, k := range keys {
		if i > 0 {
			buf.WriteString(", ")
		}
		writeJSONString(buf, k)
		buf.WriteString(": ")
		writeJSONString(buf, m[k])
	}
}

func writeStrMapList(buf *bytes.Buffer, m map[string][]string) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i, k := range keys {
		if i > 0 {
			buf.WriteString(", ")
		}
		writeJSONString(buf, k)
		buf.WriteString(": [")
		for j, v := range m[k] {
			if j > 0 {
				buf.WriteString(", ")
			}
			writeJSONString(buf, v)
		}
		buf.WriteString("]")
	}
}

func writeJSONString(buf *bytes.Buffer, s string) {
	buf.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\\\`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(buf, `\u%04x`, r)
			} else {
				buf.WriteRune(r)
			}
		}
	}
	buf.WriteByte('"')
}
