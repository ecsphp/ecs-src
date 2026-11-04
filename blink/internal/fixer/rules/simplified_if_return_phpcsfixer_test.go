package rules

import "testing"

// Cases ported from PHP-CS-Fixer's SimplifiedIfReturnFixerTest; every expected
// output is what PHP-CS-Fixer produces for the input.
func TestSimplifiedIfReturnPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, SimplifiedIfReturn{}, []aliasUpstreamCase{
		{"simple", nil, "<?php if ($foo) { return true; } return false;", "<?php return (bool) ($foo);"},
		{"simple-negative", nil, "<?php if ($foo) { return false; } return true;", "<?php return ! ($foo);"},
		{"simple-negative II", nil, "<?php if (!$foo && $a()) { return false; } return true;", "<?php return ! (!$foo && $a());"},
		{"simple-braceless", nil, "<?php if ($foo) return true; return false;", "<?php return (bool) ($foo);"},
		{"simple-braceless-negative", nil, "<?php if ($foo) return false; return true;", "<?php return ! ($foo);"},
		{"bug-consecutive-ifs", nil, "<?php if ($bar) { return 1; } if ($foo) { return true; } return false;", "<?php if ($bar) { return 1; } return (bool) ($foo);"},
		{"bug-consecutive-ifs-braceless", nil, "<?php if ($bar) return 1; if ($foo) return true; return false;", "<?php if ($bar) return 1; return (bool) ($foo);"},
		{"else-if", nil, "<?php if ($bar) { return $bar; } else if ($foo) { return true; } return false;", "<?php if ($bar) { return $bar; } else return (bool) ($foo);"},
		{"else-if-negative", nil, "<?php if ($bar) { return $bar; } else if ($foo) { return false; } return true;", "<?php if ($bar) { return $bar; } else return ! ($foo);"},
		{"elseif", nil, "<?php if ($bar) { return $bar; } elseif ($foo) { return true; } return false;", "<?php if ($bar) { return $bar; } return (bool) ($foo);"},
		{"elseif-braceless", nil, "<?php if ($bar) return $bar; elseif ($foo) return true; return false;", "<?php if ($bar) return $bar; return (bool) ($foo);"},
		{"multiple-consecutive-ifs", nil, "<?php if ($a) { return 1; } if ($b) { return true; } return false; if ($c) { return 3; }", "<?php if ($a) { return 1; } return (bool) ($b); if ($c) { return 3; }"},
		{"complex-condition-with-parentheses", nil, "<?php if (($foo && $bar)) { return true; } return false;", "<?php return (bool) (($foo && $bar));"},
		{"method-call-condition", nil, "<?php if ($obj->isValid()) { return true; } return false;", "<?php return (bool) ($obj->isValid());"},
		{"function-call-condition-negative", nil, "<?php if (empty($foo)) { return false; } return true;", "<?php return ! (empty($foo));"},
		{"array-access-condition", nil, "<?php if ($arr[0]) { return true; } return false;", "<?php return (bool) ($arr[0]);"},
		{"static-property-condition", nil, "<?php if (self::$flag) { return true; } return false;", "<?php return (bool) (self::$flag);"},
		{"instance-property-condition-negative", nil, "<?php if ($this->enabled) { return false; } return true;", "<?php return ! ($this->enabled);"},
		{"spaces-around-condition", nil, "<?php if ( $foo ) { return true; } return false;", "<?php return (bool) ( $foo );"},
		{"multiple-spaces-after-if", nil, "<?php if  (  $foo  ) { return true; } return false;", "<?php return (bool)  (  $foo  );"},
		{"ternary-not-fixed", nil, "<?php return $foo ? true : false;", "<?php return $foo ? true : false;"},
		{"single-return-without-braces-not-fixed", nil, "<?php if ($foo) return true;", "<?php if ($foo) return true;"},
		{"single-return-not-fixed", nil, "<?php if ($foo) { return true; }", "<?php if ($foo) { return true; }"},
		{"different-values-not-fixed", nil, "<?php if ($foo) { return 1; } return 2;", "<?php if ($foo) { return 1; } return 2;"},
		{"null-not-fixed", nil, "<?php if ($foo) { return true; } return null;", "<?php if ($foo) { return true; } return null;"},
		{"2x-true-not-fixed", nil, "<?php if ($foo) { return true; } return true;", "<?php if ($foo) { return true; } return true;"},
		{"2x-false-not-fixed", nil, "<?php if ($foo) { return false; } return false;", "<?php if ($foo) { return false; } return false;"},
		{"alternative syntax not supported", nil, "<?php if ($foo):\nreturn true;\nelse:\nreturn false;\nendif;", "<?php if ($foo):\nreturn true;\nelse:\nreturn false;\nendif;"},
	})
}
