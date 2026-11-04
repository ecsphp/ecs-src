package rules

import (
	"testing"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
)

// TestClassDefinitionMatchesPHPCSFixer runs the port over one input under the
// option sets ECS uses; the expected outputs come from PHP-CS-Fixer.
func TestClassDefinitionMatchesPHPCSFixer(t *testing.T) {
	t.Parallel()
	in := "<?php\n$a = new class(\n    $bar,\n    $baz\n) {};\n$b = new class ( $x ) extends Foo implements Bar,\n    Baz {};\nclass  A  extends  B  implements  C {}\nfinal readonly class D\n    extends E\n    implements F, G\n{\n}\nreadonly final class H {}\ninterface I extends J,\n    K, L {}\nabstract class M implements\n    N,\n    O\n{\n}\nclass P implements Q, /* c */ R // d\n{\n}\nenum S: string implements T {}\n$c = Foo::class;\n$d = new #[Attr(\n    1\n)] class {};\ntrait U {}\n"
	cases := []struct {
		name   string
		config map[string]any
		want   string
	}{
		{"def", map[string]any{}, "<?php\n$a = new class($bar, $baz) {};\n$b = new class($x) extends Foo implements\n    Bar,\n    Baz\n{};\nclass A extends B implements C\n{}\nfinal readonly class D extends E implements F, G\n{\n}\nfinal readonly class H\n{}\ninterface I extends J,\n    K, L\n{}\nabstract class M implements\n    N,\n    O\n{\n}\nclass P implements Q, /* c */ R // d\n{\n}\nenum S: string implements T\n{}\n$c = Foo::class;\n$d = new #[Attr(1)] class {};\ntrait U\n{}\n"},
		{"single", map[string]any{"single_line": true}, "<?php\n$a = new class($bar, $baz) {};\n$b = new class($x) extends Foo implements Bar, Baz {};\nclass A extends B implements C\n{}\nfinal readonly class D extends E implements F, G\n{\n}\nfinal readonly class H\n{}\ninterface I extends J, K, L\n{}\nabstract class M implements N, O\n{\n}\nclass P implements Q, /* c */ R // d\n{\n}\nenum S: string implements T\n{}\n$c = Foo::class;\n$d = new #[Attr(1)] class {};\ntrait U\n{}\n"},
		{"percs", map[string]any{"inline_constructor_arguments": false, "space_before_parenthesis": true, "multi_line_extends_each_single_line": true}, "<?php\n$a = new class (\n    $bar,\n    $baz\n) {};\n$b = new class ( $x ) extends Foo implements\n    Bar,\n    Baz\n{};\nclass A extends B implements C\n{}\nfinal readonly class D extends E implements F, G\n{\n}\nfinal readonly class H\n{}\ninterface I extends\n    J,\n    K,\n    L\n{}\nabstract class M implements\n    N,\n    O\n{\n}\nclass P implements Q, /* c */ R // d\n{\n}\nenum S: string implements T\n{}\n$c = Foo::class;\n$d = new #[Attr(1)] class {};\ntrait U\n{}\n"},
		{"item", map[string]any{"single_item_single_line": true}, "<?php\n$a = new class($bar, $baz) {};\n$b = new class($x) extends Foo implements\n    Bar,\n    Baz\n{};\nclass A extends B implements C\n{}\nfinal readonly class D extends E implements F, G\n{\n}\nfinal readonly class H\n{}\ninterface I extends J,\n    K, L\n{}\nabstract class M implements\n    N,\n    O\n{\n}\nclass P implements Q, /* c */ R // d\n{\n}\nenum S: string implements T\n{}\n$c = Foo::class;\n$d = new #[Attr(1)] class {};\ntrait U\n{}\n"},
	}
	for _, c := range cases {
		f, _ := ByName(`PhpCsFixer\Fixer\ClassNotation\ClassDefinitionFixer`)
		f = f.(fixer.ConfigurableFixer).WithConfig(c.config)
		if got, _ := apply(t, f, in); got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}

// A reserved word naming an argument ("#[Foo(class: Bar::class)]") is not a
// class declaration: PHP-CS-Fixer leaves the code after it alone.
func TestClassDefinitionIgnoresNamedArgumentClass(t *testing.T) {
	t.Parallel()
	src := "<?php\nclass A\n{\n    #[Foo(class: Bar::class)]\n    private $x;\n\n    public function f(\n        int $a,\n    ): int {\n        return 1;\n    }\n}\n"
	if got, _ := apply(t, ClassDefinition{}, src); got != src {
		t.Errorf("got %q", got)
	}
}
