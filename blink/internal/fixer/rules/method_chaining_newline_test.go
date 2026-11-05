package rules

import "testing"

func TestMethodChainingNewline(t *testing.T) {
	t.Parallel()
	f := MethodChainingNewline{}

	for _, tc := range []struct{ name, in, want string }{
		{
			"chain",
			"<?php\n$item = $node->getChildNodes()->get(0)->getChildNodes();\n",
			"<?php\n$item = $node->getChildNodes()\n->get(0)\n->getChildNodes();\n",
		},
		{
			"a cast is one token",
			"<?php\n$x = (int) self::FOO->getValue();\n$y = foo((int) $a)->getFirst()->getSecond();\n",
			"<?php\n$x = (int) self::FOO->getValue();\n$y = foo((int) $a)\n->getFirst()\n->getSecond();\n",
		},
		{
			"interpolation and a space before the arrow",
			"<?php\n$s = \"x {$a->getFirst()->getSecond()} y\";\n$t = $a->getFirst() ->getSecond();\n",
			"<?php\n$s = \"x {$a->getFirst()\n->getSecond()} y\";\n$t = $a->getFirst() \n->getSecond();\n",
		},
	} {
		if got, changed := apply(t, f, tc.in); !changed || got != tc.want {
			t.Errorf("%s: changed=%v got=%q", tc.name, changed, got)
		}
	}

	// condition, comparison, short no-arg method, array item, already multiline
	skip := "<?php\nif ($a->getFirst()->getSecond()) {\n}\n$b = $a->getFirst()->getSecond() && $c;\n$d = $a->getFirst()->get();\n$e = [$a->getFirst()->getSecond()];\n$f = $a->getFirst()\n    ->getSecond();\n"
	if got, changed := apply(t, f, skip); changed {
		t.Fatalf("skip: got=%q", got)
	}
}
