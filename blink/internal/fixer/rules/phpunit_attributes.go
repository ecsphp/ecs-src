package rules

import (
	"regexp"
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitAttributesFixer.php
//
// PhpUnitAttributes converts PHPUnit PHPDoc annotations on test classes and
// methods into their PHP 8 attribute form (fully qualified, e.g.
// #[\PHPUnit\Framework\Attributes\CoversClass(Foo::class)]), removing the
// annotation by default. Inserted attributes always use the fully-qualified
// PHPUnit\Framework\Attributes namespace, mirroring upstream.
type PhpUnitAttributes struct {
	keepAnnotations bool
}

func (PhpUnitAttributes) Name() string {
	return `PhpCsFixer\Fixer\PhpUnit\PhpUnitAttributesFixer`
}

func (PhpUnitAttributes) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitAttributesFixer.php"
}

func (f PhpUnitAttributes) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["keep_annotations"].(bool); ok {
		f.keepAnnotations = v
	}
	return f
}

func (f PhpUnitAttributes) Fix(s *tokens.Stream) bool {
	classes := puClassesLastFirst(s)
	if len(classes) == 0 {
		return false
	}
	changed := false
	for _, cls := range classes {
		if f.applyClass(s, cls[0], cls[1]) {
			changed = true
		}
	}
	return changed
}

// applyClass mirrors applyPhpUnitClassFix: scan every doc comment from the class
// close brace down to the class doc comment (if any), inserting attributes.
func (f PhpUnitAttributes) applyClass(s *tokens.Stream, open, close int) bool {
	startScan := open
	if classKw := puaPrevClassKeyword(s, open); classKw >= 0 {
		if d := puAnnotationDocBlockIndex(s, classKw); d >= 0 && s.At(d).Kind == token.DocComment {
			startScan = d
		}
	}
	changed := false
	for i := close; i >= startScan && i < s.Len(); i-- {
		if s.At(i).Kind != token.DocComment {
			continue
		}
		if f.applyDocBlock(s, i) {
			changed = true
		}
	}
	return changed
}

func (f PhpUnitAttributes) applyDocBlock(s *tokens.Stream, docIdx int) bool {
	scope := puaDocScope(s, docIdx)
	content := s.At(docIdx).Value
	lines := splitDocLines(content)
	anns := docAnnotations(lines)
	if len(anns) == 0 {
		return false
	}
	present := map[string]bool{}
	seen := map[string]bool{}
	inserted := false
	for _, ann := range slices.Backward(anns) {
		if !puaHasMapping(ann.name) || !puaShouldBeFixed(ann.name, scope) {
			continue
		}
		annContent := strings.Join(lines[ann.start:ann.end+1], "")
		attrs := puaFixAnnotation(ann.name, annContent)
		if !seen[ann.name] {
			present[ann.name] = puaAttributeAlreadyPresent(s, docIdx, attrs)
			seen[ann.name] = true
		}
		if len(attrs) == 0 {
			continue
		}
		if !present[ann.name] {
			puaInsertAttributes(s, docIdx, attrs)
			inserted = true
		}
		if !f.keepAnnotations {
			puaRemoveAnnotationLines(lines, ann.start, ann.end)
		}
	}
	newContent := strings.Join(lines, "")
	if newContent != content {
		if newContent == "" {
			clearTokenAndMergeWS(s, docIdx)
		} else {
			s.SetValue(docIdx, newContent)
		}
		return true
	}
	return inserted
}

// puaInsertAttributes inserts the attribute texts right after the doc comment,
// each on its own line, mirroring createAttributeTokens' cloned whitespace.
func puaInsertAttributes(s *tokens.Stream, docIdx int, attrs []string) {
	if docIdx+1 >= s.Len() {
		return
	}
	cloneTok := s.At(docIdx + 1)
	var slice []token.Token
	for _, a := range attrs {
		slice = append(slice, cloneTok)
		slice = append(slice, lexAttributeFragment("#["+a+"]")...)
	}
	s.InsertSliceAt(docIdx+1, slice)
}

