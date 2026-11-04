package rules

import "testing"

// Cases ported from PHP-CS-Fixer's CombineConsecutiveUnsetsFixerTest
// (tests/Fixer/LanguageConstruct); every expected output is what PHP-CS-Fixer
// produces for the input.
func TestCombineConsecutiveUnsetsPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	runPortConstructsCases(t, CombineConsecutiveUnsets{}, []portConstructsCase{
		{"CombineConsecutiveUnsetsFixer #0", nil, "<?php //1\n                    unset($foo/*;*/);\n                    unset(/*;*/$bar, $c ); //test\n                    unset($foobar  ); /* more comment test*/\n                    unset(  $foobar2);\n                ", "<?php //1\n                    unset($foo/*;*/, /*;*/$bar, $c , $foobar  ,  $foobar2);\n                     //test\n                     /* more comment test*/\n                    \n                "},
		{"CombineConsecutiveUnsetsFixer #1", nil, "<?php unset($d );/*unset(    $d2);unset($e   );;*/    uNseT($e);", "<?php unset($d , $e);/*unset(    $d2);unset($e   );;*/    "},
		{"CombineConsecutiveUnsetsFixer #2", nil, "<?php UNSET($a); unset($b,$c/**/);", "<?php UNSET($a, $b,$c/**/); "},
		{"CombineConsecutiveUnsetsFixer #3", nil, "<?php\n              $config = array();\n              if ($config) {\n              }\n              unset($config['autoescape_service'], $config['autoescape_service_method']);\n              ", "<?php\n              $config = array();\n              if ($config) {\n              }\n              unset($config['autoescape_service'], $config['autoescape_service_method']);\n              "},
		{"CombineConsecutiveUnsetsFixer #4", nil, "<?php //2\n                    unset($foo);/*1*/\n                    unset($bar);/*2*/\n                    unset($foobar);//3\n                    unset($foobar2);/*4*/\n                    /*5*/ unset($foobar3);\n                ", "<?php //2\n                    unset($foo, $bar, $foobar, $foobar2, $foobar3);/*1*/\n                    /*2*/\n                    //3\n                    /*4*/\n                    /*5*/ \n                "},
		{"CombineConsecutiveUnsetsFixer #5", nil, "<?php\n                    unset($foo3);\n                        /* c1 */\n                        unset($bar);\n                \n                // c2\n                unset($test,$test1);\n                ", "<?php\n                    unset($foo3, $bar, $test,$test1);\n                        /* c1 */\n                        \n                \n                // c2\n                \n                "},
		{"CombineConsecutiveUnsetsFixer #6", nil, "<?php unset($x);  /**/ unset ($b  , $d) ?> b", "<?php unset($x, $b  , $d);  /**/   ?> b"},
		{"CombineConsecutiveUnsetsFixer #7", nil, "<?php unset($x) ?>", "<?php unset($x) ?>"},
		{"CombineConsecutiveUnsetsFixer #8", nil, "<?php unset($y);unset($u) ?>", "<?php unset($y, $u); ?>"},
		{"CombineConsecutiveUnsetsFixer #9", nil, "<?php\n                    unset($a[0]);\n                    unset($a['a']);\n                    unset($a[\"b\"]);\n                    unset($a->b);\n                    unset($a->b->c);\n                    unset($a->b[0]->c['a']);\n                ", "<?php\n                    unset($a[0], $a['a'], $a[\"b\"], $a->b, $a->b->c, $a->b[0]->c['a']);\n                    \n                    \n                    \n                    \n                    \n                "},
		{"CombineConsecutiveUnsetsFixer #10", nil, "<?php\n                unset(\n                    $array1[\"foo\"],\n                );\n\n                unset(\n                    $array2[\"bar2\"],\n                );", "<?php\n                unset(\n                    $array1[\"foo\"],\n                    $array2[\"bar2\"],\n                );\n\n                "},
		{"CombineConsecutiveUnsetsFixer #11", nil, "<?php\n                unset($array1[\"foo\"],);\n\n                unset($array2[\"bar2\"],);", "<?php\n                unset($array1[\"foo\"],$array2[\"bar2\"],);\n\n                "},
	})
}
