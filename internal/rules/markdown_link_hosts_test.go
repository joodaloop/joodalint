package rules

import "testing"

func TestLinkHosts_ValidHostsOK(t *testing.T) {
	for _, in := range []string{
		"[a](https://example.com/path)\n",
		"[a](http://sub.example.co.uk)\n",
		"[a](https://example.com:8080/x)\n",
	} {
		diags := markdownLinkHosts{}.Check(mdFile(in), nil)
		if len(diags) != 0 {
			t.Errorf("input %q should be valid, got %v", in, messages(diags))
		}
	}
}

func TestLinkHosts_InvalidHosts(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"[a](https://example/path)\n", "host has no dot"},
		{"[a](https://exa..mple.com)\n", "consecutive dots"},
		{"[a](https://-bad.com)\n", "label starts or ends with hyphen"},
		{"[a](https://bad-.com)\n", "label starts or ends with hyphen"},
		{"[a](https://bad_host.com)\n", "invalid characters in host"},
	}
	for _, tc := range cases {
		diags := markdownLinkHosts{}.Check(mdFile(tc.in), nil)
		if !containsMsg(diags, tc.want) {
			t.Errorf("input %q: want %q, got %v", tc.in, tc.want, messages(diags))
		}
	}
}

func TestLinkHosts_SkippedSchemes(t *testing.T) {
	for _, in := range []string{
		"[a](mailto:foo@bar.com)\n",
		"[a](mailto:foo+tag@bar.com?subject=hi)\n",
		"[a](tel:+15551234)\n",
		"[a](javascript:alert(1))\n",
		"[a](data:text/plain,hi)\n",
	} {
		diags := markdownLinkHosts{}.Check(mdFile(in), nil)
		if len(diags) != 0 {
			t.Errorf("input %q should be skipped, got %v", in, messages(diags))
		}
	}
}

func TestLinkHosts_InvalidMailto(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"[a](mailto:)\n", "empty mailto address"},
		{"[a](mailto:foo)\n", "mailto missing @"},
		{"[a](mailto:@bar.com)\n", "mailto empty local part"},
		{"[a](mailto:.foo@bar.com)\n", "mailto invalid local part"},
		{"[a](mailto:foo.@bar.com)\n", "mailto invalid local part"},
		{"[a](mailto:fo..o@bar.com)\n", "mailto invalid local part"},
		{"[a](mailto:foo bar@bar.com)\n", "mailto invalid characters in local part"},
		{"[a](mailto:foo,bar@bar.com)\n", "mailto invalid characters in local part"},
		{"[a](mailto:foo@bar)\n", "mailto host has no dot"},
		{"[a](mailto:foo@bad_host.com)\n", "mailto invalid characters in host"},
	}
	for _, tc := range cases {
		diags := markdownLinkHosts{}.Check(mdFile(tc.in), nil)
		if !containsMsg(diags, tc.want) {
			t.Errorf("input %q: want %q, got %v", tc.in, tc.want, messages(diags))
		}
	}
}

func TestLinkHosts_RootAndFragmentSkipped(t *testing.T) {
	for _, in := range []string{
		"[a](/foo)\n",
		"[a](#bar)\n",
	} {
		diags := markdownLinkHosts{}.Check(mdFile(in), nil)
		if len(diags) != 0 {
			t.Errorf("input %q should be skipped, got %v", in, messages(diags))
		}
	}
}

func TestStripTitle(t *testing.T) {
	cases := []struct{ in, want string }{
		{`https://x.com`, `https://x.com`},
		{`https://x.com "Title"`, `https://x.com`},
		{`  https://x.com  `, `https://x.com`},
	}
	for _, tc := range cases {
		if got := stripTitle(tc.in); got != tc.want {
			t.Errorf("stripTitle(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestLinkHosts_ID(t *testing.T) {
	if (markdownLinkHosts{}).ID() != "link-host" {
		t.Fatal("wrong ID")
	}
}
