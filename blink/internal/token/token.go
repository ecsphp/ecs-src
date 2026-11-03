package token

import "strings"

// Kind is the lexical category of a token.
type Kind uint8

const (
	Unknown    Kind = iota
	OpenTag         // <?php <?= <?
	CloseTag        // ?>
	InlineHTML      // text outside PHP tags
	Whitespace      // spaces, tabs, newlines
	Comment         // // # /* */
	DocComment      // /** */
	Variable        // $foo
	Ident           // names (T_STRING): function/class names, constants, types
	Keyword         // reserved words: function, class, return, namespace, ...
	Number
	String         // '...' "..."
	Punct          // operators, braces, ; , etc.
	AttributeStart // "#[" opening an attribute; its contents lex as ordinary tokens
	AttributeEnd   // the "]" closing an attribute (retagged so depth scans skip it)
	EOF
)

var kindNames = [...]string{
	Unknown:        "Unknown",
	OpenTag:        "OpenTag",
	CloseTag:       "CloseTag",
	InlineHTML:     "InlineHTML",
	Whitespace:     "Whitespace",
	Comment:        "Comment",
	DocComment:     "DocComment",
	Variable:       "Variable",
	Ident:          "Ident",
	Keyword:        "Keyword",
	Number:         "Number",
	String:         "String",
	Punct:          "Punct",
	AttributeStart: "AttributeStart",
	AttributeEnd:   "AttributeEnd",
	EOF:            "EOF",
}

func (k Kind) String() string {
	if int(k) < 0 || int(k) >= len(kindNames) {
		return "Kind(?)"
	}
	return kindNames[k]
}

// SplitBinaryPrefix splits the binary-string prefix "b" or "B" off the value of
// a string token ("b'x'", "B\"x\"", "b<<<EOT"); prefix is "" when there is none.
func SplitBinaryPrefix(v string) (prefix, rest string) {
	if len(v) > 1 && (v[0] == 'b' || v[0] == 'B') && (v[1] == '\'' || v[1] == '"' || (len(v) > 3 && v[1:4] == "<<<")) {
		return v[:1], v[1:]
	}
	return "", v
}

// Token is a single lexical unit. The stream is lossless: concatenating every
// Value in order reproduces the original source byte for byte.
type Token struct {
	Kind Kind
	// Part tells which piece of a string literal a String token is: an
	// interpolated string, heredoc or backtick literal is split like
	// token_get_all splits it, with its code lexed as ordinary tokens.
	Part  Part
	Value string
	Pos   int // byte offset in source
}

// Part is the place of a String token in its string literal.
type Part uint8

const (
	Whole Part = iota // the whole literal: nothing in it lexed as code
	Head              // opens an interpolated literal: "..., `..., <<<EOT\n... (up to "{", "${" or "$")
	Mid               // literal text between two interpolated parts
	Tail              // closes an interpolated literal: ...", ...` or ...\nEOT
)

// IsLiteral reports whether t is a whole string literal (one token, no code in it).
func (t Token) IsLiteral() bool { return t.Kind == String && t.Part == Whole }

// OpensString reports whether t is a whole string literal or the Head of one.
func (t Token) OpensString() bool { return t.Kind == String && (t.Part == Whole || t.Part == Head) }

// ClosesString reports whether t is a whole string literal or the Tail of one.
func (t Token) ClosesString() bool { return t.Kind == String && (t.Part == Whole || t.Part == Tail) }

// IsShellExec reports whether t is the whole, Head or Tail String token of a
// backtick (shell-exec) literal, which is never a constant string for PHP.
func (t Token) IsShellExec() bool {
	if t.Kind != String || t.Value == "" {
		return false
	}
	switch t.Part {
	case Whole, Head:
		return t.Value[0] == '`'
	case Tail:
		return t.Value[len(t.Value)-1] == '`'
	}
	return false
}

// StartsHeredoc reports whether t opens a heredoc or nowdoc (T_START_HEREDOC).
func (t Token) StartsHeredoc() bool {
	_, v := SplitBinaryPrefix(t.Value)
	return t.OpensString() && strings.HasPrefix(v, "<<<")
}

// EndsHeredoc reports whether t closes a heredoc or nowdoc (T_END_HEREDOC).
func (t Token) EndsHeredoc() bool {
	switch {
	case t.Kind != String || t.Value == "":
		return false
	case t.Part == Whole:
		return t.StartsHeredoc()
	case t.Part == Tail:
		last := t.Value[len(t.Value)-1]
		return last != '"' && last != '`'
	}
	return false
}
