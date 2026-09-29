package rules

import (
	"regexp"
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// Shared machinery for Symplify's single-task @param/@return/@var doc block
// fixers (AbstractDocBlockFixer): iterate doc/comment tokens carrying a
// param/return/var annotation, transform the content, and drop the whole block
// when it becomes empty.

var docTypeAnnotationRe = regexp.MustCompile(`@(psalm-|phpstan-)?(param|return|var)`)

// applyDocBlockFixer runs process over each doc/comment token that holds a
// @param/@return/@var annotation, mirroring AbstractDocBlockFixer::fix.
func applyDocBlockFixer(s *tokens.Stream, process func(string) string) bool {
	if !dbfCandidateW4(s) {
		return false
	}
	changed := false
	for i := s.Len() - 1; i >= 0; i-- {
		t := s.At(i)
		if t.Kind != token.DocComment && t.Kind != token.Comment {
			continue
		}
		if !docTypeAnnotationRe.MatchString(t.Value) {
			continue
		}
		nc := process(t.Value)
		if nc == t.Value {
			continue
		}
		if isEmptyDocBlock(nc) {
			s.RemoveAt(i)
			if i < s.Len() && s.At(i).Kind == token.Whitespace {
				s.RemoveAt(i)
			}
		} else {
			s.Set(i, token.Token{Kind: token.DocComment, Value: nc})
		}
		changed = true
	}
	return changed
}

// dbfCandidateW4 mirrors AbstractDocBlockFixer::isCandidate.
func dbfCandidateW4(s *tokens.Stream) bool {
	hasComment, hasFuncOrVar := false, false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		switch t.Kind {
		case token.DocComment, token.Comment:
			hasComment = true
		case token.Variable:
			hasFuncOrVar = true
		case token.Keyword:
			if strings.EqualFold(t.Value, "function") {
				hasFuncOrVar = true
			}
			// a "callable(...)" typehint disables the fixer
			if strings.EqualFold(t.Value, "callable") && i+3 < s.Len() && s.At(i+3).Value == ")" {
				return false
			}
		}
	}
	return hasComment && hasFuncOrVar
}

// isEmptyDocBlock reports whether only "/**", "*/", "*" and whitespace remain.
func isEmptyDocBlock(content string) bool {
	r := strings.NewReplacer("/**", "", "*/", "", "*", "")
	stripped := r.Replace(content)
	return strings.TrimSpace(stripped) == ""
}

// --- RemoveDeadParam: drop a @param line that carries only a name, no type ---

var deadParamLineRe = regexp.MustCompile(`(?m)@(?:psalm-|phpstan-)?param\s+\$\w+\s*$`)

type RemoveDeadParam struct{}

func (RemoveDeadParam) Name() string {
	return `Symplify\CodingStandard\Fixer\Commenting\RemoveDeadParamFixer`
}

func (RemoveDeadParam) SourceURL() string {
	return "https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/Commenting/RemoveDeadParamFixer.php"
}

func (RemoveDeadParam) Fix(s *tokens.Stream) bool {
	return applyDocBlockFixer(s, func(content string) string {
		return removeDocLinesMatching(content, deadParamLineRe)
	})
}

// --- RemoveDeadVarThis: drop an inline @var line typing $this ---

var deadVarThisLineRe = regexp.MustCompile(`@(?:psalm-|phpstan-)?var\b[^\n]*\$this\b`)

type RemoveDeadVarThis struct{}

func (RemoveDeadVarThis) Name() string {
	return `Symplify\CodingStandard\Fixer\Commenting\RemoveDeadVarThisFixer`
}

func (RemoveDeadVarThis) SourceURL() string {
	return "https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/Commenting/RemoveDeadVarThisFixer.php"
}

func (RemoveDeadVarThis) Fix(s *tokens.Stream) bool {
	return applyDocBlockFixer(s, func(content string) string {
		return removeDocLinesMatching(content, deadVarThisLineRe)
	})
}

// --- RemoveParamNameReference: drop "&" from a @param variable name ---

var paramNameRefRe = regexp.MustCompile(`(@param.*?)&(\$\w+)`)

type RemoveParamNameReference struct{}

func (RemoveParamNameReference) Name() string {
	return `Symplify\CodingStandard\Fixer\Commenting\RemoveParamNameReferenceFixer`
}

