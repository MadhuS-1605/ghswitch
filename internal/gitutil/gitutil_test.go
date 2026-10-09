package gitutil

import "testing"

func TestParseRemote(t *testing.T) {
	cases := map[string][2]string{
		"git@github.com:MadhuS-1605/ghswitch.git":       {"github.com", "MadhuS-1605"},
		"https://github.com/MadhuS-1605/ghswitch.git":   {"github.com", "MadhuS-1605"},
		"https://user@ghe.corp/team/repo":               {"ghe.corp", "team"},
		"ssh://git@github.com/MadhuS-1605/ghswitch.git": {"github.com", "MadhuS-1605"},
		"": {"", ""},
	}
	for url, want := range cases {
		if h, o := ParseRemote(url); h != want[0] || o != want[1] {
			t.Errorf("ParseRemote(%q) = %q, %q; want %v", url, h, o, want)
		}
	}
}
