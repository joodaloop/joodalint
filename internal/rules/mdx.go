package rules

import "bytes"

// MaskMDX returns a copy of body with MDX-specific syntax — ESM
// import/export statements, {...} expressions, and JSX tags — painted
// over with spaces, so the CommonMark parser and every prose rule
// downstream never see them.
//
// Two invariants make this safe to run before parsing:
//
//   - the result has the same length as the input, and
//   - every newline survives masking.
//
// Together they mean a byte offset into the masked body still resolves
// to the same line as in the original, so MarkdownFile.LineAt keeps
// reporting correct line numbers without any rule being aware masking
// happened.
//
// JSX *children* are deliberately left intact — only the tags around
// them are painted over — so prose wrapped in a component is still
// checked:
//
//	<Callout type="warning">Real prose here</Callout>
//
// leaves "Real prose here" for the rules and blanks the rest.
//
// Expressions are delimited by brace counting rather than parsed as
// JavaScript. This mirrors the acorn-less mode of MDX's own expression
// extension: the contents are opaque, so they only need to be located,
// not understood. Unbalanced or malformed constructs are left untouched
// rather than guessed at, on the theory that a missed mask produces a
// stray diagnostic while an overreaching one silently eats prose.
//
// Fenced code blocks and inline code spans are skipped, so documentation
// *about* JSX keeps its examples. Indented code blocks are not treated
// specially: their contents may be masked, which is harmless because
// FlattenProse already excludes them from prose.
func MaskMDX(body []byte) []byte {
	out := make([]byte, len(body))
	copy(out, body)

	i := 0
	lineStart := true
	for i < len(body) {
		if lineStart {
			if end, ok := skipFencedBlock(body, i); ok {
				i = end
				continue
			}
			if end, ok := scanESM(body, i); ok {
				maskRange(out, i, end)
				i = end
				continue
			}
		}

		switch c := body[i]; {
		case c == '\n':
			i++
			lineStart = true
			continue
		case c == '`':
			if end, ok := skipCodeSpan(body, i); ok {
				i = end
				lineStart = false
				continue
			}
		case c == '{':
			if end, ok := scanExpression(body, i); ok {
				maskRange(out, i, end)
				i = end
				lineStart = false
				continue
			}
		case c == '<':
			if end, ok := scanJSXTag(body, i); ok {
				maskRange(out, i, end)
				i = end
				lineStart = false
				continue
			}
		}

		if body[i] != ' ' && body[i] != '\t' {
			lineStart = false
		}
		i++
	}
	return out
}

// maskRange blanks body[start:end], preserving newlines so line numbers
// downstream are unaffected. This is the single place the length- and
// newline-preserving invariants are enforced.
func maskRange(out []byte, start, end int) {
	for i := start; i < end && i < len(out); i++ {
		if out[i] != '\n' {
			out[i] = ' '
		}
	}
}

func lineEnd(b []byte, i int) int {
	for ; i < len(b); i++ {
		if b[i] == '\n' {
			return i
		}
	}
	return len(b)
}

// fenceMarker reports the fence character and run length if the line
// beginning at i opens or closes a code fence.
func fenceMarker(b []byte, i int) (byte, int, bool) {
	j := i
	for j < len(b) && j-i < 4 && b[j] == ' ' {
		j++
	}
	if j >= len(b) || (b[j] != '`' && b[j] != '~') {
		return 0, 0, false
	}
	ch := b[j]
	n := 0
	for ; j < len(b) && b[j] == ch; j++ {
		n++
	}
	if n < 3 {
		return 0, 0, false
	}
	return ch, n, true
}

// skipFencedBlock returns the offset just past a fenced code block
// starting at line offset i. An unterminated fence runs to end of input,
// matching how CommonMark treats it.
func skipFencedBlock(b []byte, i int) (int, bool) {
	ch, n, ok := fenceMarker(b, i)
	if !ok {
		return 0, false
	}
	pos := lineEnd(b, i)
	for pos < len(b) {
		pos++ // step over the newline
		if pos >= len(b) {
			break
		}
		if c, m, ok := fenceMarker(b, pos); ok && c == ch && m >= n {
			return lineEnd(b, pos), true
		}
		pos = lineEnd(b, pos)
	}
	return len(b), true
}

// skipCodeSpan returns the offset just past an inline code span opening
// at i. Per CommonMark the closing run must have the same length.
func skipCodeSpan(b []byte, i int) (int, bool) {
	n := 0
	for ; i+n < len(b) && b[i+n] == '`'; n++ {
	}
	j := i + n
	for j < len(b) {
		if b[j] != '`' {
			j++
			continue
		}
		m := 0
		for ; j+m < len(b) && b[j+m] == '`'; m++ {
		}
		if m == n {
			return j + m, true
		}
		j += m
	}
	return 0, false
}

// esmDeclarators are the tokens that may follow `export`. Bare prose
// starting with "export" or "import" — "import the file into..." — must
// not be mistaken for code, so an import requires a following module
// specifier and an export requires a declaration keyword.
var esmDeclarators = [][]byte{
	[]byte("default"), []byte("const"), []byte("let"), []byte("var"),
	[]byte("function"), []byte("class"), []byte("async"), []byte("{"),
	[]byte("*"),
}

