package rules

import (
	"bytes"
	"strings"
	"testing"
	"testing/quick"

	"github.com/yuin/goldmark/text"
)

// TestMaskMDXPreservesOffsets pins the two invariants the whole approach
// rests on: masking never changes the length of the body and never
// removes a newline. If either breaks, every line number reported for an
// MDX file silently shifts.
func TestMaskMDXPreservesOffsets(t *testing.T) {
	f := func(s string) bool {
		in := []byte(s)
		out := MaskMDX(in)
		return len(out) == len(in) &&
			bytes.Count(out, []byte("\n")) == bytes.Count(in, []byte("\n"))
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 2000}); err != nil {
		t.Error(err)
	}
}

// TestMaskMDXPreservesOffsetsRealistic runs the same invariants over MDX
// shapes, which random strings rarely produce.
func TestMaskMDXPreservesOffsetsRealistic(t *testing.T) {
	for _, src := range []string{
		"import Foo from './foo'\n\nText.\n",
		"<Callout type=\"warning\">\n  Prose.\n</Callout>\n",
		"<Grid cols={{ sm: 1, lg: 3 }} />\n",
		"{/* a comment */}\n",
		"Text with {value} inline.\n",
		"```jsx\n<Callout>x</Callout>\n```\n",
		"Unclosed {brace and <Tag\n",
		"export const meta = {\n  a: 1,\n}\n",
	} {
		in := []byte(src)
		out := MaskMDX(in)
		if len(out) != len(in) {
			t.Errorf("length changed for %q: %d -> %d", src, len(in), len(out))
		}
		if got, want := bytes.Count(out, []byte("\n")), bytes.Count(in, []byte("\n")); got != want {
			t.Errorf("newline count changed for %q: %d -> %d", src, want, got)
		}
	}
}

// surviving reports the text left after masking, with whitespace runs
// collapsed. Exact space counts are an artefact of the input's width and
// are already covered by the length invariant; what each case here cares
// about is which text still reaches the rules.
func surviving(b []byte) string {
	return strings.Join(strings.Fields(string(b)), " ")
}

func TestMaskMDX(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "plain markdown untouched",
			in:   "# Heading\n\nSome **bold** prose with `code`.\n",
			want: "# Heading Some **bold** prose with `code`.",
		},
		{
			name: "import statement masked",
			in:   "import Callout from './Callout'\n\nProse.\n",
			want: "Prose.",
		},
		{
			name: "multi-line named import masked",
			in:   "import {\n  A,\n  B,\n} from './x'\n\nProse.\n",
			want: "Prose.",
		},
		{
			name: "export declaration masked",
			in:   "export const meta = { title: 'x' }\n\nProse.\n",
			want: "Prose.",
		},
		{
			name: "prose starting with export is not code",
			in:   "export your data before deleting it.\n",
			want: "export your data before deleting it.",
		},
		{
			name: "prose starting with import is not code",
			in:   "import the file into the editor.\n",
			want: "import the file into the editor.",
		},
		{
			name: "jsx tags masked but children kept",
			in:   "<Callout type=\"warning\">Real prose here</Callout>\n",
			want: "Real prose here",
		},
		{
			name: "self-closing tag masked",
			in:   "Text <Icon name=\"x\" /> more.\n",
			want: "Text more.",
		},
		{
			name: "braces in attributes do not end the tag early",
			in:   "<Grid cols={{ sm: 1 }}>Body</Grid>\n",
			want: "Body",
		},
		{
			name: "greater-than inside attribute string",
			in:   "<Note label=\"a > b\">Body</Note>\n",
			want: "Body",
		},
		{
			name: "expression masked",
			in:   "Value is {count + 1} today.\n",
			want: "Value is today.",
		},
		{
			name: "mdx comment masked",
			in:   "{/* not prose */}\n\nProse.\n",
			want: "Prose.",
		},
		{
			name: "brace inside string does not desync",
			in:   "{ '}' }Text\n",
			want: "Text",
		},
		{
			name: "fenced code block untouched",
			in:   "```jsx\n<Callout>x</Callout>\n```\n",
			want: "```jsx <Callout>x</Callout> ```",
		},
		{
			name: "inline code span untouched",
			in:   "Use `<Callout>` for notes.\n",
			want: "Use `<Callout>` for notes.",
		},
		{
			name: "autolink not mistaken for a tag",
			in:   "See <https://example.com> for more.\n",
			want: "See <https://example.com> for more.",
		},
		{
			name: "email autolink not mistaken for a tag",
			in:   "Mail <a@b.com> today.\n",
			want: "Mail <a@b.com> today.",
		},
		{
			name: "unbalanced brace left alone",
			in:   "Cost is {5 dollars.\n",
			want: "Cost is {5 dollars.",
		},
		{
			name: "unterminated tag left alone",
			in:   "A < B and C > D.\n",
			want: "A < B and C > D.",
		},
		{
			name: "multi-line tag masked",
			in:   "<Callout\n  type=\"warning\"\n>\nProse.\n</Callout>\n",
			want: "Prose.",
		},
		{
			name: "fragments masked",
			in:   "<>Prose.</>\n",
			want: "Prose.",
		},
		{
			name: "html tags masked like components",
			in:   "<div class=\"x\">Prose.</div>\n",
			want: "Prose.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := surviving(MaskMDX([]byte(tt.in))); got != tt.want {
				t.Errorf("surviving text\n in:   %q\n got:  %q\n want: %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestMaskMDXLineNumbers is the end-to-end check that matters: after
// masking, an offset still resolves to the line it came from.
func TestMaskMDXLineNumbers(t *testing.T) {
	src := "import Foo from './foo'\n\n<Callout>\n  A sentence.\n</Callout>\n"
	masked := MaskMDX([]byte(src))

	idx := bytes.Index(masked, []byte("A sentence."))
	if idx < 0 {
		t.Fatal("prose was masked away")
	}
	f := &MarkdownFile{Body: masked, BodyStartLine: 1}
	if got := f.LineAt(idx); got != 4 {
		t.Errorf("LineAt = %d, want 4", got)
	}
}

// TestMaskMDXThroughProseFlattening confirms the payoff: JSX never
// reaches the prose spans the rules consume, while its children do.
func TestMaskMDXThroughProseFlattening(t *testing.T) {
	src := "import Callout from './Callout'\n\n<Callout type=\"warning\">\n  Real prose here.\n</Callout>\n"
	masked := MaskMDX([]byte(src))
	root := testParser.Parse(text.NewReader(masked))

	var sb strings.Builder
	for _, blk := range FlattenProse(masked, root) {
		for _, sp := range blk.Spans {
			sb.Write(sp.Text)
		}
	}
	prose := sb.String()

	if !strings.Contains(prose, "Real prose here.") {
		t.Errorf("prose lost: %q", prose)
	}
	for _, bad := range []string{"Callout", "warning", "import"} {
		if strings.Contains(prose, bad) {
			t.Errorf("MDX syntax %q leaked into prose: %q", bad, prose)
		}
	}
}
