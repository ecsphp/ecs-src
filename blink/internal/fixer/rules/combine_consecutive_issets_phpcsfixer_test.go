package rules

import "testing"

// Cases ported from PHP-CS-Fixer's CombineConsecutiveIssetsFixerTest
// (tests/Fixer/LanguageConstruct); every expected output is what PHP-CS-Fixer
// produces for the input.
func TestCombineConsecutiveIssetsPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	runPortConstructsCases(t, CombineConsecutiveIssets{}, []portConstructsCase{
		{"two", nil, "<?php $a = isset($a) && isset($b);", "<?php $a = isset($a, $b)  ;"},
		{"two with multi arg", nil, "<?php $a = isset($a) && isset($b,$c);", "<?php $a = isset($a, $b,$c)  ;"},
		{"both multi arg", nil, "<?php $a = isset($a,$c) && isset($b,$c);", "<?php $a = isset($a,$c, $b,$c)  ;"},
		{"first multi arg", nil, "<?php $a = isset($a,$c) && isset($b);", "<?php $a = isset($a,$c, $b)  ;"},
		{"or chain", nil, "<?php $a = isset($a) && isset($b) || isset($c) && isset($e)?>", "<?php $a = isset($a, $b)   || isset($c, $e)  ?>"},
		{"ternary arg", nil, "<?php $a = isset($a[a() ? b() : $d]) && isset($b);", "<?php $a = isset($a[a() ? b() : $d], $b)  ;"},
		{"comment arg", nil, "<?php $a = isset($a[$b]/**/) && isset($b);", "<?php $a = isset($a[$b], $b/**/)  ;"},
		{"inner spaces", nil, "<?php $a = isset ( $a /*1*/ )    &&    isset ( $c   ) && isset( $d );", "<?php $a = isset ( $a, $c, $d /*1*/ )                 ;"},
		{"minimal fix case", nil, "<?php {{isset($a)&&isset($b);}}", "<?php {{isset($a, $b);}}"},
		{"inside call", nil, "<?php foo(isset($a) && isset($b) && isset($c));", "<?php foo(isset($a, $b, $c)    );"},
		{"not followed by isset", nil, "<?php isset($a) && isset($b) && !isset($c) ?>", "<?php isset($a, $b)   && !isset($c) ?>"},
		{"long chain", nil, "<?php $a = isset($a,$c) && isset($b,$c) && isset($b,$c,$d,$f) && isset($b);", "<?php $a = isset($a,$c, $b,$c, $b,$c,$d,$f, $b)      ;"},
		{"comments", nil, `<?php

$a =#0
isset#1
(#2
$a#3
)#4
&&#5
isset
 #6
 #7
 ( #8
 $b #9
 /*10*/,     $c/**11
*/
)&& isset($d)
;`, "<?php\n\n$a =#0\nisset#1\n(#2\n$a, $b,$c, $d#3\n)#4\n#5\n\n #6\n #7\n  #8\n  #9\n /*10*/     /**11\n*/\n \n;"},
		{"spread over echos", nil, `<?php
                    $a = isset($a) && isset($b) && isset($c) && isset($d) && isset($e) && isset($f);
                    echo 1; echo 1; echo 1; echo 1; echo 1; echo 1; echo 1;
                    echo 1; echo 1; echo 1; echo 1; echo 1; echo 1; echo 1;
                    echo 1; echo 1; echo 1; echo 1; echo 1; echo 1; echo 1;
                    echo 1; echo 1; echo 1; echo 1; echo 1; echo 1; echo 1;
                    $a = isset($a) && isset($b);
                `, `<?php
                    $a = isset($a, $b, $c, $d, $e, $f)          ;
                    echo 1; echo 1; echo 1; echo 1; echo 1; echo 1; echo 1;
                    echo 1; echo 1; echo 1; echo 1; echo 1; echo 1; echo 1;
                    echo 1; echo 1; echo 1; echo 1; echo 1; echo 1; echo 1;
                    echo 1; echo 1; echo 1; echo 1; echo 1; echo 1; echo 1;
                    $a = isset($a, $b)  ;
                `},
		{"or precedence", nil, "<?php $d = isset($z[1]) && isset($z[2]) && isset($z[3]) || false;", "<?php $d = isset($z[1], $z[2], $z[3])     || false;"},
		{"comparison precedence", nil, `<?php
                    $a = isset($a) && isset($b) && isset($c) === false;
                    $a = isset($a) && isset($b) && isset($c) | false;
                    $a = isset($a) && isset($b) && isset($c) ^ false;
                `, `<?php
                    $a = isset($a, $b)   && isset($c) === false;
                    $a = isset($a, $b)   && isset($c) | false;
                    $a = isset($a, $b)   && isset($c) ^ false;
                `},
		{"method isset", nil, "<?php $a = isset($a) && $a->isset(); $b=isset($d);", "<?php $a = isset($a) && $a->isset(); $b=isset($d);"},
		{"negated issets", nil, `<?php
                    $a = !isset($a) && isset($b);
                    $a = !isset($a) && !isset($b);
                    $a = isset($a) && !isset($b);
                    //
                    $a = isset($b) && isset($c) === false;
                    $a = isset($b) && isset($c) | false;
                    $a = isset($b) && isset($c) ^ false;
                    //
                    $a = false === isset($b) && isset($c);
                    $a = false | isset($b) && isset($c);
                    $a = false ^ isset($b) && isset($c);
                `, `<?php
                    $a = !isset($a) && isset($b);
                    $a = !isset($a) && !isset($b);
                    $a = isset($a) && !isset($b);
                    //
                    $a = isset($b) && isset($c) === false;
                    $a = isset($b) && isset($c) | false;
                    $a = isset($b) && isset($c) ^ false;
                    //
                    $a = false === isset($b) && isset($c);
                    $a = false | isset($b) && isset($c);
                    $a = false ^ isset($b) && isset($c);
                `},
		{"mixed negation chain", nil, "<?php $a = !isset($container[$a]) && isset($container[$b]) && !isset($container[$c]) && isset($container[$d]);", "<?php $a = !isset($container[$a]) && isset($container[$b]) && !isset($container[$c]) && isset($container[$d]);"},
		{"anonymous class", nil, `<?php
                class A {function isset(){}} // isset($b) && isset($c)
                $a = new A(); /** isset($b) && isset($c) */
                if (isset($b) && $a->isset()) {}
            `, `<?php
                class A {function isset(){}} // isset($b) && isset($c)
                $a = new A(); /** isset($b) && isset($c) */
                if (isset($b) && $a->isset()) {}
            `},
	})
}