func (RemoveParamNameReference) SourceURL() string {
	return "https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/Commenting/RemoveParamNameReferenceFixer.php"
}

func (RemoveParamNameReference) Fix(s *tokens.Stream) bool {
	return applyDocBlockFixer(s, func(content string) string {
		return paramNameRefRe.ReplaceAllString(content, "$1$2")
	})
}

// --- SwitchedTypeAndName: reorder "@param $name Type" -> "@param Type $name" ---

var switchedTypeNameRe = regexp.MustCompile(`(?m)@((?:psalm-|phpstan-)?(?:param|var))(\s+)(\$\w+)(\s+)((?:[|\\\w\[\]]|<[^<>]*>)+)(\s+\S.*)?$`)
var typeSyntaxRe = regexp.MustCompile(`[\\\[\]<>|]`)

var switchedKnownPrimitiveTypes = map[string]bool{
	"string": true, "int": true, "integer": true, "float": true, "bool": true,
	"boolean": true, "array": true, "object": true, "callable": true, "iterable": true,
	"mixed": true, "void": true, "null": true, "false": true, "true": true, "self": true,
	"static": true, "parent": true, "resource": true, "scalar": true, "never": true,
	"number": true, "double": true,
}

type SwitchedTypeAndName struct{}

func (SwitchedTypeAndName) Name() string {
	return `Symplify\CodingStandard\Fixer\Commenting\SwitchedTypeAndNameFixer`
}

func (SwitchedTypeAndName) SourceURL() string {
	return "https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/Commenting/SwitchedTypeAndNameFixer.php"
}

func (SwitchedTypeAndName) Fix(s *tokens.Stream) bool {
	return applyDocBlockFixer(s, func(content string) string {
		lines := splitDocLines(content)
		changed := false
		for i, l := range lines {
			m := switchedTypeNameRe.FindStringSubmatch(l)
			if m == nil || m[3] == "" || m[5] == "" || !switchedIsKnownType(m[5]) {
				continue
			}
			lines[i] = switchedTypeNameRe.ReplaceAllString(l, "@${1}${2}${5}${4}${3}${6}")
			changed = true
		}
		if !changed {
			return content
		}
		return strings.Join(lines, "")
	})
}

func switchedIsKnownType(t string) bool {
	if typeSyntaxRe.MatchString(t) {
		return true
	}
	return switchedKnownPrimitiveTypes[strings.ToLower(t)]
}

// --- RemovePHPStormAnnotation: drop a "Created by PhpStorm" doc block ---

var createdByPhpStormRe = regexp.MustCompile(`(?is)/\*\*\s+\*\s+Created by PHPStorm(.*?)\*/`)

type RemovePHPStormAnnotation struct{}

func (RemovePHPStormAnnotation) Name() string {
	return `Symplify\CodingStandard\Fixer\Annotation\RemovePHPStormAnnotationFixer`
}

func (RemovePHPStormAnnotation) SourceURL() string {
	return "https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/Annotation/RemovePHPStormAnnotationFixer.php"
}

func (RemovePHPStormAnnotation) Fix(s *tokens.Stream) bool {
	changed := false
	for i := s.Len() - 1; i >= 0; i-- {
		t := s.At(i)
		if t.Kind != token.DocComment && t.Kind != token.Comment {
			continue
		}
		if createdByPhpStormRe.ReplaceAllString(t.Value, "") == "" {
			s.RemoveAt(i)
			changed = true
		}
	}
	return changed
}

// --- shared name resolvers (mirror Symplify Naming resolvers) ---

// methodNameAfter returns the name of the first named function after commentIdx.
func methodNameAfter(s *tokens.Stream, commentIdx int) string {
	for i := commentIdx + 1; i < s.Len(); i++ {
		if s.At(i).Kind == token.Keyword && strings.EqualFold(s.At(i).Value, "function") {
			n := sigNext(s, i)
			if n >= 0 && s.At(n).Kind == token.Ident {
				return s.At(n).Value
			}
		}
	}
	return ""
}

// propertyNameAfter returns the first variable after commentIdx.
func propertyNameAfter(s *tokens.Stream, commentIdx int) string {
	for i := commentIdx + 1; i < s.Len(); i++ {
		if s.At(i).Kind == token.Variable {
			return s.At(i).Value
		}
	}
	return ""
}

var camelBoundaryRe = regexp.MustCompile(`([a-z])([A-Z])`)
var wordRe = regexp.MustCompile(`[a-zA-Z]+`)

