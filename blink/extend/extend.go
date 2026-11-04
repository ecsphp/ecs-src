// Package extend builds a blink binary with your own fixers:
//
//	func main() {
//		extend.Register(MyFixer{}, 0)
//		extend.Main()
//	}
//
// A fixer's Name is the PHP class the ECS config lists. Fix runs on many files
// at once, so it must not keep state.
package extend

import (
	"github.com/ecsphp/ecs-src/blink/internal/cli"
	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/fixer/rules"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

type (
	Fixer             = fixer.Fixer
	ConfigurableFixer = fixer.ConfigurableFixer
	Stream            = tokens.Stream
	Token             = token.Token
	Kind              = token.Kind
)

// The token kinds.
const (
	Unknown        = token.Unknown
	OpenTag        = token.OpenTag
	CloseTag       = token.CloseTag
	InlineHTML     = token.InlineHTML
	Whitespace     = token.Whitespace
	Comment        = token.Comment
	DocComment     = token.DocComment
	Variable       = token.Variable
	Ident          = token.Ident
	Keyword        = token.Keyword
	Number         = token.Number
	String         = token.String
	Punct          = token.Punct
	AttributeStart = token.AttributeStart
	AttributeEnd   = token.AttributeEnd
	EOF            = token.EOF
)

// Register adds f, run at the PHP fixer's priority (higher runs first). Call
// it before Main: it panics later, or when the name is taken.
func Register(f Fixer, priority int) { rules.Register(f, priority) }

func Main() { cli.Main() }

// Lex turns PHP source into a Stream, to test a fixer.
func Lex(src string) *Stream { return tokens.New(lexer.Lex(src)) }

// NextMeaningful and PrevMeaningful skip whitespace and comments; they return -1
// when there is no such token.
func NextMeaningful(s *Stream, i int) int { return rules.NextMeaningful(s, i) }
func PrevMeaningful(s *Stream, i int) int { return rules.PrevMeaningful(s, i) }

// blink lexes every ":" and "?" alike. These tell a return or enum type ":"
// (CT::T_TYPE_COLON) and a nullable "?" (CT::T_NULLABLE_TYPE) apart.
func IsTypeColon(s *Stream, i int) bool    { return rules.IsTypeColon(s, i) }
func IsNullableType(s *Stream, i int) bool { return rules.IsNullableType(s, i) }