// scanESM returns the end offset of an ESM import/export statement
// beginning at line offset i, following it across lines until its
// brackets balance.
func scanESM(b []byte, i int) (int, bool) {
	var kw []byte
	switch {
	case hasPrefixAt(b, i, []byte("import")):
		kw = []byte("import")
	case hasPrefixAt(b, i, []byte("export")):
		kw = []byte("export")
	default:
		return 0, false
	}

	j := i + len(kw)
	if j < len(b) && b[j] != ' ' && b[j] != '{' && b[j] != '\t' {
		return 0, false
	}
	for j < len(b) && (b[j] == ' ' || b[j] == '\t') {
		j++
	}
	if j >= len(b) {
		return 0, false
	}

	if string(kw) == "export" {
		matched := false
		for _, d := range esmDeclarators {
			if hasPrefixAt(b, j, d) {
				matched = true
				break
			}
		}
		if !matched {
			return 0, false
		}
	}

	// Walk to the end of the statement, tracking bracket depth and string
	// state so multi-line named imports are consumed whole.
	depth := 0
	pos := i
	for pos < len(b) {
		end := lineEnd(b, pos)
		for ; pos < end; pos++ {
			switch c := b[pos]; c {
			case '{', '(', '[':
				depth++
			case '}', ')', ']':
				if depth > 0 {
					depth--
				}
			case '\'', '"', '`':
				if p, ok := skipJSString(b, pos, end); ok {
					pos = p - 1
				}
			}
		}
		if depth == 0 {
			// An import with no specifier is prose, not code.
			if string(kw) == "import" && !hasFrom(b, i, end) {
				return 0, false
			}
			return end, true
		}
		pos = end + 1
	}
	return len(b), true
}

// hasFrom reports whether an import statement carries a module
// specifier — either `from "x"` or the side-effect form `import "x"`.
func hasFrom(b []byte, start, end int) bool {
	for i := start; i < end; i++ {
		if b[i] == '\'' || b[i] == '"' {
			return true
		}
		if hasPrefixAt(b, i, []byte("from")) {
			return true
		}
	}
	return false
}

// skipJSString steps over a quoted JavaScript string, honouring
// backslash escapes. Template literals are not followed across lines;
// the caller's bracket tracking recovers.
func skipJSString(b []byte, i, limit int) (int, bool) {
	quote := b[i]
	for j := i + 1; j < limit; j++ {
		if b[j] == '\\' {
			j++
			continue
		}
		if b[j] == quote {
			return j + 1, true
		}
	}
	return 0, false
}

// scanExpression returns the end offset of a {...} MDX expression
// opening at i, counting braces while ignoring those inside strings and
// comments. Returns false when the braces never balance, leaving the
// text unmasked rather than swallowing the rest of the document.
func scanExpression(b []byte, i int) (int, bool) {
	depth := 0
	for j := i; j < len(b); j++ {
		switch c := b[j]; c {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return j + 1, true
			}
		case '\'', '"', '`':
			if p, ok := skipJSStringMultiline(b, j); ok {
				j = p - 1
			}
		case '/':
			if j+1 < len(b) && b[j+1] == '/' {
				j = lineEnd(b, j) - 1
			} else if j+1 < len(b) && b[j+1] == '*' {
				if p, ok := skipBlockComment(b, j); ok {
					j = p - 1
				}
			}
		}
	}
	return 0, false
}

// skipJSStringMultiline is skipJSString without a line limit, for string
// literals inside expressions where template literals legitimately span
// lines.
func skipJSStringMultiline(b []byte, i int) (int, bool) {
	quote := b[i]
	for j := i + 1; j < len(b); j++ {
		if b[j] == '\\' {
			j++
			continue
		}
		if b[j] == quote {
			return j + 1, true
		}
		if quote != '`' && b[j] == '\n' {
			return 0, false
		}
	}
	return 0, false
}

func skipBlockComment(b []byte, i int) (int, bool) {
	for j := i + 2; j+1 < len(b); j++ {
		if b[j] == '*' && b[j+1] == '/' {
			return j + 2, true
		}
	}
	return 0, false
}

// scanJSXTag returns the end offset of a JSX opening, closing, or
// self-closing tag starting at i. Fragments (<> and </>) are recognised
// too. Autolinks such as <https://example.com> and <a@b.com> are
// rejected, since ':' and '@' cannot appear in a tag name.
func scanJSXTag(b []byte, i int) (int, bool) {
	j := i + 1
	if j < len(b) && b[j] == '/' {
		j++
	}
	if j < len(b) && b[j] == '>' { // <> or </>
		return j + 1, true
	}
	if j >= len(b) || !isTagNameStart(b[j]) {
		return 0, false
	}
	for j < len(b) && isTagNameByte(b[j]) {
		j++
	}
	// The name must end at a boundary; ':' or '@' means this is a URL or
	// an email autolink, not a tag.
	if j >= len(b) {
		return 0, false
	}
	switch b[j] {
	case ' ', '\t', '\n', '\r', '/', '>':
	default:
		return 0, false
	}

	// Consume attributes, respecting quoted values and {...} braces so a
	// '>' inside either doesn't end the tag early.
	depth := 0
	for ; j < len(b); j++ {
		switch c := b[j]; c {
		case '{':
			depth++
		case '}':
			if depth > 0 {
				depth--
			}
		case '\'', '"':
			if p, ok := skipJSStringMultiline(b, j); ok {
				j = p - 1
			}
		case '>':
			if depth == 0 {
				return j + 1, true
			}
		}
	}
	return 0, false
}

func isTagNameStart(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' || c == '$'
}

func isTagNameByte(c byte) bool {
	return isTagNameStart(c) || c >= '0' && c <= '9' || c == '.' || c == '-'
}

func hasPrefixAt(b []byte, i int, prefix []byte) bool {
	if i+len(prefix) > len(b) {
		return false
	}
	return bytes.Equal(b[i:i+len(prefix)], prefix)
}
