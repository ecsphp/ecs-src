package rules

import "testing"

// The expected output of every case comes from the PHP engine.
func TestMethodArgumentSpaceParentheses(t *testing.T) {
	t.Parallel()
	f := configuredFromJSON(t, MethodArgumentSpace{}, `{"after_heredoc": true, "on_multiline": "ignore"}`)
	cases := []struct{ name, src, want string }{
		// every "(" that does not follow a keyword
		{"parentheses", "$x = ([$a , $b] = $c);", "$x = ([$a, $b] = $c);"},
		{"parentheses in a condition", "if (([$a , $b] = foo()) !== null) {}", "if (([$a, $b] = foo()) !== null) {}"},
		{"parentheses in arguments", "foo(([$a , $b] = $c));", "foo(([$a, $b] = $c));"},
		{"parentheses in an array", "$x = [([$a , $b] = $c)];", "$x = [([$a, $b] = $c)];"},
		{"attribute in parentheses", "$f = (#[A , B] fn () => 1);", "$f = (#[A, B] fn () => 1);"},
		{"by-reference closure", "$f = function &($a , $b) {};", "$f = function &($a, $b) {};"},
		{"by-reference arrow function", "$f = fn &($a , $b) => $a;", "$f = fn &($a, $b) => $a;"},
		{"string callable", "'strlen'($a , $b);", "'strlen'($a, $b);"},

		// a "(" after a keyword
		{"return", "return ([$a , $b] = $c);", "return ([$a , $b] = $c);"},
		{"new static", "new static($a , $b);", "new static($a , $b);"},
		{"long array", "$x = array($a , $b);", "$x = array($a , $b);"},
		{"isset", "isset($a , $b);", "isset($a , $b);"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertFix(t, f, "<?php\n\n"+c.src+"\n", "<?php\n\n"+c.want+"\n", c.src != c.want)
		})
	}
}