// puaRemoveAnnotationLines mirrors Annotation::remove: each line of the span is
// cleared, or trimmed to keep the "/**" start or "*/" end that shares it.
func puaRemoveAnnotationLines(lines []string, start, end int) {
	for idx := start; idx <= end && idx < len(lines); idx++ {
		line := lines[idx]
		hasStart := strings.Contains(line, "/**")
		hasEnd := strings.Contains(line, "*/")
		switch {
		case hasStart && hasEnd:
			lines[idx] = ""
		case hasStart:
			lines[idx] = puaRemoveStartRe.ReplaceAllString(line, "$1")
		case hasEnd:
			lines[idx] = puaRemoveEndRe.ReplaceAllString(line, "$1$2")
		default:
			lines[idx] = ""
		}
	}
}

var (
	puaRemoveStartRe = regexp.MustCompile(`(\s*/\*\*).*`)
	puaRemoveEndRe   = regexp.MustCompile(`(\s*)\S.*(\*/.*)`)
)

// puaDocScope mirrors the getTokenNotOfKindSibling scope detection: a doc block
// that resolves to a class keyword has class scope, otherwise method scope.
func puaDocScope(s *tokens.Stream, docIdx int) string {
	for j := docIdx + 1; j < s.Len(); j++ {
		t := s.At(j)
		switch t.Kind {
		case token.Whitespace, token.Comment:
			continue
		case token.Keyword:
			switch strings.ToLower(t.Value) {
			case "abstract", "final", "private", "protected", "public", "static", "function":
				continue
			}
		}
		if kwIs(t, "class") {
			return "class"
		}
		return "method"
	}
	return "method"
}

func puaPrevClassKeyword(s *tokens.Stream, open int) int {
	for j := open - 1; j >= 0; j-- {
		if kwIs(s.At(j), "class") {
			return j
		}
	}
	return -1
}

// puaFixingWithout are annotations mapping to a parameterless attribute.
var puaFixingWithout = map[string]bool{
	"after": true, "afterClass": true, "before": true, "beforeClass": true,
	"coversNothing": true, "doesNotPerformAssertions": true, "large": true,
	"medium": true, "runInSeparateProcess": true, "runTestsInSeparateProcesses": true,
	"small": true, "test": true, "preCondition": true, "postCondition": true,
}

// puaFixingSingleString are annotations mapping to an attribute with one string.
var puaFixingSingleString = map[string]bool{"group": true, "testDox": true, "ticket": true}

// puaFixingEnabledDisabled are annotations mapping an enabled/disabled word to a
// boolean attribute argument.
var puaFixingEnabledDisabled = map[string]bool{
	"backupGlobals": true, "backupStaticAttributes": true, "preserveGlobalState": true,
}

var puaFixingCustom = map[string]bool{
	"covers": true, "dataProvider": true, "depends": true,
	"requires": true, "testWith": true, "uses": true,
}

func puaHasMapping(name string) bool {
	return puaFixingWithout[name] || puaFixingSingleString[name] ||
		puaFixingEnabledDisabled[name] || puaFixingCustom[name]
}

// puaShouldBeFixed mirrors PhpUnitAttributesFixer::shouldBeFixed.
func puaShouldBeFixed(name, scope string) bool {
	if scope == "method" {
		switch name {
		case "covers", "large", "medium", "runTestsInSeparateProcesses", "small", "uses":
			return false
		}
	}
	if scope == "class" {
		switch name {
		case "after", "afterClass", "before", "beforeClass", "dataProvider", "depends",
			"postCondition", "preCondition", "runInSeparateProcess", "test", "testWith":
			return false
		}
	}
	return true
}

