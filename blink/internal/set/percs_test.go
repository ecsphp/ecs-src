package set

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"blink/internal/fixer"
	"blink/internal/fixer/rules"
)

const perCSFile = "../../../config/set/per-cs.php"

// perCSRules must list exactly the rules and options of per-cs.php.
func TestPERCSMatchesECS(t *testing.T) {
	src, err := os.ReadFile(perCSFile)
	if err != nil {
		t.Fatal(err)
	}
	want, err := parseECSSet(string(src))
	if err != nil {
		t.Fatalf("%s: %v", perCSFile, err)
	}

	got := map[string]map[string]any{}
	for _, r := range perCSRules {
		if _, dup := got[r.class]; dup {
			t.Errorf("perCSRules lists %s twice", r.class)
		}
		got[r.class] = r.config
	}
	for class, config := range want {
		g, ok := got[class]
		if !ok {
			t.Errorf("per-cs.php has %s, perCSRules does not", class)
			continue
		}
		if !reflect.DeepEqual(g, config) {
			t.Errorf("%s: perCSRules config %#v, per-cs.php %#v", class, g, config)
		}
	}
	for class := range got {
		if _, ok := want[class]; !ok {
			t.Errorf("perCSRules has %s, per-cs.php does not", class)
		}
	}
}

// PERCS() holds exactly the per-cs rules blink implements.
func TestPERCSIsExactlyTheImplementedPERCSRules(t *testing.T) {
	want := map[string]bool{}
	for _, r := range perCSRules {
		f, ok := rules.ByName(r.class)
		if !ok {
			continue // not ported yet
		}
		if _, ok := f.(fixer.ConfigurableFixer); r.config != nil && !ok {
			t.Errorf("%s is configured by per-cs.php but blink's fixer takes no configuration", r.class)
		}
		want[f.Name()] = true
	}
	got := map[string]bool{}
	for _, f := range PERCS() {
		got[f.Name()] = true
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("PERCS() = %v\nwant %v", keys(got), keys(want))
	}
	if len(got) == 0 {
		t.Fatal("empty per-cs set")
	}
}

