package rules

import "testing"

// Cases ported from PHP-CS-Fixer's GroupImportFixerTest (tests/Fixer/Import);
// every expected output is what PHP-CS-Fixer produces for the input. The
// invalid-configuration and PHP<8 comment cases are omitted: blink does not
// validate options and targets PHP 8+.
func TestGroupImportPhpCsFixerCases(t *testing.T) {
	t.Parallel()

	classy := map[string]any{"group_types": []string{groupImportClassy}}
	functions := map[string]any{"group_types": []string{groupImportFunctions}}
	constants := map[string]any{"group_types": []string{groupImportConstants}}
	classyFunctions := map[string]any{"group_types": []string{groupImportClassy, groupImportFunctions}}
	classyConstants := map[string]any{"group_types": []string{groupImportClassy, groupImportConstants}}
	functionsConstants := map[string]any{"group_types": []string{groupImportFunctions, groupImportConstants}}

	aliasRunUpstreamCases(t, GroupImport{}, []aliasUpstreamCase{
		{"leading namespace, aliases", nil,
			"<?php\nnamespace X1\\Y1;\n\nuse A\\B as X;\nuse A\\C as Y;\nuse B\\C;\n",
			"<?php\nnamespace X1\\Y1;\n\nuse A\\{B as X, C as Y};\nuse B\\C;\n"},
		{"two aliases", nil,
			"<?php\n\nnamespace Foo\\Bar;\n\nuse A\\B as X;\nuse A\\C as Y;\n",
			"<?php\n\nnamespace Foo\\Bar;\n\nuse A\\{B as X, C as Y};\n"},
		{"three classes", nil,
			"<?php\n\nnamespace Test;\n\nuse Foo\\Bar;\nuse Foo\\Baz;\nuse Foo\\Test;\n",
			"<?php\n\nnamespace Test;\n\nuse Foo\\{Bar, Baz, Test};\n"},
		{"single kept between group", nil,
			"<?php\n\nuse App\\First;\nuse Foo\\Bar;\nuse Test\\Second;\nuse Foo\\Baz;\n",
			"<?php\n\nuse App\\First;\nuse Foo\\{Bar, Baz};\nuse Test\\Second;\n"},
		{"two groups, sorted", nil,
			"<?php\n\nuse Foo\\Bar;\nuse Foo\\Baz;\nuse App\\First;\nuse App\\Second;\n",
			"<?php\n\nuse App\\{First, Second};\nuse Foo\\{Bar, Baz};\n"},
		{"two interleaved groups", nil,
			"<?php\n\nuse Foo\\Bar;\nuse App\\First;\nuse Foo\\Baz;\nuse App\\Second;\n",
			"<?php\n\nuse App\\{First, Second};\nuse Foo\\{Bar, Baz};\n"},
		{"group then single", nil,
			"<?php\n\nuse Foo\\Bar as Test;\nuse Foo\\Baz;\nuse App;\n",
			"<?php\n\nuse Foo\\{Bar as Test, Baz};\nuse App;\n"},
		{"alias sorted first", nil,
			"<?php\n\nuse App\\Repository\\User;\nuse App\\Repository\\Profile;\nuse App\\Repository\\Customer as Client;\n",
			"<?php\n\nuse App\\Repository\\{Customer as Client, Profile, User};\n"},
		{"functions with alias", nil,
			"<?php\n\nuse function Foo\\Bar;\nuse function Foo\\Baz;\nuse function Foo\\Test as Alias;\n",
			"<?php\n\nuse function Foo\\{Bar, Baz, Test as Alias};\n"},
		{"constants with alias", nil,
			"<?php\n\nuse const Some\\Place\\A;\nuse const Some\\Place\\B;\nuse const Some\\Place\\C as D;\n",
			"<?php\n\nuse const Some\\Place\\{A, B, C as D};\n"},
		{"long namespace", nil,
			"<?php\nuse Foo\\Bar;\nuse Foo\\Baz\\Lorem\\Ipsum\\Lets\\Write\\Some\\More\\Strings\\One;\nuse Foo\\Baz\\Lorem\\Ipsum\\Lets\\Write\\Some\\More\\Strings\\Two;\n",
			"<?php\nuse Foo\\Bar;\nuse Foo\\Baz\\Lorem\\Ipsum\\Lets\\Write\\Some\\More\\Strings\\{One, Two};\n"},
		{"nested namespaces", nil,
			"<?php\nuse Foo\\Bar;\nuse Foo\\Baz;\nuse Foo\\Baz\\John\\Doe;\nuse Foo\\Baz\\John\\Smith;\nuse Foo\\Baz\\John\\Smith\\Junior;\nuse Foo\\Baz\\Johnny\\DoeSecond;\nuse Foo\\Baz\\Johnny\\SmithSecond;\n",
			"<?php\nuse Foo\\{Bar, Baz};\nuse Foo\\Baz\\John\\{Doe, Smith};\nuse Foo\\Baz\\Johnny\\{DoeSecond, SmithSecond};\nuse Foo\\Baz\\John\\Smith\\Junior;\n"},
		{"real class names", nil,
			"<?php\nuse PhpCsFixer\\Tokenizer\\AbstractTransformer;\nuse PhpCsFixer\\Tokenizer\\CT;\nuse PhpCsFixer\\Tokenizer\\Token;\nuse PhpCsFixer\\Tokenizer\\Tokens;\n",
			"<?php\nuse PhpCsFixer\\Tokenizer\\{AbstractTransformer, CT, Token, Tokens};\n"},
		{"no whitespace between statements", nil,
			"<?php\nuse Foo\\Bar;use Foo\\Baz;\n",
			"<?php\nuse Foo\\{Bar, Baz};\n"},
		{"code right after imports", nil,
			"<?php\nuse Foo\\Bar;use Foo\\Baz;\\DontTouch::me();",
			"<?php\nuse Foo\\{Bar, Baz};\n\\DontTouch::me();"},
		{"single global classes kept", nil,
			"<?php\n\nuse Foo\\Bar;\nuse Foo\\Baz;\nuse ReflectionClass;\nuse ReflectionMethod;\n",
			"<?php\n\nuse Foo\\{Bar, Baz};\nuse ReflectionClass;\nuse ReflectionMethod;\n"},
		{"leading slash global kept", nil,
			"<?php\n\nuse Foo\\Bar;\nuse Foo\\Baz;\nuse \\ReflectionClass;\nuse \\ReflectionMethod;\n",
			"<?php\n\nuse Foo\\{Bar, Baz};\nuse \\ReflectionClass;\nuse \\ReflectionMethod;\n"},
		{"many groups with one single", nil,
			"<?php\n\nuse Framework\\Database\\ORM\\Model;\nuse Framework\\Database\\ORM\\Relations\\BelongsTo;\nuse Framework\\Database\\ORM\\Relations\\HasOne;\nuse Framework\\Database\\ORM\\SoftDeletes;\nuse Framework\\Database\\Query\\JoinClause;\nuse Framework\\Notifications\\Notifiable;\nuse Framework\\Notifications\\Notification;\nuse Framework\\Support\\Arr;\nuse Framework\\Support\\Collection;\nuse Framework\\Support\\Facades\\DB;\nuse Framework\\Support\\Facades\\Log;\n",
			"<?php\n\nuse Framework\\Database\\Query\\JoinClause;\nuse Framework\\Support\\{Arr, Collection};\nuse Framework\\Database\\ORM\\{Model, SoftDeletes};\nuse Framework\\Notifications\\{Notifiable, Notification};\nuse Framework\\Support\\Facades\\{DB, Log};\nuse Framework\\Database\\ORM\\Relations\\{BelongsTo, HasOne};\n"},
		{"equal length namespaces", nil,
			"<?php\n\nuse Framework\\Foo\\Class1;\nuse Framework\\Foo\\Class2;\nuse Framework\\Bar\\Class3;\nuse Framework\\Bar\\Class4;\nuse Framework\\Bar\\Class5;\nuse Framework\\Baz\\Class6;\nuse Framework\\Foo\\Class7;\n",
			"<?php\n\nuse Framework\\Bar\\{Class3, Class4, Class5};\nuse Framework\\Foo\\{Class1, Class2, Class7};\nuse Framework\\Baz\\Class6;\n"},
		{"class group and single function", nil,
			"<?php\n\nuse Foo\\Bar;\nuse function Foo\\baz;\nuse Foo\\Baz;\n",
			"<?php\n\nuse Foo\\{Bar, Baz};\nuse function Foo\\baz;\n"},
		{"classes and functions grouped", nil,
			"<?php\n\nuse Foo\\Bar\\Baz\\A;\nuse Foo\\Bar\\Baz\\B;\nuse Foo\\Bar\\C;\nuse Foo\\D;\nuse Foo\\E;\nuse function Foo\\a;\nuse function Foo\\b;\n",
			"<?php\n\nuse Foo\\Bar\\C;\nuse Foo\\{D, E};\nuse Foo\\Bar\\Baz\\{A, B};\nuse function Foo\\{a, b};\n"},
		{"only class-like", classy,
			"<?php\n\nuse Foo\\Bar\\Baz\\A;\nuse Foo\\Bar\\Baz\\B;\nuse Foo\\Bar\\C;\nuse Foo\\D;\nuse Foo\\E;\nuse function Foo\\a;\nuse function Foo\\b;\nuse const Foo\\X;\nuse const Foo\\Y;\n",
			"<?php\n\nuse Foo\\Bar\\C;\nuse Foo\\{D, E};\nuse Foo\\Bar\\Baz\\{A, B};\nuse function Foo\\a;\nuse function Foo\\b;\nuse const Foo\\X;\nuse const Foo\\Y;\n"},
		{"only functions", functions,
			"<?php\n\nuse Foo\\Bar\\Baz\\A;\nuse Foo\\Bar\\Baz\\B;\nuse function Foo\\a;\nuse function Foo\\b;\nuse Foo\\Bar\\C;\nuse Foo\\D;\nuse Foo\\E;\nuse const Foo\\X;\nuse const Foo\\Y;\n",
			"<?php\n\nuse Foo\\Bar\\Baz\\A;\nuse Foo\\Bar\\Baz\\B;\nuse function Foo\\{a, b};\nuse Foo\\Bar\\C;\nuse Foo\\D;\nuse Foo\\E;\nuse const Foo\\X;\nuse const Foo\\Y;\n"},
		{"only constants", constants,
			"<?php\n\nuse Foo\\Bar\\Baz\\A;\nuse Foo\\Bar\\Baz\\B;\nuse function Foo\\a;\nuse function Foo\\b;\nuse Foo\\Bar\\C;\nuse Foo\\D;\nuse Foo\\E;\nuse const Foo\\X;\nuse const Foo\\Y;\n",
			"<?php\n\nuse Foo\\Bar\\Baz\\A;\nuse Foo\\Bar\\Baz\\B;\nuse function Foo\\a;\nuse function Foo\\b;\nuse Foo\\Bar\\C;\nuse Foo\\D;\nuse Foo\\E;\nuse const Foo\\{X, Y};\n"},
		{"class-likes and functions", classyFunctions,
			"<?php\n\nuse Foo\\Bar\\Baz\\A;\nuse Foo\\Bar\\Baz\\B;\nuse function Foo\\a;\nuse function Foo\\b;\nuse Foo\\Bar\\C;\nuse Foo\\D;\nuse Foo\\E;\nuse const Foo\\X;\nuse const Foo\\Y;\n",
			"<?php\n\nuse function Foo\\{a, b};\nuse Foo\\Bar\\C;\nuse Foo\\{D, E};\nuse Foo\\Bar\\Baz\\{A, B};\nuse const Foo\\X;\nuse const Foo\\Y;\n"},
		{"class-likes and constants", classyConstants,
			"<?php\n\nuse Foo\\Bar\\Baz\\A;\nuse Foo\\Bar\\Baz\\B;\nuse function Foo\\a;\nuse function Foo\\b;\nuse Foo\\Bar\\C;\nuse Foo\\D;\nuse Foo\\E;\nuse const Foo\\X;\nuse const Foo\\Y;\n",
			"<?php\n\nuse function Foo\\a;\nuse function Foo\\b;\nuse Foo\\Bar\\C;\nuse Foo\\{D, E};\nuse Foo\\Bar\\Baz\\{A, B};\nuse const Foo\\{X, Y};\n"},
		{"functions and constants", functionsConstants,
			"<?php\n\nuse Foo\\Bar\\Baz\\A;\nuse Foo\\Bar\\Baz\\B;\nuse function Foo\\a;\nuse function Foo\\b;\nuse Foo\\Bar\\C;\nuse Foo\\D;\nuse Foo\\E;\nuse const Foo\\X;\nuse const Foo\\Y;\n",
			"<?php\n\nuse Foo\\Bar\\Baz\\A;\nuse Foo\\Bar\\Baz\\B;\nuse function Foo\\{a, b};\nuse Foo\\Bar\\C;\nuse Foo\\D;\nuse Foo\\E;\nuse const Foo\\{X, Y};\n"},
	})
}
