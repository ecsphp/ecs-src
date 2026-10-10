package rules

import (
	"path/filepath"
	"regexp"
	"strings"

	"blink/internal/fixer"
	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Basic/PsrAutoloadingFixer.php
//
// PsrAutoloading renames the single class/interface/trait/enum in a file so its
// name matches the file basename, and (with "dir" set) corrects the namespace
// casing to the path. It needs the file path, which the runner puts on the
// stream; a stream with no backing path is left untouched.
type PsrAutoloading struct {
	// Dir, when set, restricts the fixer to files under it and prefixes class
	// names from their path relative to it. Empty means the default mode.
	Dir string
}

func (f PsrAutoloading) WithConfig(config map[string]any) fixer.Fixer {
	if d, ok := config["dir"].(string); ok {
		f.Dir = d
	}
	return f
}

func (PsrAutoloading) Name() string {
	return `PhpCsFixer\Fixer\Basic\PsrAutoloadingFixer`
}

func (PsrAutoloading) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Basic/PsrAutoloadingFixer.php"
}

var psrIdentifierRe = regexp.MustCompile(`^[A-Za-z_\x80-\xff][A-Za-z0-9_\x80-\xff]*$`)

var psrStubFixtureRe = regexp.MustCompile(`(?i)[/\\](stub|fixture)s?[/\\]`)

var psrClassyKeywords = map[string]bool{
	"class": true, "interface": true, "trait": true, "enum": true,
}

func (f PsrAutoloading) Fix(s *tokens.Stream) bool {
	path := s.Path()
	if path == "" {
		return false
	}

	realPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	realPath = filepath.Clean(realPath)

	base := filepath.Base(realPath)
	if strings.ToLower(filepath.Ext(base)) != ".php" {
		return false
	}
	name := strings.TrimSuffix(base, filepath.Ext(base))
	if !psrIdentifierRe.MatchString(name) {
		return false
	}
	if psrStubFixtureRe.MatchString(realPath) {
		return false
	}

	if f.Dir != "" {
		dir, derr := filepath.Abs(f.Dir)
		if derr != nil {
			return false
		}
		if !strings.HasPrefix(realPath, filepath.Clean(dir)) {
			return false
		}
	}

	namespace := ""
	haveNamespace := false
	namespaceStart, namespaceEnd := -1, -1

	classyName := ""
	classyIndex := -1

	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if kwIs(t, "namespace") {
			if haveNamespace {
				return false
			}
			start := sigNext(s, i)
			end := nextPunctOfKind(s, start, ";")
			if start < 0 || end < 0 {
				return false
			}
			haveNamespace = true
			namespaceStart, namespaceEnd = start, end
			namespace = collectNamespace(s, start, end)
		} else if t.Kind == token.Keyword && psrClassyKeywords[strings.ToLower(t.Value)] {
			if strings.EqualFold(t.Value, "class") {
				if p := sigPrev(s, i); p >= 0 && (kwIs(s.At(p), "new") || isPunctVal(s, p, "::")) {
					continue // anonymous class or ::class constant
				}
			}
			if classyIndex != -1 {
				return false
			}
			classyIndex = sigNext(s, i)
			if classyIndex < 0 {
				return false
			}
			classyName = s.At(classyIndex).Value
		}
	}

	if classyIndex == -1 {
		return false
	}

	changed := false
	expected := f.calculateClassyName(realPath, namespace, haveNamespace, classyName)
	if classyName != expected {
		s.SetValue(classyIndex, expected)
		changed = true
	}

	if f.Dir == "" || !haveNamespace {
		return changed
	}
	if f.rewriteNamespaceCase(s, realPath, namespace, namespaceStart, namespaceEnd) {
		changed = true
	}
	return changed
}

// collectNamespace joins the name tokens between the "namespace" keyword and the
// terminating ";" into a backslash-separated string.
func collectNamespace(s *tokens.Stream, start, end int) string {
	var b strings.Builder
	for i := start; i < end; i++ {
		b.WriteString(s.At(i).Value)
	}
	return strings.TrimSpace(b.String())
}

