package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// Shared PHPUnit-test-class detection, mirroring PhpUnitTestCaseAnalyzer:
// a class that `extends` something and whose name (or an extended/implemented
// name) looks like a PHPUnit test case.

// phpUnitTestClasses returns [openBrace, closeBrace] index pairs for every
// PHPUnit test class body in the stream.
func phpUnitTestClasses(s *tokens.Stream) [][2]int {
	var out [][2]int
	for i := 0; i < s.Len(); i++ {
		if !kwIs(s.At(i), "class") {
			continue
		}
		if !isPhpUnitClass(s, i) {
			continue
		}
		open := nextPunctOfKind(s, i, "{")
		if open < 0 {
			continue
		}
		end := s.MatchForward(open)
		if end < 0 {
			continue
		}
		out = append(out, [2]int{open, end})
	}
	return out
}

func isPhpUnitClass(s *tokens.Stream, index int) bool {
	// must have an `extends` before the class body
	extends := -1
	for j := index + 1; j < s.Len(); j++ {
		if isPunctVal(s, j, "{") {
			break
		}
		if kwIs(s.At(j), "extends") {
			extends = j
			break
		}
	}
	if extends < 0 {
		return false
	}
	// class name
	if name := sigNext(s, index); name >= 0 && s.At(name).Kind == token.Ident && endsWithTestName(s.At(name).Value, false) {
		return true
	}
	// any name in extends/implements clause
	for j := index + 1; j < s.Len(); j++ {
		if isPunctVal(s, j, "{") {
			break
		}
		if s.At(j).Kind == token.Ident && endsWithTestName(s.At(j).Value, true) {
			return true
		}
	}
	return false
}

// endsWithTestName mirrors /(?:Test|TestCase)$/ (name) and, with iface,
// /(?:Test|TestCase)(?:Interface)?$/ (extended/implemented names).
func endsWithTestName(name string, iface bool) bool {
	if strings.HasSuffix(name, "Test") || strings.HasSuffix(name, "TestCase") {
		return true
	}
	if iface && (strings.HasSuffix(name, "TestInterface") || strings.HasSuffix(name, "TestCaseInterface")) {
		return true
	}
	return false
}

// puIsMethod reports whether index is a real (non-lambda) method: a `function`
// keyword directly followed by a name.
func puIsMethod(s *tokens.Stream, index int) bool {
	if !kwIs(s.At(index), "function") {
		return false
	}
	n := sigNext(s, index)
	if n < 0 {
		return false
	}
	// a name means it's a method; "(" or "&" means a closure
	return s.At(n).Kind == token.Ident
}

var puModifierKw = map[string]bool{
	"public": true, "protected": true, "private": true,
	"final": true, "abstract": true, "readonly": true, "static": true,
}

// puDocBlockIndex mirrors DocBlockAnnotationTrait::getDocBlockIndex: walk back
// over whitespace, modifiers, comments and folded attributes to the token that
// may be the method's doc comment.
func puDocBlockIndex(s *tokens.Stream, index int) int {
	idx := index
	for {
		p := getPrevNonWhitespace(s, idx)
		if p < 0 {
			return idx
		}
		t := s.At(p)
		if t.Kind == token.Comment { // includes folded #[...] attributes
			idx = p
			continue
		}
		if t.Kind == token.Keyword && puModifierKw[strings.ToLower(t.Value)] {
			idx = p
			continue
		}
		return p
	}
}

