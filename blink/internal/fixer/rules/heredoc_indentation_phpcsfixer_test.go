package rules

import "testing"

// Cases ported from PHP-CS-Fixer's HeredocIndentationFixerTest
// (tests/Fixer/Whitespace); every expected output is what PHP-CS-Fixer
// produces for the input. testWithWhitespacesConfig is left out: blink always
// indents with four spaces.
func TestHeredocIndentationPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, HeredocIndentation{}, []aliasUpstreamCase{
		{"case 0", nil,
			`<?php
    foo(<<<EOD
EOD
    );`,
			`<?php
    foo(<<<EOD
        EOD
    );`},
		{"case 1", nil,
			`<?php
    foo(<<<EOD

EOD
    );`,
			`<?php
    foo(<<<EOD

        EOD
    );`},
		{"case 2", nil,
			`<?php
    foo(<<<EOD
abc

    def
EOD
    );`,
			`<?php
    foo(<<<EOD
        abc

            def
        EOD
    );`},
		{"case 3", nil,
			`<?php
    foo(<<<'EOD'

abc
    def

EOD
    );`,
			`<?php
    foo(<<<'EOD'

        abc
            def

        EOD
    );`},
		{"case 4", nil,
			`<?php
    foo(<<<'EOD'
            abc
                def
            EOD
    );`,
			`<?php
    foo(<<<'EOD'
        abc
            def
        EOD
    );`},
		{"case 5", nil,
			`<?php
    foo(<<<EOD
$abc
    $def
{$ghi}
EOD
    );`,
			`<?php
    foo(<<<EOD
        $abc
            $def
        {$ghi}
        EOD
    );`},
		{"case 6", nil,
			`<?php
    $a = <<<'EOD'
<?php
    $b = <<<FOO
abc
FOO;
EOD;`,
			`<?php
    $a = <<<'EOD'
        <?php
            $b = <<<FOO
        abc
        FOO;
        EOD;`},
		{"case 7", nil,
			`
<?php
    foo(<<<EOD
        
      abc
        
      def
        
      EOD
    );`,
			`
<?php
    foo(<<<EOD
          
        abc
          
        def
          
        EOD
    );`},
		{"case 8", nil,
			`
<?php
    foo(<<<EOD
  
      abc
  
      def
  
      EOD
    );`,
			`
<?php
    foo(<<<EOD

        abc

        def

        EOD
    );`},
		{"case 9", nil,
			`<?php foo(<<<EOD
EOD
);`,
			`<?php foo(<<<EOD
    EOD
);`},
		{"case 10", map[string]any{"indentation": "same_as_start"},
			`<?php
    foo(<<<EOD
abc

    def
EOD
    );`,
			`<?php
    foo(<<<EOD
    abc

        def
    EOD
    );`},
		{"case 11", map[string]any{"indentation": "same_as_start"},
			`<?php
    foo(<<<EOD
        abc

            def
        EOD
    );`,
			`<?php
    foo(<<<EOD
    abc

        def
    EOD
    );`},
	})
}