func (f PsrAutoloading) calculateClassyName(realPath, namespace string, haveNamespace bool, currentName string) string {
	name := strings.TrimSuffix(filepath.Base(realPath), filepath.Ext(realPath))
	maxNamespace := f.calculateMaxNamespace(realPath, namespace, haveNamespace)

	if f.Dir != "" {
		if maxNamespace != "" {
			return strings.ReplaceAll(maxNamespace, `\`, "_") + "_" + name
		}
		return name
	}

	parts := reverseStrings(strings.Split(maxNamespace, `\`))
	for _, part := range parts {
		candidate := part + "_" + name
		if !strings.EqualFold(candidate, suffix(currentName, len(candidate))) {
			break
		}
		name = candidate
	}
	return name
}

func (f PsrAutoloading) calculateMaxNamespace(realPath, namespace string, haveNamespace bool) string {
	fileDir := filepath.Dir(realPath)

	var root string
	if f.Dir != "" {
		abs, err := filepath.Abs(f.Dir)
		if err != nil {
			return ""
		}
		root = filepath.Clean(abs)
	} else {
		root = fileDir
		for root != filepath.Dir(root) {
			root = filepath.Dir(root)
		}
	}

	rel := strings.TrimPrefix(fileDir, root)
	rel = strings.Trim(strings.ReplaceAll(rel, string(filepath.Separator), `\`), `\`)
	if !haveNamespace {
		return rel
	}

	locationReversed := reverseStrings(strings.Split(rel, `\`))
	namespaceReversed := reverseStrings(strings.Split(namespace, `\`))
	for k, part := range namespaceReversed {
		if k >= len(locationReversed) {
			break
		}
		if !strings.EqualFold(part, locationReversed[k]) {
			break
		}
		locationReversed[k] = ""
	}

	kept := make([]string, 0, len(locationReversed))
	for _, p := range reverseStrings(locationReversed) {
		if p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, `\`)
}

// rewriteNamespaceCase corrects the trailing namespace segments to the path
// casing when they differ only by case, mirroring the "dir" branch of applyFix.
func (f PsrAutoloading) rewriteNamespaceCase(s *tokens.Stream, realPath, namespace string, nsStart, nsEnd int) bool {
	abs, err := filepath.Abs(f.Dir)
	if err != nil {
		return false
	}
	configuredDir := filepath.Clean(abs)
	fileDir := filepath.Dir(realPath)
	if len(configuredDir) >= len(fileDir) {
		return false
	}

	newNamespace := strings.ReplaceAll(fileDir[len(configuredDir)+1:], string(filepath.Separator), `\`)
	if len(newNamespace) > len(namespace) {
		return false
	}
	original := namespace[len(namespace)-len(newNamespace):]
	if original == newNamespace || !strings.EqualFold(original, newNamespace) {
		return false
	}

	corrected := namespace[:len(namespace)-len(newNamespace)] + newNamespace
	s.ReplaceRange(nsStart, nsEnd-1, namespaceNameTokens(corrected))
	return true
}

// namespaceNameTokens builds the Ident / "\" token run for a namespace name.
func namespaceNameTokens(name string) []token.Token {
	segments := strings.Split(name, `\`)
	out := make([]token.Token, 0, len(segments)*2-1)
	for i, seg := range segments {
		if i > 0 {
			out = append(out, token.Token{Kind: token.Punct, Value: `\`})
		}
		out = append(out, token.Token{Kind: token.Ident, Value: seg})
	}
	return out
}

func reverseStrings(in []string) []string {
	out := make([]string, len(in))
	for i, v := range in {
		out[len(in)-1-i] = v
	}
	return out
}

// suffix returns the last n bytes of v, or all of v when it is shorter.
func suffix(v string, n int) string {
	if n >= len(v) {
		return v
	}
	return v[len(v)-n:]
}