// resolveWords splits camelCase, lowercases and extracts word runs.
func resolveWords(value string) []string {
	spaced := camelBoundaryRe.ReplaceAllString(value, "$1 $2")
	return wordRe.FindAllString(strings.ToLower(spaced), -1)
}

// --- RemovePropertyVariableNameDescription: drop trailing "$name" from @var ---

var varTagRe = regexp.MustCompile(`@(?:psalm-|phpstan-)?var`)

type RemovePropertyVariableNameDescription struct{}

func (RemovePropertyVariableNameDescription) Name() string {
	return `Symplify\CodingStandard\Fixer\Annotation\RemovePropertyVariableNameDescriptionFixer`
}

func (RemovePropertyVariableNameDescription) SourceURL() string {
	return "https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/Annotation/RemovePropertyVariableNameDescriptionFixer.php"
}

func (RemovePropertyVariableNameDescription) Fix(s *tokens.Stream) bool {
	if !streamHasVariable(s) || !streamHasComment(s) {
		return false
	}
	changed := false
	for i := s.Len() - 1; i >= 0; i-- {
		t := s.At(i)
		if t.Kind != token.DocComment && t.Kind != token.Comment {
			continue
		}
		propertyName := propertyNameAfter(s, i)
		if propertyName == "" {
			continue
		}
		if len(varTagRe.FindAllString(t.Value, -1)) != 1 {
			continue
		}
		lines := strings.Split(t.Value, "\n")
		lineChanged := false
		suffix := " " + propertyName
		for k, line := range lines {
			if !strings.HasSuffix(line, suffix) || !varTagRe.MatchString(line) {
				continue
			}
			lines[k] = strings.TrimRight(line[:len(line)-len(suffix)], " \t\n\v\f\r")
			lineChanged = true
		}
		if lineChanged {
			s.Set(i, token.Token{Kind: token.DocComment, Value: strings.Join(lines, "\n")})
			changed = true
		}
	}
	return changed
}

// --- RemoveMethodNameDuplicateDescription ---

var articlesRe = regexp.MustCompile(`(?i)\b(?:a|an|the)\b`)
var thirdPersonSRe = regexp.MustCompile(`^([\s*]*\w+?)s\b`)
var spaceNlRe = regexp.MustCompile(`[\s\n]+`)

type RemoveMethodNameDuplicateDescription struct{}

func (RemoveMethodNameDuplicateDescription) Name() string {
	return `Symplify\CodingStandard\Fixer\Annotation\RemoveMethodNameDuplicateDescriptionFixer`
}

func (RemoveMethodNameDuplicateDescription) SourceURL() string {
	return "https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/Annotation/RemoveMethodNameDuplicateDescriptionFixer.php"
}

func (RemoveMethodNameDuplicateDescription) Fix(s *tokens.Stream) bool {
	if !streamHasFunction(s) || !streamHasComment(s) {
		return false
	}
	changed := false
	for i := s.Len() - 1; i >= 0; i-- {
		t := s.At(i)
		if t.Kind != token.DocComment && t.Kind != token.Comment {
			continue
		}
		methodName := methodNameAfter(s, i)
		if methodName == "" {
			continue
		}
		lines := strings.Split(t.Value, "\n")
		var kept []string
		lineChanged := false
		for _, line := range lines {
			l := articlesRe.ReplaceAllString(line, "")
			l = thirdPersonSRe.ReplaceAllString(l, "$1")
			spaceless := spaceNlRe.ReplaceAllString(l, "")
			spaceless = strings.TrimRight(spaceless, ".!")
			if isDuplicateDescription(spaceless, methodName) {
				lineChanged = true
				continue
			}
			kept = append(kept, line)
		}
		if lineChanged {
			s.Set(i, token.Token{Kind: token.DocComment, Value: strings.Join(kept, "\n")})
			changed = true
		}
	}
	return changed
}

func isDuplicateDescription(spaceless, methodName string) bool {
	desc := strings.ToLower(strings.TrimLeft(spaceless, "*"))
	mn := strings.ToLower(methodName)
	if desc == mn {
		return true
	}
	dn := getterSetterNoun(desc)
	return dn != "" && dn == getterSetterNoun(mn)
}

