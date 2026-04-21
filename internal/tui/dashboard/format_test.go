package dashboard

import (
	"reflect"
	"testing"
)

func TestParseFormatEmpty(t *testing.T) {
	if got := ParseFormat(""); len(got) != 0 {
		t.Errorf("empty string should yield no tokens: %+v", got)
	}
}

func TestParseFormatLiteralOnly(t *testing.T) {
	got := ParseFormat("hello world")
	want := []Token{{Kind: TokLiteral, Literal: "hello world"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v want %+v", got, want)
	}
}

func TestParseFormatSimpleVar(t *testing.T) {
	got := ParseFormat("$label")
	want := []Token{{Kind: TokVar, VarName: "label"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v want %+v", got, want)
	}
}

func TestParseFormatBraceVar(t *testing.T) {
	got := ParseFormat("${label}beta")
	want := []Token{
		{Kind: TokVar, VarName: "label"},
		{Kind: TokLiteral, Literal: "beta"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v want %+v", got, want)
	}
}

func TestParseFormatMixedTokens(t *testing.T) {
	got := ParseFormat("  $label | $state")
	want := []Token{
		{Kind: TokLiteral, Literal: "  "},
		{Kind: TokVar, VarName: "label"},
		{Kind: TokLiteral, Literal: " | "},
		{Kind: TokVar, VarName: "state"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v want %+v", got, want)
	}
}

func TestParseFormatAdjacentVars(t *testing.T) {
	got := ParseFormat("$cursor$label$divider$state")
	want := []Token{
		{Kind: TokVar, VarName: "cursor"},
		{Kind: TokVar, VarName: "label"},
		{Kind: TokVar, VarName: "divider"},
		{Kind: TokVar, VarName: "state"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v want %+v", got, want)
	}
}

func TestParseFormatEscapedDollar(t *testing.T) {
	got := ParseFormat(`price: \$100 for $label`)
	want := []Token{
		{Kind: TokLiteral, Literal: "price: $100 for "},
		{Kind: TokVar, VarName: "label"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v want %+v", got, want)
	}
}

func TestParseFormatLoneDollarIsLiteral(t *testing.T) {
	got := ParseFormat("cost is $ not a var")
	want := []Token{{Kind: TokLiteral, Literal: "cost is $ not a var"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v want %+v", got, want)
	}
}

func TestParseFormatMalformedBraceIsLiteralDollar(t *testing.T) {
	// Unclosed brace: the `$` is kept as a literal dollar so the user
	// can see what they typed.
	got := ParseFormat("${unclosed")
	want := []Token{{Kind: TokLiteral, Literal: "${unclosed"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v want %+v", got, want)
	}
}

func TestParseFormatUnderscoresAndDigitsInName(t *testing.T) {
	got := ParseFormat("$next_run2")
	want := []Token{{Kind: TokVar, VarName: "next_run2"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v want %+v", got, want)
	}
}

func TestRenderFormatSubstitutes(t *testing.T) {
	tokens := ParseFormat("$cursor$label  $state")
	out := RenderFormat(tokens, map[string]string{
		"cursor": "❯ ",
		"label":  "com.x",
		"state":  "running",
	})
	if out != "❯ com.x  running" {
		t.Errorf("unexpected render: %q", out)
	}
}

func TestRenderFormatUnknownVarPassesThrough(t *testing.T) {
	tokens := ParseFormat("$label $nope $state")
	out := RenderFormat(tokens, map[string]string{
		"label": "com.x",
		"state": "loaded",
	})
	if out != "com.x $nope loaded" {
		t.Errorf("unknown var should emit literal $nope: %q", out)
	}
}

func TestRenderFormatRestoresEscape(t *testing.T) {
	tokens := ParseFormat(`\$free and $label`)
	out := RenderFormat(tokens, map[string]string{"label": "x"})
	if out != "$free and x" {
		t.Errorf("escape not rendered correctly: %q", out)
	}
}

func TestIsValidName(t *testing.T) {
	ok := []string{"a", "A", "_", "label", "next_run", "next_run2"}
	bad := []string{"", "1x", "-x", "$x", "label name"}
	for _, s := range ok {
		if !isValidName(s) {
			t.Errorf("%q should be valid", s)
		}
	}
	for _, s := range bad {
		if isValidName(s) {
			t.Errorf("%q should be invalid", s)
		}
	}
}
