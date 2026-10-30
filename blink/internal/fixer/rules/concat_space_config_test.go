package rules

import "testing"

func TestConcatSpaceConfigCases(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		rule fixerRule
		src  string
		want string
	}{
		{"default one", ConcatSpace{}, "<?php $a = 'a'.'b';", "<?php $a = 'a' . 'b';"},
		{"one explicit", ConcatSpace{}.WithConfig(map[string]any{"spacing": "one"}).(fixerRule), "<?php $a = 'a'.'b';", "<?php $a = 'a' . 'b';"},
		{"none", ConcatSpace{}.WithConfig(map[string]any{"spacing": "none"}).(fixerRule), "<?php $a = 'a' . 'b';", "<?php $a = 'a'.'b';"},
		{"attribute concat spaced", ConcatSpace{}, "<?php #[WithEnv('K', __DIR__.'/x.php')]\nclass C {}", "<?php #[WithEnv('K', __DIR__ . '/x.php')]\nclass C {}"},
		{"attribute concat none", ConcatSpace{}.WithConfig(map[string]any{"spacing": "none"}).(fixerRule), "<?php #[WithEnv('K', __DIR__ . '/x.php')]\nclass C {}", "<?php #[WithEnv('K', __DIR__.'/x.php')]\nclass C {}"},
		{"dot inside attribute string left alone", ConcatSpace{}, "<?php #[X('a.b . c')]\nclass C {}", "<?php #[X('a.b . c')]\nclass C {}"},
		{"float literal in attribute left alone", ConcatSpace{}, "<?php #[X(1.5)]\nclass C {}", "<?php #[X(1.5)]\nclass C {}"},
		{"spread in attribute left alone", ConcatSpace{}, "<?php #[X(...$a)]\nclass C {}", "<?php #[X(...$a)]\nclass C {}"},
		{"scope resolution in attribute left alone", ConcatSpace{}, "<?php #[X(self::A.'b')]\nclass C {}", "<?php #[X(self::A . 'b')]\nclass C {}"},
	}
	for _, c := range cases {
		got, _ := apply(t, c.rule, c.src)
		if got != c.want {
			t.Errorf("%s: got=%q want=%q", c.name, got, c.want)
		}
	}
}
