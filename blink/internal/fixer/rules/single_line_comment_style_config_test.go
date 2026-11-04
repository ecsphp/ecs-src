package rules

import "testing"

func TestSingleLineCommentStyleConfig(t *testing.T) {
	t.Parallel()
	const in = "<?php\n# hash\n/* asterisk */\n$a = 1;\n"
	cases := []struct {
		name   string
		config map[string]any
		want   string
	}{
		{"default (empty config)", map[string]any{}, "<?php\n// hash\n// asterisk\n$a = 1;\n"},
		{"hash only", map[string]any{"comment_types": []any{"hash"}}, "<?php\n// hash\n/* asterisk */\n$a = 1;\n"},
		{"asterisk only", map[string]any{"comment_types": []any{"asterisk"}}, "<?php\n# hash\n// asterisk\n$a = 1;\n"},
		{"none", map[string]any{"comment_types": []any{}}, in},
	}
	for _, c := range cases {
		f := SingleLineCommentStyle{}.WithConfig(c.config).(fixerRule)
		if got, _ := apply(t, f, in); got != c.want {
			t.Errorf("%s: got=%q want=%q", c.name, got, c.want)
		}
	}
}