// camelCaseToUnderscore mirrors Utils::camelCaseToUnderscore for ASCII names
// (PHPUnit method names): insert "_" at case boundaries, then lowercase.
func camelCaseToUnderscore(str string) string {
	b := []byte(str)
	isUpper := func(c byte) bool { return c >= 'A' && c <= 'Z' }
	var out []byte
	for i := range b {
		c := b[i]
		if i > 0 && b[i-1] != '_' {
			var next byte
			hasNext := i+1 < len(b)
			if hasNext {
				next = b[i+1]
			}
			condA := isUpper(c) && hasNext && !isUpper(next)
			condB := !isUpper(b[i-1]) && isUpper(c)
			if condA || condB {
				out = append(out, '_')
			}
		}
		out = append(out, c)
	}
	return strings.ToLower(string(out))
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitMethodCasingFixer.php
//
// PhpUnitMethodCasing enforces snake_case (the ECS laravel config) for PHPUnit
// test method names.
type PhpUnitMethodCasing struct{}

func (PhpUnitMethodCasing) Name() string {
	return `PhpCsFixer\Fixer\PhpUnit\PhpUnitMethodCasingFixer`
}

func (PhpUnitMethodCasing) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitMethodCasingFixer.php"
}

func (PhpUnitMethodCasing) Fix(s *tokens.Stream) bool {
	changed := false
	for _, cls := range phpUnitTestClasses(s) {
		if methodCasingClass(s, cls[0], cls[1]) {
			changed = true
		}
	}
	return changed
}

func methodCasingClass(s *tokens.Stream, start, end int) bool {
	changed := false
	existing := map[string]bool{}
	for i := end - 1; i > start; i-- {
		if !puIsMethod(s, i) {
			continue
		}
		if n := sigNext(s, i); n >= 0 {
			existing[strings.ToLower(s.At(n).Value)] = true
		}
	}
	for i := end - 1; i > start; i-- {
		if !isTestMethod(s, i) {
			continue
		}
		nameIdx := sigNext(s, i)
		if nameIdx < 0 {
			continue
		}
		name := s.At(nameIdx).Value
		nameLower := strings.ToLower(name)
		newName := camelCaseToUnderscore(name)
		newLower := strings.ToLower(newName)
		if existing[newLower] && nameLower != newLower {
			continue
		}
		existing[newLower] = true
		if newName != name {
			s.SetValue(nameIdx, newName)
			changed = true
		}
		docIdx := puDocBlockIndex(s, i)
		if docIdx >= 0 && s.At(docIdx).Kind == token.DocComment {
			if updateDependsDoc(s, docIdx) {
				changed = true
			}
		}
	}
	return changed
}

func isTestMethod(s *tokens.Stream, index int) bool {
	if !puIsMethod(s, index) {
		return false
	}
	nameIdx := sigNext(s, index)
	if nameIdx < 0 {
		return false
	}
	if strings.HasPrefix(s.At(nameIdx).Value, "test") {
		return true
	}
	if isTestAttributePresent(s, index) {
		return true
	}
	docIdx := puDocBlockIndex(s, index)
	return docIdx >= 0 && s.At(docIdx).Kind == token.DocComment && strings.Contains(s.At(docIdx).Value, "@test")
}

// isTestAttributePresent reports whether the method carries a #[Test] attribute,
// which the lexer folds into a single comment token starting with "#[".
func isTestAttributePresent(s *tokens.Stream, index int) bool {
	p := getPrevNonWhitespace(s, index)
	for p >= 0 {
		t := s.At(p)
		if t.Kind == token.Comment {
			if strings.HasPrefix(t.Value, "#[") && phpUnitTestAttrRe(t.Value) {
				return true
			}
			p = getPrevNonWhitespace(s, p)
			continue
		}
		if t.Kind == token.Keyword && puModifierKw[strings.ToLower(t.Value)] {
			p = getPrevNonWhitespace(s, p)
			continue
		}
		break
	}
	return false
}

