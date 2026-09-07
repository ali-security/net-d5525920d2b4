// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package idna

import (
	"testing"
)

var idnaTestCases = [...]struct {
	ascii, unicode string
}{
	// Labels.
	{"books", "books"},
	{"xn--bcher-kva", "bücher"},

	// Domains.
	{"foo--xn--bar.org", "foo--xn--bar.org"},
	{"golang.org", "golang.org"},
	{"example.xn--p1ai", "example.рф"},
	{"xn--czrw28b.tw", "商業.tw"},
	{"www.xn--mller-kva.de", "www.müller.de"},
}

func TestIDNA(t *testing.T) {
	for _, tc := range idnaTestCases {
		if a, err := ToASCII(tc.unicode); err != nil {
			t.Errorf("ToASCII(%q): %v", tc.unicode, err)
		} else if a != tc.ascii {
			t.Errorf("ToASCII(%q): got %q, want %q", tc.unicode, a, tc.ascii)
		}

		if u, err := ToUnicode(tc.ascii); err != nil {
			t.Errorf("ToUnicode(%q): %v", tc.ascii, err)
		} else if u != tc.unicode {
			t.Errorf("ToUnicode(%q): got %q, want %q", tc.ascii, u, tc.unicode)
		}
	}
}

func TestIDNASeparators(t *testing.T) {
	type subCase struct {
		unicode   string
		wantASCII string
		wantErr   bool
	}

	testCases := []struct {
		name     string
		profile  *Profile
		subCases []subCase
	}{
		{
			name: "Punycode", profile: Punycode,
			subCases: []subCase{
				{"example\u3002jp", "xn--examplejp-ck3h", false},
				{"東京\uFF0Ejp", "xn--jp-l92cn98g071o", false},
				{"大阪\uFF61jp", "xn--jp-ku9cz72u463f", false},
			},
		},
		{
			name: "Lookup", profile: Lookup,
			subCases: []subCase{
				{"example\u3002jp", "example.jp", false},
				{"東京\uFF0Ejp", "xn--1lqs71d.jp", false},
				{"大阪\uFF61jp", "xn--pssu33l.jp", false},
			},
		},
		{
			name: "Display", profile: Display,
			subCases: []subCase{
				{"example\u3002jp", "example.jp", false},
				{"東京\uFF0Ejp", "xn--1lqs71d.jp", false},
				{"大阪\uFF61jp", "xn--pssu33l.jp", false},
			},
		},
		{
			name: "Registration", profile: Registration,
			subCases: []subCase{
				{"example\u3002jp", "", true},
				{"東京\uFF0Ejp", "", true},
				{"大阪\uFF61jp", "", true},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, c := range tc.subCases {
				gotA, err := tc.profile.ToASCII(c.unicode)
				if c.wantErr {
					if err == nil {
						t.Errorf("ToASCII(%q): got no error, but an error expected", c.unicode)
					}
				} else {
					if err != nil {
						t.Errorf("ToASCII(%q): got err=%v, but no error expected", c.unicode, err)
					} else if gotA != c.wantASCII {
						t.Errorf("ToASCII(%q): got %q, want %q", c.unicode, gotA, c.wantASCII)
					}
				}
			}
		})
	}
}

// asciiOnlyPunycodeHosts are hosts with an "xn--" label whose Punycode payload
// decodes to a non-empty, ASCII-only label. No legitimate encoding produces
// one, and accepting them means ToUnicode/ToASCII silently turn the host into a
// different, purely ASCII host: ToUnicode("xn--example-.com") used to yield
// "example.com" with a nil error. A caller that vets the ASCII form of a host
// before converting it could then be made to admit a name it had rejected.
var asciiOnlyPunycodeHosts = []string{
	"xn--example-.com",       // decodes to "example.com"
	"xn--admin-",             // a single label
	"www.xn--example-.com",   // not the first label
	"xn--example-.xn--test-", // more than one such label
	"xn--a-",                 // shortest payload
	"xn--a-b-",               // payload containing a hyphen
	"xn--EXAMPLE-.com",       // upper case payload
}

func TestASCIIOnlyPunycode(t *testing.T) {
	profiles := []struct {
		name    string
		profile *Profile
	}{
		{"Punycode", Punycode},
		{"Lookup", Lookup},
		{"Display", Display},
		{"Registration", Registration},
	}
	for _, p := range profiles {
		t.Run(p.name, func(t *testing.T) {
			for _, host := range asciiOnlyPunycodeHosts {
				if got, err := p.profile.ToUnicode(host); err == nil {
					t.Errorf("%s.ToUnicode(%q) = %q, <nil>; want an error", p.name, host, got)
				}
				if got, err := p.profile.ToASCII(host); err == nil {
					t.Errorf("%s.ToASCII(%q) = %q, <nil>; want an error", p.name, host, got)
				}
			}
		})
	}

	// The package-level entry points must reject them too.
	t.Run("PackageLevel", func(t *testing.T) {
		for _, host := range asciiOnlyPunycodeHosts {
			if got, err := ToUnicode(host); err == nil {
				t.Errorf("ToUnicode(%q) = %q, <nil>; want an error", host, got)
			}
			if got, err := ToASCII(host); err == nil {
				t.Errorf("ToASCII(%q) = %q, <nil>; want an error", host, got)
			}
		}
	})

	// Genuine A-labels, hosts that merely look like one, and an empty payload
	// (which decodes to the empty string and is not newly rejected) must keep
	// converting without error.
	t.Run("Allowed", func(t *testing.T) {
		for _, host := range []string{
			"example.com",
			"xn--bcher-kva.de",
			"example.xn--p1ai",
			"xn--czrw28b.tw",
			"www.xn--mller-kva.de",
			"foo--xn--bar.org",
		} {
			if _, err := Lookup.ToUnicode(host); err != nil {
				t.Errorf("Lookup.ToUnicode(%q): unexpected error: %v", host, err)
			}
			if _, err := Lookup.ToASCII(host); err != nil {
				t.Errorf("Lookup.ToASCII(%q): unexpected error: %v", host, err)
			}
		}
		if got, err := Punycode.ToUnicode("xn--.com"); err != nil {
			t.Errorf("Punycode.ToUnicode(%q) = %q, %v; want no error", "xn--.com", got, err)
		}
	})
}

// TODO(nigeltao): test errors, once we've specified when ToASCII and ToUnicode
// return errors.
