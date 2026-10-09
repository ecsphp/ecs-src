package token

// Kind is the lexical category of a token.
type Kind int

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
	String // '...' "..."
	Punct  // operators, braces, ; , etc.
	EOF
)

var kindNames = [...]string{
	Unknown:    "Unknown",
	OpenTag:    "OpenTag",
	CloseTag:   "CloseTag",
	InlineHTML: "InlineHTML",
	Whitespace: "Whitespace",
	Comment:    "Comment",
	DocComment: "DocComment",
	Variable:   "Variable",
	Ident:      "Ident",
	Keyword:    "Keyword",
	Number:     "Number",
	String:     "String",
	Punct:      "Punct",
	EOF:        "EOF",
}

func (k Kind) String() string {
	if int(k) < 0 || int(k) >= len(kindNames) {
		return "Kind(?)"
	}
	return kindNames[k]
}

// Token is a single lexical unit. The stream is lossless: concatenating every
// Value in order reproduces the original source byte for byte.
type Token struct {
	Kind  Kind
	Value string
	Pos   int // byte offset in source
}
