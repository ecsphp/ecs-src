package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/DateTimeCreateFromFormatCallFixer.php
//
// DateTimeCreateFromFormatCall prefixes the format argument of a
// DateTime(Immutable)::createFromFormat() call with "!" so the returned value's
// time section is zeroed. It only touches the global DateTime/DateTimeImmutable
// classes (leading "\" or global namespace with no shadowing import) whose
// format argument is a single string literal. Risky: the timing may be relied on.
type DateTimeCreateFromFormatCall struct{}

func (DateTimeCreateFromFormatCall) Name() string {
	return `PhpCsFixer\Fixer\FunctionNotation\DateTimeCreateFromFormatCallFixer`
}

func (DateTimeCreateFromFormatCall) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/FunctionNotation/DateTimeCreateFromFormatCallFixer.php"
}

func (DateTimeCreateFromFormatCall) Fix(s *tokens.Stream) bool {
	changed := false
	imports := aliasClassImports(s)
	for _, ns := range aliasNamespaceDeclarations(s) {
		for index := ns.scopeEnd; index > ns.scopeStart; index-- {
			if index >= s.Len() || !isPunctVal(s, index, "::") {
				continue
			}
			fnName := aliasNextMeaningful(s, index)
			if fnName < 0 || s.At(fnName).Kind != token.Ident || !strings.EqualFold(s.At(fnName).Value, "createFromFormat") {
				continue
			}
			if !isPunctVal(s, aliasNextMeaningful(s, fnName), "(") {
				continue
			}
			classIdx := aliasPrevMeaningful(s, index)
			if classIdx < 0 || s.At(classIdx).Kind != token.Ident ||
				(!strings.EqualFold(s.At(classIdx).Value, "DateTime") && !strings.EqualFold(s.At(classIdx).Value, "DateTimeImmutable")) {
				continue
			}
			preClass := aliasPrevMeaningful(s, classIdx)
			if isPunctVal(s, preClass, `\`) {
				if before := aliasPrevMeaningful(s, preClass); before >= 0 && s.At(before).Kind == token.Ident {
					continue // qualified name like A\DateTime
				}
			} else if !ns.global {
				continue
			} else if dateTimeShadowed(imports, ns) {
				continue
			}

			open := aliasNextMeaningful(s, fnName)
			closeIdx := s.MatchForward(open)
			if closeIdx < 0 {
				continue
			}
			argIdx := dateTimeFormatArgument(s, aliasArguments(s, open, closeIdx))
			if argIdx < 0 {
				continue
			}
			format := s.At(argIdx).Value
			if len(format) < 3 {
				continue
			}
			offset := 1
			if format[0] == 'b' || format[0] == 'B' {
				offset = 2
			}
			if format[offset] == '!' {
				continue
			}
			s.SetValue(argIdx, format[:offset]+"!"+format[offset:])
			changed = true
		}
	}
	return changed
}

// dateTimeShadowed reports whether a class import in the namespace imports a
// different class under the short name DateTime or DateTimeImmutable.
func dateTimeShadowed(imports []aliasFunctionImport, ns aliasNamespace) bool {
	for _, imp := range imports {
		if imp.start < ns.scopeStart || imp.end > ns.scopeEnd {
			continue
		}
		short := strings.ToLower(imp.shortName)
		if (short == "datetime" || short == "datetimeimmutable") && short != strings.ToLower(strings.TrimLeft(imp.fullName, `\`)) {
			return true
		}
	}
	return false
}

// dateTimeFormatArgument ports getFirstArgumentTokenIndex: the call must take
// exactly two arguments and the first must be a single string literal, whose
// index is returned (or -1).
func dateTimeFormatArgument(s *tokens.Stream, args [][2]int) int {
	if len(args) != 2 {
		return -1
	}
	start := aliasNextMeaningful(s, args[0][0]-1)
	rawEnd := args[0][1]
	if start < 0 {
		return -1
	}
	if next := aliasNextMeaningful(s, start); start != rawEnd && next >= 0 && next <= rawEnd {
		return -1 // more than one meaningful token: not a simple single string
	}
	if !aliasIsConstantString(s.At(start)) {
		return -1
	}
	return start
}

// aliasClassImports ports the single (non-group, non-multi) class "use" imports
// NamespaceUsesAnalyzer returns: those that are neither "use function" nor
// "use const".
func aliasClassImports(s *tokens.Stream) []aliasFunctionImport {
	var out []aliasFunctionImport
	for _, use := range aliasImportUseIndexes(s) {
		next := aliasNextMeaningful(s, use)
		if aliasIsKeyword(s, next, "function") || aliasIsKeyword(s, next, "const") {
			continue
		}
		end := -1
		for j := use + 1; j < s.Len(); j++ {
			if isPunctVal(s, j, ";") || s.At(j).Kind == token.CloseTag {
				end = j
				break
			}
		}
		if end < 0 {
			continue
		}
		multi := false
		for j := use + 1; j < end; j++ {
			if isPunctVal(s, j, ",") || isPunctVal(s, j, "{") {
				multi = true
				break
			}
		}
		if multi {
			continue
		}
		var full strings.Builder
		short := ""
		aliased := false
		for j := next; j >= 0 && j < end; j = aliasNextMeaningful(s, j) {
			t := s.At(j)
			switch {
			case kwIs(t, "as"):
				aliased = true
			case t.Kind == token.Ident || t.Kind == token.Keyword:
				short = t.Value
				if !aliased {
					full.WriteString(t.Value)
				}
			case t.Kind == token.Punct && t.Value == `\`:
				full.WriteString(t.Value)
			}
		}
		if short == "" {
			continue
		}
		out = append(out, aliasFunctionImport{fullName: full.String(), shortName: short, start: use, end: end})
	}
	return out
}