// puaFixAnnotation dispatches an annotation to its mapping and returns the
// attribute bodies to insert (without the surrounding "#[" "]"). An empty slice
// means the annotation could not be parsed and is left untouched.
func puaFixAnnotation(name, content string) []string {
	switch {
	case puaFixingWithout[name]:
		return []string{puaBody(puaAttributeName(name), "")}
	case puaFixingSingleString[name]:
		if m := puaSingleStringRe(name).FindStringSubmatch(content); m != nil {
			return []string{puaBody(puaAttributeName(name), puaEscapedString(m[1]))}
		}
		return nil
	case puaFixingEnabledDisabled[name]:
		m := puaGetMatches(name, content)
		if m[1] == "" {
			return nil
		}
		val := "false"
		if m[1] == "enabled" {
			val = "true"
		}
		return []string{puaBody(puaAttributeName(name), val)}
	case name == "covers":
		return puaFixCovers(content)
	case name == "dataProvider":
		return puaFixDataProvider(content)
	case name == "depends":
		return puaFixDepends(content)
	case name == "requires":
		return puaFixRequires(content)
	case name == "testWith":
		return puaFixTestWith(content)
	case name == "uses":
		return puaFixUses(content)
	}
	return nil
}

func puaFixCovers(content string) []string {
	m := puaGetMatches("covers", content)
	if m[1] == "" {
		return nil
	}
	parts := strings.Split(m[1], "::")
	switch len(parts) {
	case 1:
		name := "CoversClass"
		if strings.HasSuffix(parts[0], "Trait") {
			name = "CoversTrait"
		}
		return []string{puaBody(name, puaClassConstant(parts[0]))}
	case 2:
		if parts[0] == "" {
			return []string{puaBody("CoversFunction", puaEscapedString(parts[1]))}
		}
		if parts[1] != "" {
			args := puaClassConstant(parts[0]) + ", " + puaEscapedString(parts[1])
			return []string{puaBody("CoversMethod", args)}
		}
	}
	return nil
}

func puaFixDataProvider(content string) []string {
	m := puaGetMatches("dataProvider", content)
	if m[1] == "" {
		return nil
	}
	if strings.Contains(m[1], "::") {
		p := strings.SplitN(m[1], "::", 2)
		args := puaClassConstant(p[0]) + ", " + puaEscapedString(puaFixName(p[1]))
		return []string{puaBody("DataProviderExternal", args)}
	}
	return []string{puaBody("DataProvider", puaEscapedString(puaFixName(m[1])))}
}

func puaFixDepends(content string) []string {
	m := puaGetMatches("depends", content)
	if m[1] == "" {
		return nil
	}
	nameSuffix := ""
	depended := m[1]
	if m[2] != "" {
		switch m[1] {
		case "clone":
			nameSuffix = "UsingDeepClone"
			depended = m[2]
		case "shallowClone":
			nameSuffix = "UsingShallowClone"
			depended = m[2]
		}
	}
	hasClass := false
	class := ""
	method := depended
	hasMethod := true
	if strings.Contains(depended, "::") {
		p := strings.SplitN(depended, "::", 2)
		class, method = p[0], p[1]
		hasClass = true
		if method == "class" {
			hasMethod = false
			if nameSuffix == "" {
				nameSuffix = "OnClass"
			} else {
				nameSuffix = "OnClass" + nameSuffix
			}
		} else {
			nameSuffix = "External" + nameSuffix
		}
	}
	args := ""
	if hasClass {
		args = puaClassConstant(class)
	}
	if hasMethod {
		if args != "" {
			args += ", "
		}
		args += puaEscapedString(method)
	}
	return []string{puaBody("Depends"+nameSuffix, args)}
}

var puaRequiresMap = map[string]string{
	"extension": "RequiresPhpExtension", "function": "RequiresFunction",
	"PHP": "RequiresPhp", "PHPUnit": "RequiresPhpunit", "OS": "RequiresOperatingSystem",
	"OSFAMILY": "RequiresOperatingSystemFamily", "setting": "RequiresSetting",
}

func puaFixRequires(content string) []string {
	m := puaGetMatches("requires", content)
	attrName, ok := puaRequiresMap[m[1]]
	if m[2] == "" || !ok {
		return nil
	}
	var args string
	switch {
	case attrName == "RequiresFunction" && strings.Contains(m[2], "::"):
		p := strings.SplitN(m[2], "::", 2)
		attrName = "RequiresMethod"
		args = puaClassConstant(p[0]) + ", " + puaEscapedString(p[1])
	case (attrName == "RequiresPhp" || attrName == "RequiresPhpunit") && m[3] != "":
		args = puaEscapedString(m[2] + " " + m[3])
	default:
		args = puaEscapedString(puaFixVersionConstraint(m[2]))
	}
	if m[3] != "" && attrName != "RequiresPhp" && attrName != "RequiresPhpunit" {
		args += ", " + puaEscapedString(puaFixVersionConstraint(m[3]))
	}
	return []string{puaBody(attrName, args)}
}