func getterSetterNoun(v string) string {
	for _, p := range []string{"get", "set"} {
		if strings.HasPrefix(v, p) && len(v) > len(p) {
			return v[len(p):]
		}
	}
	return ""
}

// --- RemoveEventSubscriberDescription ---

type RemoveEventSubscriberDescription struct{}

func (RemoveEventSubscriberDescription) Name() string {
	return `Symplify\CodingStandard\Fixer\Annotation\RemoveEventSubscriberDescriptionFixer`
}

func (RemoveEventSubscriberDescription) SourceURL() string {
	return "https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/Annotation/RemoveEventSubscriberDescriptionFixer.php"
}

func (RemoveEventSubscriberDescription) Fix(s *tokens.Stream) bool {
	if !streamHasFunction(s) || !streamHasComment(s) {
		return false
	}
	changed := false
	for i := s.Len() - 1; i >= 0; i-- {
		t := s.At(i)
		if t.Kind != token.DocComment && t.Kind != token.Comment {
			continue
		}
		fnIdx := nextFunctionIndex(s, i)
		if fnIdx < 0 || !isPublicBetween(s, i, fnIdx) {
			continue
		}
		methodName := methodNameAfter(s, i)
		if methodName == "" {
			continue
		}
		lines := strings.Split(t.Value, "\n")
		var kept []string
		lineChanged := false
		for _, line := range lines {
			if isEventDescriptionLine(line, methodName) {
				lineChanged = true
				continue
			}
			kept = append(kept, line)
		}
		if !lineChanged {
			continue
		}
		if isEmptyDocblockLines(kept) {
			s.RemoveAt(i)
		} else {
			s.Set(i, token.Token{Kind: token.DocComment, Value: strings.Join(kept, "\n")})
		}
		changed = true
	}
	return changed
}

func nextFunctionIndex(s *tokens.Stream, commentIdx int) int {
	for i := commentIdx + 1; i < s.Len(); i++ {
		if s.At(i).Kind == token.Keyword && strings.EqualFold(s.At(i).Value, "function") {
			return i
		}
	}
	return -1
}

func isPublicBetween(s *tokens.Stream, commentIdx, fnIdx int) bool {
	for i := commentIdx + 1; i < fnIdx; i++ {
		if s.At(i).Kind == token.Keyword {
			lv := strings.ToLower(s.At(i).Value)
			if lv == "private" || lv == "protected" {
				return false
			}
		}
	}
	return true
}

func isEventDescriptionLine(line, methodName string) bool {
	words := resolveWords(line)
	hasEvent := false
	var descWords []string
	for _, w := range words {
		if w == "event" {
			hasEvent = true
			continue
		}
		descWords = append(descWords, w)
	}
	if !hasEvent || len(descWords) == 0 {
		return false
	}
	var methodWords []string
	for _, w := range resolveWords(methodName) {
		if w != "on" {
			methodWords = append(methodWords, w)
		}
	}
	sortStrings(descWords)
	sortStrings(methodWords)
	return equalStrings(descWords, methodWords)
}

func isEmptyDocblockLines(lines []string) bool {
	joined := strings.Join(lines, "")
	for _, r := range joined {
		if r != '/' && r != '*' && r != ' ' && r != '\t' && r != '\n' && r != '\r' && r != '\v' && r != '\f' {
			return false
		}
	}
	return true
}

func sortStrings(a []string) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j] < a[j-1]; j-- {
			a[j], a[j-1] = a[j-1], a[j]
		}
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func streamHasFunction(s *tokens.Stream) bool {
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind == token.Keyword && strings.EqualFold(s.At(i).Value, "function") {
			return true
		}
	}
	return false
}

func streamHasVariable(s *tokens.Stream) bool {
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind == token.Variable {
			return true
		}
	}
	return false
}

func streamHasComment(s *tokens.Stream) bool {
	for i := 0; i < s.Len(); i++ {
		if k := s.At(i).Kind; k == token.DocComment || k == token.Comment {
			return true
		}
	}
	return false
}

// removeDocLinesMatching removes each doc line whose content matches re,
// mirroring DocBlock getLines()->remove().
func removeDocLinesMatching(content string, re *regexp.Regexp) string {
	lines := splitDocLines(content)
	changed := false
	for i, l := range lines {
		if re.MatchString(l) {
			lines[i] = ""
			changed = true
		}
	}
	if !changed {
		return content
	}
	return strings.Join(lines, "")
}
