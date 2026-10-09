package rules

import (
	"regexp"
	"sort"
	"strings"
	"sync"

	"blink/internal/token"
	"blink/internal/tokens"
)

// fpntTypoReCache memoizes the per-typo-name @param patterns, which are
// otherwise recompiled on every fpntFixTypos call. Keyed by typo name.
var fpntTypoReCache sync.Map

func fpntTypoRe(typoName string) *regexp.Regexp {
	if re, ok := fpntTypoReCache.Load(typoName); ok {
		return re.(*regexp.Regexp)
	}
	re := regexp.MustCompile(`@param(.*?)(` + regexp.QuoteMeta(typoName) + `\b)`)
	fpntTypoReCache.Store(typoName, re)
	return re
}

// --- FixParamNameTypo: correct a @param variable name to match the real argument ---

type FixParamNameTypo struct{}

var fpntParamNameRe = regexp.MustCompile(`@param(\s+)(callable)?(.*?)(\$\w+)`)

// fpntParamAnnRe gates a line to a real doc-block @param annotation, mirroring
// PhpCsFixer DocBlock::getAnnotationsOfType (a tag line has `* ` before `@`).
// This stops mid-text "@param" inside // comments from counting as a param.
var fpntParamAnnRe = regexp.MustCompile(`\*\s*@param`)

func (FixParamNameTypo) Name() string {
	return `Symplify\CodingStandard\Fixer\Commenting\FixParamNameTypoFixer`
}

func (FixParamNameTypo) SourceURL() string {
	return "https://github.com/symplify/coding-standard/blob/main/src/Fixer/Commenting/FixParamNameTypoFixer.php"
}

func (FixParamNameTypo) Fix(s *tokens.Stream) bool {
	return docBlockFixerApply(s, func(content string, s *tokens.Stream, i int) string {
		argumentNames := fpntResolveArgNames(s, i)
		if len(argumentNames) == 0 {
			return content
		}
		paramNames := fpntParamNames(content) // map index->name
		missArg := map[int]string{}
		for k, arg := range argumentNames {
			found := -1
			for _, pk := range sortedKeys(paramNames) {
				if paramNames[pk] == arg {
					found = pk
					break
				}
			}
			if found >= 0 {
				delete(paramNames, found)
			} else {
				missArg[k] = arg
			}
		}
		if len(missArg) == 0 || len(paramNames) == 0 {
			return content
		}
		return fpntFixTypos(argumentNames, missArg, paramNames, content)
	})
}

// fpntResolveArgNames mirrors DocblockRelatedParamNamesResolver: the parameter
// variable names of the next function/fn after the doc block.
func fpntResolveArgNames(s *tokens.Stream, docPos int) []string {
	fn := -1
	for j := docPos + 1; j < s.Len(); j++ {
		if s.At(j).Kind == token.Keyword {
			lv := strings.ToLower(s.At(j).Value)
			if lv == "function" || lv == "fn" {
				fn = j
				break
			}
		}
	}
	if fn < 0 {
		return nil
	}
	open := nextPunctOfKind(s, fn, "(")
	if open < 0 {
		return nil
	}
	closeIdx := s.MatchForward(open)
	if closeIdx < 0 {
		return nil
	}
	var names []string
	depth := 0
	for i := open + 1; i < closeIdx; i++ {
		t := s.At(i)
		if t.Kind == token.Punct {
			switch t.Value {
			case "(", "[", "{":
				depth++
			case ")", "]", "}":
				depth--
			}
			continue
		}
		if depth == 0 && t.Kind == token.Variable {
			names = append(names, t.Value)
		}
	}
	return names
}

func fpntParamNames(content string) map[int]string {
	out := map[int]string{}
	idx := 0
	for line := range strings.SplitSeq(content, "\n") {
		if !fpntParamAnnRe.MatchString(line) {
			continue
		}
		m := fpntParamNameRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if m[2] == "callable" {
			continue
		}
		out[idx] = m[4]
		idx++
	}
	return out
}

func fpntFixTypos(argumentNames []string, missArg, paramNames map[int]string, content string) string {
	replaced := map[string]bool{}
	for _, a := range argumentNames {
		replaced[a] = false
	}
	for _, k := range sortedKeys(missArg) {
		argName := missArg[k]
		typoName, ok := paramNames[k]
		if !ok {
			continue
		}
		re := fpntTypoRe(typoName)
		content = replaceFirstFunc(content, re, func(groups []string) string {
			paramName := groups[2]
			if done, exists := replaced[paramName]; exists && !done {
				replaced[paramName] = true
				return groups[0]
			}
			return "@param" + groups[1] + argName
		})
	}
	return content
}

// replaceFirstFunc replaces every match (like preg_replace_callback) using the
// submatch groups.
func replaceFirstFunc(content string, re *regexp.Regexp, fn func(groups []string) string) string {
	var b strings.Builder
	last := 0
	for _, loc := range re.FindAllStringSubmatchIndex(content, -1) {
		groups := make([]string, len(loc)/2)
		for g := 0; g < len(loc)/2; g++ {
			if loc[2*g] >= 0 {
				groups[g] = content[loc[2*g]:loc[2*g+1]]
			}
		}
		b.WriteString(content[last:loc[0]])
		b.WriteString(fn(groups))
		last = loc[1]
	}
	b.WriteString(content[last:])
	return b.String()
}

func sortedKeys(m map[int]string) []int {
	ks := make([]int, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	return ks
}
