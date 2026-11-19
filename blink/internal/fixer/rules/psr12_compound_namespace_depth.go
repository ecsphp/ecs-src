package rules

import (
	"strconv"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/Namespaces/CompoundNamespaceDepthSniff.php
//
// PSR12CompoundNamespaceDepth reports a name in a group use statement nested
// deeper than MaxDepth (default 2). It fixes nothing.
type PSR12CompoundNamespaceDepth struct {
	MaxDepth int
}

func (PSR12CompoundNamespaceDepth) Name() string {
	return `PHP_CodeSniffer\Standards\PSR12\Sniffs\Namespaces\CompoundNamespaceDepthSniff`
}

func (PSR12CompoundNamespaceDepth) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/PSR12/Sniffs/Namespaces/CompoundNamespaceDepthSniff.php"
}

func (r PSR12CompoundNamespaceDepth) WithConfig(config map[string]any) fixer.Fixer {
	r.MaxDepth = configInt(config, "maxDepth", r.MaxDepth)
	return r
}

func (PSR12CompoundNamespaceDepth) Fix(*tokens.Stream) bool { return false }

func (r PSR12CompoundNamespaceDepth) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	f := newSlvFile(lexer.Lex(s.Render()))
	sc := newPhpcsScope(f)
	var errs []fixer.CodingStandardError
	for open := range f.toks {
		if sc.code[open] != "open_use_group" {
			continue
		}
		end := -1
		for j := open + 1; j < len(f.toks); j++ {
			if sc.code[j] == "close_use_group" {
				end = j
				break
			}
		}
		if end < 0 {
			continue
		}
		depth := 1
		for i := open + 1; i <= end; i++ {
			t := f.toks[i]
			switch {
			case t.kind == token.Punct && t.orig == `\`:
				depth++
				continue
			case t.kind == token.Ident && !strings.HasPrefix(strings.ToLower(t.orig), `namespace\`):
				// T_NAME_FULLY_QUALIFIED and T_NAME_QUALIFIED
				depth += strings.Count(t.orig, `\`)
				continue
			}
			if i == end || f.is(i, ",") {
				if depth > r.MaxDepth {
					errs = append(errs, fixer.CodingStandardError{
						Line:    t.line,
						Message: "Compound namespaces cannot have a depth more than " + strconv.Itoa(r.MaxDepth),
						Checker: r.Name() + ".TooDeep",
					})
				}
				depth = 1
			}
		}
	}
	return errs
}