var puaVersionConstraintRe = regexp.MustCompile(`^[\d.-]+(dev|(RC|alpha|beta)[\d.])?$`)

func puaFixVersionConstraint(version string) string {
	if puaVersionConstraintRe.MatchString(version) {
		return ">= " + version
	}
	return version
}

var (
	puaTestWithTagRe  = regexp.MustCompile(`@testWith\s+`)
	puaTestWithLineRe = regexp.MustCompile(`(^|\r\n|\n|\r)\s+\**\s*`)
)

func puaFixTestWith(content string) []string {
	content = puaTestWithTagRe.ReplaceAllString(content, "")
	content = puaTestWithLineRe.ReplaceAllString(content, "\n")
	content = strings.Trim(content, " \t\n\r\x00\x0b")
	if content == "" {
		return nil
	}
	var out []string
	for json := range strings.SplitSeq(content, "\n") {
		out = append(out, puaBody("TestWithJson", puaEscapedString(json)))
	}
	return out
}

var puaUsesClassRe = regexp.MustCompile(`^[a-zA-Z\d\\]+$`)

func puaFixUses(content string) []string {
	m := puaGetMatches("uses", content)
	if m[1] == "" {
		return nil
	}
	switch {
	case strings.HasPrefix(m[1], "::"):
		return []string{puaBody("UsesFunction", puaEscapedString(m[1][2:]))}
	case puaUsesClassRe.MatchString(m[1]):
		return []string{puaBody("UsesClass", puaClassConstant(m[1]))}
	}
	return nil
}

// puaGetMatches mirrors PhpUnitAttributesFixer::getMatches, returning a 4-element
// slice where index 1..3 are the optional capture groups ("" when absent).
func puaGetMatches(name, content string) []string {
	re := puaMatchesRe(name)
	m := re.FindStringSubmatch(content)
	if m == nil {
		return []string{"", "", "", ""}
	}
	for len(m) < 4 {
		m = append(m, "")
	}
	return m
}

var puaMatchesReCache = map[string]*regexp.Regexp{}

func puaMatchesRe(name string) *regexp.Regexp {
	if re, ok := puaMatchesReCache[name]; ok {
		return re
	}
	re := regexp.MustCompile(`@` + regexp.QuoteMeta(name) + `\s+(\S+)(?:\s+(\S+))?(?:\s+(.+\S))?\s*(?:\r\n|\n|\r|\*+/$)`)
	puaMatchesReCache[name] = re
	return re
}

var puaSingleStringReCache = map[string]*regexp.Regexp{}

func puaSingleStringRe(name string) *regexp.Regexp {
	if re, ok := puaSingleStringReCache[name]; ok {
		return re
	}
	re := regexp.MustCompile(`@` + regexp.QuoteMeta(name) + `\s+(.*\S)(?:\r\n|\n|\r|\s*\*+/$)`)
	puaSingleStringReCache[name] = re
	return re
}

func puaAttributeName(name string) string {
	if name == "backupStaticAttributes" {
		return "BackupStaticProperties"
	}
	return puaUcfirst(name)
}

func puaUcfirst(v string) string {
	if v != "" && v[0] >= 'a' && v[0] <= 'z' {
		return string(v[0]-'a'+'A') + v[1:]
	}
	return v
}

func puaBody(className, args string) string {
	b := `\PHPUnit\Framework\Attributes\` + className
	if args != "" {
		b += "(" + args + ")"
	}
	return b
}

func puaClassConstant(name string) string {
	return name + "::class"
}

func puaEscapedString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `\'`) + "'"
}

// puaFixName mirrors fixNameAndCreateEscapedStringToken's rtrim.
func puaFixName(name string) string {
	return strings.TrimRight(name, " ()\n\r\t\x0b\x00")
}

