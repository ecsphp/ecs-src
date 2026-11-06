package rules

import "testing"

// More ReturnAssignmentFixerTest cases, extracted verbatim from upstream.
func TestReturnAssignmentPhpCsFixerBlockCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, ReturnAssignment{}, []aliasUpstreamCase{
		{
			`comprehensive nested closures`,
			nil,
			`<?php
function A($a0,$a1,$a2,$d)
{
    if ($a0) {
        $b = 1;
        return $b; // fix me
    }

    if ($a1) {
        $c = 2;
        return $c; // fix me
    }

    $nested0 = function() {
        global $a;

        ++$a;
        $d = 2;

        $nested1 = function () use ($d) {
            if ($d) {
                $f = 3;
                return $f; // fix me
            }

            $nested2 = function (&$d) {
                if ($d) {
                    $f = 1;
                    return $f; // fix me not
                }

                $d = function () {
                    $a = 4;
                    return $a; // fix me
                };

                if ($d+1) {
                    $f = 1;
                    return $f; // fix me not
                }
            };

            return $nested2();
        };

        return $a; // fix me not
    };

    if ($a2) {
        $d = 5;
        return $d; // fix me
    }
}

function B($b0, $b1, $b2)
{
    if ($b0) {
        $b = 10;
        return $b; // fix me
    }

    if ($b1) {
        $c = 20;
        return $c; // fix me
    }

    if ($b2) {
        $d = 30;
        return $d; // fix me
    }
}
`,
			`<?php
function A($a0,$a1,$a2,$d)
{
    if ($a0) {
        return 1;
          // fix me
    }

    if ($a1) {
        return 2;
          // fix me
    }

    $nested0 = function() {
        global $a;

        ++$a;
        $d = 2;

        $nested1 = function () use ($d) {
            if ($d) {
                return 3;
                  // fix me
            }

            $nested2 = function (&$d) {
                if ($d) {
                    $f = 1;
                    return $f; // fix me not
                }

                $d = function () {
                    return 4;
                      // fix me
                };

                if ($d+1) {
                    $f = 1;
                    return $f; // fix me not
                }
            };

            return $nested2();
        };

        return $a; // fix me not
    };

    if ($a2) {
        return 5;
          // fix me
    }
}

function B($b0, $b1, $b2)
{
    if ($b0) {
        return 10;
          // fix me
    }

    if ($b1) {
        return 20;
          // fix me
    }

    if ($b2) {
        return 30;
          // fix me
    }
}
`,
		},
		{
			`comments with leading space`,
			nil,
			`<?php
                    function A()
                    { #1
 #2
 $a #3
 #4
 = #5
 #6
 15 #7
 ; #8
 #9
 return #10
 #11
 $a  #12
 #13
 ;  #14
 #15
                    }
                `,
			`<?php
                    function A()
                    { #1
 #2
 return #3
 #4
  #5
 #6
 15 #7
 ; #8
 #9
  #10
 #11
   #12
 #13
   #14
 #15
                    }
                `,
		},
		{
			`hash comments`,
			nil,
			`<?php
                    function A()
                    {#1
#2                    `,
			`<?php
                    function A()
                    {#1
#2                    `,
		},
		{
			`case-insensitive var name`,
			nil,
			`<?php
                    function a()
                    {
                        $A = $c + 1; /*
var names are case-insensitive */ return $a   ;}
                `,
			`<?php
                    function a()
                    {
                        return $c + 1; /*
var names are case-insensitive */ }
                `,
		},
		{
			`multiple try/catch separated by conditional return`,
			nil,
			`<?php
                function foo()
                {
                    try {
                        $result = getResult();
                        return $result;
                    } catch (\Throwable $exception) {
                        error_log($exception->getMessage());
                    }

                    if (isSomeCondition()) {
                        $result = getSomeResult();
                        return $result;
                    }

                    try {
                        $result = $a + $b;
                        return $result;
                    } catch (\Throwable $th) {
                        var_dump($result ?? null);
                    }
                }
                `,
			`<?php
                function foo()
                {
                    try {
                        return getResult();
                    } catch (\Throwable $exception) {
                        error_log($exception->getMessage());
                    }

                    if (isSomeCondition()) {
                        return getSomeResult();
                    }

                    try {
                        $result = $a + $b;
                        return $result;
                    } catch (\Throwable $th) {
                        var_dump($result ?? null);
                    }
                }
                `,
		},
		{
			`multiple try/catch with finally`,
			nil,
			`<?php
                function foo()
                {
                    try {
                        $result = getResult();
                        return $result;
                    } catch (\Throwable $exception) {
                        error_log($exception->getMessage());
                    }

                    if (isSomeCondition()) {
                        $result = getSomeResult();
                        return $result;
                    }

                    try {
                        $result = $a + $b;
                        return $result;
                    } catch (\Throwable $th) {
                        throw $th;
                    } finally {
                        echo "result:", $result, \PHP_EOL;
                    }
                }
                `,
			`<?php
                function foo()
                {
                    try {
                        return getResult();
                    } catch (\Throwable $exception) {
                        error_log($exception->getMessage());
                    }

                    if (isSomeCondition()) {
                        return getSomeResult();
                    }

                    try {
                        $result = $a + $b;
                        return $result;
                    } catch (\Throwable $th) {
                        throw $th;
                    } finally {
                        echo "result:", $result, \PHP_EOL;
                    }
                }
                `,
		},
		{
			`try/catch/finally with comments no change`,
			nil,
			`<?php
                function add($a, $b): mixed
                {
                    try {
                        $result = $a + $b;

                        return $result;
                    } /* foo */ catch /** bar */ (\LogicException $th) {
                        throw $th;
                    }
                    // Or maybe this....
                    catch (\RuntimeException $th) {
                        throw $th;
                    }
                    # Print the result anyway
                    finally {
                        echo 'result:', $result, \PHP_EOL;
                    }
                }
                `,
			`<?php
                function add($a, $b): mixed
                {
                    try {
                        $result = $a + $b;

                        return $result;
                    } /* foo */ catch /** bar */ (\LogicException $th) {
                        throw $th;
                    }
                    // Or maybe this....
                    catch (\RuntimeException $th) {
                        throw $th;
                    }
                    # Print the result anyway
                    finally {
                        echo 'result:', $result, \PHP_EOL;
                    }
                }
                `,
		},
		{
			`try with multiple catch blocks no change`,
			nil,
			`<?php
                function foo() {
                    try {
                        $bar = bar();

                        return $bar;
                    } catch (\LogicException $e) {
                        echo "catch ... ";
                    } catch (\RuntimeException $e) {
                        echo $bar;
                    }
                }`,
			`<?php
                function foo() {
                    try {
                        $bar = bar();

                        return $bar;
                    } catch (\LogicException $e) {
                        echo "catch ... ";
                    } catch (\RuntimeException $e) {
                        echo $bar;
                    }
                }`,
		},
		{
			`while loop keeps assignment`,
			nil,
			`<?php
                function F() {
                    $a = 1;

                    while(bar()) {
                        ++$a;
                    }; // keep this

                    return $a;
                }
                `,
			`<?php
                function F() {
                    $a = 1;

                    while(bar()) {
                        ++$a;
                    }; // keep this

                    return $a;
                }
                `,
		},
	})
}
