package main

import "bytes"

func revert(s []byte) []byte {
	runes := []rune(string(s))
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return []byte(string(runes))
}
func unescape(b []byte) []byte {
	b = bytes.ReplaceAll(b, []byte(`\\`), []byte(`\`))
	b = bytes.ReplaceAll(b, []byte(`\/`), []byte(`/`))
	return b
}

func escape(b []byte) []byte {
	b = bytes.ReplaceAll(b, []byte(`\`), []byte(`\\`))
	b = bytes.ReplaceAll(b, []byte(`/`), []byte(`\/`))
	return b
}
