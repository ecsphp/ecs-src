package rules

import (
	"fmt"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Classes/ValidClassNameSniff.php
//
// SquizValidClassName reports a class, interface, trait or enum name that is
// not PascalCase.
type SquizValidClassName struct{}

func (SquizValidClassName) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\Classes\ValidClassNameSniff`
}

func (SquizValidClassName) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/Classes/ValidClassNameSniff.php"
}

func (SquizValidClassName) Fix(*tokens.Stream) bool { return false }

func (r SquizValidClassName) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	f := newSlvFile(s.Tokens())
	sc := newPhpcsScope(f)
	var errs []fixer.CodingStandardError
	for i := range f.toks {
		if squizReportOO(f, i) == "" || sc.cond[i] != i || sc.opener[i] < 0 {
			continue
		}
		opener := sc.opener[i]
		start := f.nextEffective(i+1, opener)
		if start < 0 {
			start = opener
		}
		end := -1
		for j := start; j < opener; j++ {
			if f.isEmpty(j) || f.is(j, ":") {
				end = j
				break
			}
		}
		name := f.toks[start].content
		if end >= 0 {
			name = strings.Trim(f.content(start, end-1), " \t\n\r\x00\x0B")
		}
		if squizValidClassNameIsPascalCase(name) {
			continue
		}
		kw := f.toks[i].orig
		kw = strings.ToUpper(kw[:1]) + kw[1:]
		errs = append(errs, squizReportError(r.Name(), "NotPascalCase", f.toks[start].line,
			fmt.Sprintf(`%s name "%s" is not in PascalCase format`, kw, name)))
	}
	return errs
}

// squizValidClassNameIsPascalCase is Common::isCamelCaps($name, true, true, false).
func squizValidClassNameIsPascalCase(name string) bool {
	if name == "" || name[0] < 'A' || name[0] > 'Z' {
		return false
	}
	for _, c := range []byte(name[1:]) {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}