func keys(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

// parseECSSet returns the rules of an ECS set file built with withConfiguredRule
// and withRules, with their configuration (nil for withRules). Anything else is
// an error, so a new construct fails the test instead of being ignored.
func parseECSSet(src string) (map[string]map[string]any, error) {
	imports := map[string]string{}
	for _, m := range regexp.MustCompile(`(?m)^use ([\w\\]+);`).FindAllStringSubmatch(src, -1) {
		fqcn := m[1]
		imports[fqcn[strings.LastIndexByte(fqcn, '\\')+1:]] = fqcn
	}
	_, after, ok := strings.Cut(src, "ECSConfig::configure()")
	if !ok {
		return nil, fmt.Errorf("no ECSConfig::configure() builder")
	}
	p := &phpParser{src: after}
	out := map[string]map[string]any{}
	class := func(v any) (string, error) {
		ref, ok := v.(classRef)
		if !ok {
			return "", fmt.Errorf("expected X::class, got %#v", v)
		}
		fqcn, ok := imports[string(ref)]
		if !ok {
			return "", fmt.Errorf("%s::class is not imported", ref)
		}
		if _, dup := out[fqcn]; dup {
			return "", fmt.Errorf("%s enabled twice", fqcn)
		}
		return fqcn, nil
	}
	for {
		p.skipSpace()
		if p.eat(";") {
			return out, nil
		}
		if !p.eat("->") {
			return nil, fmt.Errorf("expected ->method( at %q", p.rest())
		}
		method := p.ident()
		if !p.eat("(") {
			return nil, fmt.Errorf("expected ( after %s", method)
		}
		args, err := p.list(")")
		if err != nil {
			return nil, err
		}
		switch method {
		case "withConfiguredRule":
			if len(args) != 2 {
				return nil, fmt.Errorf("withConfiguredRule takes 2 arguments, got %d", len(args))
			}
			fqcn, err := class(args[0])
			if err != nil {
				return nil, err
			}
			config, ok := args[1].(map[string]any)
			if !ok {
				return nil, fmt.Errorf("%s: configuration is not a keyed array", fqcn)
			}
			out[fqcn] = config
		case "withRules":
			list, ok := args[0].([]any)
			if len(args) != 1 || !ok {
				return nil, fmt.Errorf("withRules takes one list")
			}
			for _, v := range list {
				fqcn, err := class(v)
				if err != nil {
					return nil, err
				}
				out[fqcn] = nil
			}
		default:
			return nil, fmt.Errorf("unsupported builder call ->%s(): teach this parser and blink's per-cs set about it", method)
		}
	}
}

// classRef is a parsed `Name::class` constant.
type classRef string

// phpParser parses the PHP literals ECS set files use: single-quoted strings,
// true/false, Name::class and [...] arrays (keyed arrays become map[string]any).
type phpParser struct{ src string }

func (p *phpParser) rest() string {
	if len(p.src) > 40 {
		return p.src[:40]
	}
	return p.src
}

func (p *phpParser) skipSpace() { p.src = strings.TrimLeft(p.src, " \t\r\n") }

func (p *phpParser) eat(s string) bool {
	p.skipSpace()
	if strings.HasPrefix(p.src, s) {
		p.src = p.src[len(s):]
		return true
	}
	return false
}

func (p *phpParser) ident() string {
	p.skipSpace()
	i := 0
	for i < len(p.src) && (p.src[i] == '_' || p.src[i] == '\\' ||
		'a' <= p.src[i] && p.src[i] <= 'z' || 'A' <= p.src[i] && p.src[i] <= 'Z' || '0' <= p.src[i] && p.src[i] <= '9') {
		i++
	}
	id := p.src[:i]
	p.src = p.src[i:]
	return id
}

// list parses comma-separated values up to closer (a trailing comma allowed).
func (p *phpParser) list(closer string) ([]any, error) {
	var out []any
	for !p.eat(closer) {
		v, err := p.value()
		if err != nil {
			return nil, err
		}
		out = append(out, v)
		if !p.eat(",") && !strings.HasPrefix(p.src, closer) {
			return nil, fmt.Errorf("expected , or %s at %q", closer, p.rest())
		}
	}
	return out, nil
}

func (p *phpParser) value() (any, error) {
	p.skipSpace()
	switch {
	case p.eat("["):
		return p.array()
	case p.eat("'"):
		i := strings.IndexAny(p.src, `'\\`)
		if i < 0 || p.src[i] != '\'' {
			return nil, fmt.Errorf("unsupported string at %q", p.rest())
		}
		v := p.src[:i]
		p.src = p.src[i+1:]
		return v, nil
	}
	id := p.ident()
	switch {
	case id == "":
		return nil, fmt.Errorf("unexpected %q", p.rest())
	case p.eat("::class"):
		return classRef(id), nil
	case id == "true":
		return true, nil
	case id == "false":
		return false, nil
	}
	return nil, fmt.Errorf("unsupported PHP value %q", id)
}

// array parses a [...] literal after its "[".
func (p *phpParser) array() (any, error) {
	var list []any
	keyed := map[string]any{}
	for !p.eat("]") {
		v, err := p.value()
		if err != nil {
			return nil, err
		}
		if p.eat("=>") {
			key, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("non-string array key %#v", v)
			}
			if keyed[key], err = p.value(); err != nil {
				return nil, err
			}
		} else {
			list = append(list, v)
		}
		if !p.eat(",") && !strings.HasPrefix(p.src, "]") {
			return nil, fmt.Errorf("expected , or ] at %q", p.rest())
		}
	}
	switch {
	case len(keyed) > 0 && len(list) > 0:
		return nil, fmt.Errorf("mixed list and keyed array")
	case len(keyed) > 0:
		return keyed, nil
	}
	return list, nil
}
