package pongo2

import (
	"strings"
	"testing"
)

func TestTests(t *testing.T) {
	b := &strings.Builder{}
	tests := []struct {
		Name   string
		Tpl    string
		Ctx    Context
		Expect string
	}{
		{"callable1", `{% if n is callable %}true{% else %}false{% endif %}`, Context{"n": 32}, "false"},
		{"callable2", `{% if n.WriteString is callable %}true{% else %}false{% endif %}`, Context{"n": b}, "true"},

		{"divisibleby1", `{% if n is divisibleby 2 %}true{% else %}false{% endif %}`, Context{"n": 32}, "true"},
		{"divisibleby2", `{% if n is divisibleby 3 %}true{% else %}false{% endif %}`, Context{"n": 32}, "false"},
		{"divisibleby3", `{% if n is divisibleby(2) %}true{% else %}false{% endif %}`, Context{"n": 32}, "true"},
		{"divisibleby4", `{% if n is divisibleby(3) %}true{% else %}false{% endif %}`, Context{"n": 32}, "false"},

		{"notdivisibleby1", `{% if n is not divisibleby 2 %}true{% else %}false{% endif %}`, Context{"n": 32}, "false"},
		{"notdivisibleby2", `{% if n is not divisibleby 3 %}true{% else %}false{% endif %}`, Context{"n": 32}, "true"},
		{"notdivisibleby3", `{% if n is not divisibleby(2) %}true{% else %}false{% endif %}`, Context{"n": 32}, "false"},
		{"notdivisibleby4", `{% if n is not divisibleby(3) %}true{% else %}false{% endif %}`, Context{"n": 32}, "true"},

		{"eq", `{% if n is eq(32) %}true{% else %}false{% endif %}`, Context{"n": 32}, "true"},
		{"==", `{% if n is == 31 %}true{% else %}false{% endif %}`, Context{"n": 32}, "false"},
		{"equalto", `{% if n is equalto(32) %}true{% else %}false{% endif %}`, Context{"n": 32}, "true"},

		{"escaped1", `{% if n is escaped %}true{% else %}false{% endif %}`, Context{"n": 32}, "false"},
		{"escaped2", `{% if n|escape is escaped %}true{% else %}false{% endif %}`, Context{"n": 32}, "true"},

		{"even1", `{% if n is even %}true{% else %}false{% endif %}`, Context{"n": 32}, "true"},
		{"even2", `{% if n is even %}true{% else %}false{% endif %}`, Context{"n": 31}, "false"},

		{"false1", `{% if n is false %}true{% else %}false{% endif %}`, Context{"n": 32}, "false"},
		{"false2", `{% if n is false %}true{% else %}false{% endif %}`, Context{"n": false}, "true"},

		{"falsy1", `{% if n is falsy %}true{% else %}false{% endif %}`, Context{"n": false}, "true"},
		{"falsy2", `{% if n is falsy %}true{% else %}false{% endif %}`, Context{"n": true}, "false"},
		{"falsy3", `{% if n is falsy %}true{% else %}false{% endif %}`, Context{"n": 0}, "true"},
		{"falsy4", `{% if n is falsy %}true{% else %}false{% endif %}`, Context{"n": 1}, "false"},

		{"filter1", `{% if 'lower' is filter %}true{% else %}false{% endif %}`, Context{}, "true"},
		{"filter2", `{% if 'doesnotexist' is filter %}true{% else %}false{% endif %}`, Context{}, "false"},

		{"float1", `{% if n is float %}true{% else %}false{% endif %}`, Context{"n": 32.7}, "true"},
		{"float2", `{% if n is float %}true{% else %}false{% endif %}`, Context{"n": 32.0}, "true"},
		{"float3", `{% if n is float %}true{% else %}false{% endif %}`, Context{"n": "test"}, "false"},
		{"float4", `{% if n is float %}true{% else %}false{% endif %}`, Context{"n": 32}, "false"},

		{"ge1", `{% if n is ge 32 %}true{% else %}false{% endif %}`, Context{"n": 32}, "true"},
		{"ge2", `{% if n is ge 33 %}true{% else %}false{% endif %}`, Context{"n": 32}, "false"},
		{"ge3", `{% if n is ge 31 %}true{% else %}false{% endif %}`, Context{"n": 32}, "true"},
		{">=", `{% if n is >= 32 %}true{% else %}false{% endif %}`, Context{"n": 32}, "true"},

		{"gt1", `{% if n is gt 31 %}true{% else %}false{% endif %}`, Context{"n": 32}, "true"},
		{"gt2", `{% if n is gt 32 %}true{% else %}false{% endif %}`, Context{"n": 32}, "false"},
		{"greaterthan", `{% if n is greaterthan 31 %}true{% else %}false{% endif %}`, Context{"n": 32}, "true"},
		{">", `{% if n is > 31 %}true{% else %}false{% endif %}`, Context{"n": 32}, "true"},

		{"in1", `{% if 'yeah' is in myvar %}true{% else %}false{% endif %}`, Context{"myvar": []string{"okay", "yeah"}}, "true"},
		{"in2", `{% if 'nah' is in myvar %}true{% else %}false{% endif %}`, Context{"myvar": []string{"okay", "yeah"}}, "false"},

		{"integer1", `{% if n is integer %}true{% else %}false{% endif %}`, Context{"n": 32}, "true"},
		{"integer2", `{% if n is integer %}true{% else %}false{% endif %}`, Context{"n": "foo"}, "false"},
		{"integer3", `{% if n is integer %}true{% else %}false{% endif %}`, Context{"n": 32.7}, "false"},

		{"iterable1", `{% if n is iterable %}true{% else %}false{% endif %}`, Context{"n": 32}, "false"},
		{"iterable2", `{% if n is iterable %}true{% else %}false{% endif %}`, Context{"n": "string"}, "true"},
		{"iterable3", `{% if n is iterable %}true{% else %}false{% endif %}`, Context{"n": []string{"okay", "yeah"}}, "true"},

		{"le1", `{% if n is le 32 %}true{% else %}false{% endif %}`, Context{"n": 32}, "true"},
		{"le2", `{% if n is le 31 %}true{% else %}false{% endif %}`, Context{"n": 32}, "false"},
		{"le3", `{% if n is le(31) %}true{% else %}false{% endif %}`, Context{"n": 32}, "false"},
		{"<=", `{% if n is <= 31 %}true{% else %}false{% endif %}`, Context{"n": 32}, "false"},

		{"lower1", `{% if n is lower %}true{% else %}false{% endif %}`, Context{"n": "no caps"}, "true"},
		{"lower2", `{% if n is lower %}true{% else %}false{% endif %}`, Context{"n": "ALL CAPS"}, "false"},
		{"lower3", `{% if n is lower %}true{% else %}false{% endif %}`, Context{"n": "Some caps"}, "false"},

		{"lt1", `{% if n is lt 33 %}true{% else %}false{% endif %}`, Context{"n": 32}, "true"},
		{"lt2", `{% if n is lt 32 %}true{% else %}false{% endif %}`, Context{"n": 32}, "false"},
		{"<", `{% if n is < 33 %}true{% else %}false{% endif %}`, Context{"n": 32}, "true"},
		{"lessthan", `{% if n is lessthan 33 %}true{% else %}false{% endif %}`, Context{"n": 32}, "true"},

		{"mapping1", `{% if n is mapping %}true{% else %}false{% endif %}`, Context{"n": map[string]string{"yeah": "okay"}}, "true"},
		{"mapping2", `{% if n is mapping %}true{% else %}false{% endif %}`, Context{"n": 32}, "false"},

		{"ne1", `{% if n is ne 32 %}true{% else %}false{% endif %}`, Context{"n": 32}, "false"},
		{"ne2", `{% if n is ne 31 %}true{% else %}false{% endif %}`, Context{"n": 32}, "true"},
		{"ne3", `{% if n is ne 31 %}true{% else %}false{% endif %}`, Context{"n": "thirty-two"}, "true"},
		{"!=", `{% if n is != 31 %}true{% else %}false{% endif %}`, Context{"n": 32}, "true"},

		{"none1", `{% if n is none %}true{% else %}false{% endif %}`, Context{"n": nil}, "true"},
		{"none2", `{% if n is none %}true{% else %}false{% endif %}`, Context{"n": 0}, "false"},
		{"none3", `{% if n is none %}true{% else %}false{% endif %}`, Context{"n": "yeah"}, "false"},

		{"number1", `{% if n is number %}true{% else %}false{% endif %}`, Context{"n": 32}, "true"},
		{"number2", `{% if n is number %}true{% else %}false{% endif %}`, Context{"n": 32.7}, "true"},
		{"number3", `{% if n is number %}true{% else %}false{% endif %}`, Context{"n": "test"}, "false"},

		{"odd1", `{% if n is odd %}true{% else %}false{% endif %}`, Context{"n": 32}, "false"},
		{"odd2", `{% if n is odd %}true{% else %}false{% endif %}`, Context{"n": 31}, "true"},

		{"sameas", `{% if n is sameas 32 %}true{% else %}false{% endif %}`, Context{"n": 32}, "true"},

		{"sequence1", `{% if n is sequence %}true{% else %}false{% endif %}`, Context{"n": []string{"yeah", "okay"}}, "true"},
		{"sequence2", `{% if n is sequence %}true{% else %}false{% endif %}`, Context{"n": 32}, "false"},

		{"string1", `{% if n is string %}true{% else %}false{% endif %}`, Context{"n": "test"}, "true"},
		{"string2", `{% if n is string %}true{% else %}false{% endif %}`, Context{"n": "32"}, "true"},
		{"string3", `{% if n is string %}true{% else %}false{% endif %}`, Context{"n": 32}, "false"},
		{"string4", `{% if n is string %}true{% else %}false{% endif %}`, Context{"n": 32.7}, "false"},
		{"string5", `{% if n is string %}true{% else %}false{% endif %}`, Context{"n": []string{"yeah"}}, "false"},

		{"test1", `{% if 'falsy' is test %}true{% else %}false{% endif %}`, Context{}, "true"},
		{"test2", `{% if 'doesnotexist' is test %}true{% else %}false{% endif %}`, Context{}, "false"},

		{"true1", `{% if n is true %}true{% else %}false{% endif %}`, Context{"n": true}, "true"},
		{"true2", `{% if n is true %}true{% else %}false{% endif %}`, Context{"n": 1}, "true"},
		{"true3", `{% if n is true %}true{% else %}false{% endif %}`, Context{"n": false}, "false"},
		{"true4", `{% if n is true %}true{% else %}false{% endif %}`, Context{"n": 0}, "false"},
		{"true5", `{% if n is true %}true{% else %}false{% endif %}`, Context{"n": "true"}, "true"},
		{"true6", `{% if n is true %}true{% else %}false{% endif %}`, Context{"n": "false"}, "true"},

		{"truthy1", `{% if n is truthy %}true{% else %}false{% endif %}`, Context{"n": 1}, "true"},
		{"truthy2", `{% if n is truthy %}true{% else %}false{% endif %}`, Context{"n": false}, "false"},

		{"upper1", `{% if n is upper %}true{% else %}false{% endif %}`, Context{"n": "ALL CAPS"}, "true"},
		{"upper2", `{% if n is upper %}true{% else %}false{% endif %}`, Context{"n": "Some caps"}, "false"},
		{"upper3", `{% if n is upper %}true{% else %}false{% endif %}`, Context{"n": "no caps"}, "false"},

		{"defined1", `{% if n is defined %}true{% else %}false{% endif %}`, Context{"n": 32}, "true"},
		{"defined2", `{% if n is defined %}true{% else %}false{% endif %}`, Context{"n": 0}, "true"},
		{"defined3", `{% if y is defined %}true{% else %}false{% endif %}`, Context{"n": 32}, "false"},

		{"undefined1", `{% if y is undefined %}true{% else %}false{% endif %}`, Context{"n": 32}, "true"},
		{"undefined2", `{% if n is undefined %}true{% else %}false{% endif %}`, Context{"n": 32}, "false"},
		{"undefined3", `{% if n is undefined %}true{% else %}false{% endif %}`, Context{"n": 0}, "false"},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			tpl := getTpl(tt.Tpl)
			s, err := tpl.Execute(tt.Ctx)
			if err != nil {
				t.Errorf("Error: %v\n", err)
				return
			}
			if s != tt.Expect {
				t.Errorf("%s failed:\nwant: %s\n got: %s", tt.Name, tt.Expect, s)
			}
		})
	}

}

func getTpl(s string) *Template {
	ts := NewSet("punga", DefaultLoader)
	ts.autoescape = false
	t, err := ts.FromString(s)
	if err != nil {
		panic(err.Error())
	}
	return t
}