// puaAttributeAlreadyPresent mirrors isAttributeAlreadyPresent: the attribute
// right after the doc block already declares the class that would be inserted.
func puaAttributeAlreadyPresent(s *tokens.Stream, docIdx int, attrs []string) bool {
	if len(attrs) == 0 {
		return false
	}
	nm := sigNext(s, docIdx)
	if nm < 0 || s.At(nm).Kind != token.AttributeStart {
		return false
	}
	inserted := strings.TrimPrefix(attrs[0], `\`)
	if i := strings.IndexByte(inserted, '('); i >= 0 {
		inserted = inserted[:i]
	}
	useMap := puaUseMap(s)
	for _, name := range puaAttrNames(s, nm) {
		fqn := puaResolveName(name, useMap)
		if fqn == inserted {
			return true
		}
		if inserted == `PHPUnit\Framework\Attributes\TestWithJson` && fqn == `PHPUnit\Framework\Attributes\TestWith` {
			return true
		}
	}
	return false
}

func puaResolveName(name string, useMap map[string]string) string {
	if strings.HasPrefix(name, `\`) {
		return name[1:]
	}
	first, rest := name, ""
	if i := strings.IndexByte(name, '\\'); i >= 0 {
		first, rest = name[:i], name[i:]
	}
	if fq, ok := useMap[first]; ok {
		return fq + rest
	}
	return name
}

// puaAttrNames reads the attribute names in the run of attributes beginning at
// nm (comma-separated within a group, and consecutive "#[..]" groups).
func puaAttrNames(s *tokens.Stream, nm int) []string {
	var names []string
	i := nm
	for i >= 0 && i < s.Len() && s.At(i).Kind == token.AttributeStart {
		end := s.AttributeEnd(i)
		if end < 0 {
			return names
		}
		j := sigNext(s, i)
		for j >= 0 && j < end {
			var sb strings.Builder
			for j < end {
				t := s.At(j)
				if t.Kind == token.Ident || (t.Kind == token.Punct && t.Value == `\`) {
					sb.WriteString(t.Value)
					j++
					continue
				}
				break
			}
			if sb.Len() > 0 {
				names = append(names, sb.String())
			}
			if j < end && isPunctVal(s, j, "(") {
				if mm := s.MatchForward(j); mm >= 0 {
					j = mm + 1
				} else {
					return names
				}
			}
			for j < end && !isPunctVal(s, j, ",") {
				j++
			}
			if j < end {
				j++
			}
			for j < end && s.At(j).Kind == token.Whitespace {
				j++
			}
		}
		nx := sigNext(s, end)
		if nx >= 0 && s.At(nx).Kind == token.AttributeStart {
			i = nx
			continue
		}
		break
	}
	return names
}

// puaUseMap maps an import alias (or last segment) to its fully-qualified name.
func puaUseMap(s *tokens.Stream) map[string]string {
	m := map[string]string{}
	for i := 0; i < s.Len(); i++ {
		if !kwIs(s.At(i), "use") {
			continue
		}
		n := sigNext(s, i)
		if n < 0 || kwIs(s.At(n), "function") || kwIs(s.At(n), "const") {
			continue
		}
		var sb strings.Builder
		alias := ""
		for j := n; j < s.Len(); j++ {
			t := s.At(j)
			if isPunctVal(s, j, ";") || isPunctVal(s, j, "{") {
				break
			}
			if (t.Kind == token.Keyword || t.Kind == token.Ident) && strings.EqualFold(t.Value, "as") {
				if a := sigNext(s, j); a >= 0 {
					alias = s.At(a).Value
				}
				break
			}
			if t.Kind == token.Ident || (t.Kind == token.Punct && t.Value == `\`) {
				sb.WriteString(t.Value)
			}
		}
		fqn := strings.TrimPrefix(sb.String(), `\`)
		if fqn == "" {
			continue
		}
		if alias == "" {
			if i := strings.LastIndex(fqn, `\`); i >= 0 {
				alias = fqn[i+1:]
			} else {
				alias = fqn
			}
		}
		m[alias] = fqn
	}
	return m
}