// phpUnitTestAttrRe reports whether a folded attribute comment names the PHPUnit
// Test attribute (short "Test" or fully-qualified).
func phpUnitTestAttrRe(v string) bool {
	inner := strings.TrimSuffix(strings.TrimPrefix(v, "#["), "]")
	for part := range strings.SplitSeq(inner, ",") {
		part = strings.TrimSpace(part)
		if i := strings.IndexByte(part, '('); i >= 0 {
			part = part[:i]
		}
		part = strings.TrimSpace(part)
		short := part
		if i := strings.LastIndex(part, `\`); i >= 0 {
			short = part[i+1:]
		}
		if short == "Test" {
			return true
		}
	}
	return false
}

// updateDependsDoc rewrites @depends method references to the new casing.
func updateDependsDoc(s *tokens.Stream, docIdx int) bool {
	content := s.At(docIdx).Value
	if !strings.Contains(content, "@depends") {
		return false
	}
	changed := false
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		idx := strings.Index(line, "@depends")
		if idx < 0 {
			continue
		}
		rest := line[idx+len("@depends"):]
		// skip whitespace
		j := 0
		for j < len(rest) && (rest[j] == ' ' || rest[j] == '\t') {
			j++
		}
		ws := rest[:j]
		// capture the reference up to whitespace/end
		k := j
		for k < len(rest) && rest[k] != ' ' && rest[k] != '\t' && rest[k] != '\r' && rest[k] != '\n' {
			k++
		}
		ref := rest[j:k]
		if ref == "" {
			continue
		}
		newRef := camelCaseToUnderscore(ref)
		if newRef != ref {
			lines[i] = line[:idx+len("@depends")] + ws + newRef + rest[k:]
			changed = true
		}
	}
	if changed {
		s.SetValue(docIdx, strings.Join(lines, "\n"))
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitSetUpTearDownVisibilityFixer.php
//
// PhpUnitSetUpTearDownVisibility makes setUp()/tearDown() protected.
type PhpUnitSetUpTearDownVisibility struct{}

func (PhpUnitSetUpTearDownVisibility) Name() string {
	return `PhpCsFixer\Fixer\PhpUnit\PhpUnitSetUpTearDownVisibilityFixer`
}

func (PhpUnitSetUpTearDownVisibility) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitSetUpTearDownVisibilityFixer.php"
}

func (PhpUnitSetUpTearDownVisibility) Fix(s *tokens.Stream) bool {
	changed := false
	for _, cls := range phpUnitTestClasses(s) {
		if setUpTearDownClass(s, cls[0], cls[1]) {
			changed = true
		}
	}
	return changed
}

func setUpTearDownClass(s *tokens.Stream, start, end int) bool {
	changed := false
	counter := 0
	for i := start + 1; i < end && i < s.Len(); i++ {
		if counter == 2 {
			break
		}
		if isPunctVal(s, i, "{") {
			if e := s.MatchForward(i); e > i {
				i = e
			}
			continue
		}
		if !kwIs(s.At(i), "function") {
			continue
		}
		nameIdx := sigNext(s, i)
		if nameIdx < 0 {
			continue
		}
		fn := strings.ToLower(s.At(nameIdx).Value)
		if fn != "setup" && fn != "teardown" {
			continue
		}
		counter++
		// find the visibility modifier of this method, if any
		visIdx := methodVisibilityIndex(s, i)
		if visIdx >= 0 {
			if strings.EqualFold(s.At(visIdx).Value, "public") {
				s.Set(visIdx, token.Token{Kind: token.Keyword, Value: "protected"})
				changed = true
			}
			continue
		}
		// no visibility: insert "protected " before the function keyword
		s.InsertAt(i, token.Token{Kind: token.Whitespace, Value: " "})
		s.InsertAt(i, token.Token{Kind: token.Keyword, Value: "protected"})
		changed = true
		end += 2
	}
	return changed
}

// methodVisibilityIndex returns the index of the method's visibility keyword
// (public/protected/private) in its modifier run, or -1 if none.
func methodVisibilityIndex(s *tokens.Stream, fnIdx int) int {
	p := getPrevNonWhitespace(s, fnIdx)
	for p >= 0 {
		t := s.At(p)
		if t.Kind == token.Keyword {
			switch strings.ToLower(t.Value) {
			case "public", "protected", "private":
				return p
			case "final", "abstract", "readonly", "static":
				p = getPrevNonWhitespace(s, p)
				continue
			}
		}
		if t.Kind == token.Comment {
			p = getPrevNonWhitespace(s, p)
			continue
		}
		break
	}
	return -1
}
