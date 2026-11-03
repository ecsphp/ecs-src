package rules

import (
	"strings"
	"testing"
)

// Cases ported from PHP-CS-Fixer's tests/Fixer/ControlStructure/NoUselessElseFixerTest.php (v3.95.27).
func TestNoUselessElseUpstream(t *testing.T) {
	t.Parallel()
	runUpstreamCases(t, NoUselessElse{}, noUselessElseUpstreamCases)
}

var noUselessElseUpstreamCases = []upstreamCase{
	{"provideFixCases#0", nil, "<?php\n                    if (true) {\n                        $b = $a > 2 ? \"\" : die\n                        ?>\n                    <?php\n                    } else {\n                        echo 798;\n                    }", ""},
	{"provideFixCases#1", nil, "<?php\n                    if (true) {\n                        $b = $a > 2 ? \"\" : die\n                        ?>\n                    <?php ; // useless semicolon case\n                    } else {\n                        echo 798;\n                    }", ""},
	{"provideFixCases#2", nil, "<?php\n                    if (true) {\n                        if($a) die\n                        ?>\n                    <?php ; // useless semicolon case\n                    } else {\n                        echo 798;\n                    }", ""},
	{"provideFixCases#3", nil, "<?php\n                    if (true) {\n                        echo 1;\n                        ?>\n                    <?php ; // useless semicolon case\n                    } else {\n                        echo 798;\n                    }", ""},
	{"provideFixCases#4", nil, "<?php\n                    if (true) {\n                        echo 777;\n                        if(false) die ?>\n                    <?php\n                    } else {\n                        echo 778;\n                    }", ""},
	{"provideFixCases#5", nil, "<?php\n                    if (true)\n                        echo 3;\n                    else {\n                        ?><?php\n                        echo 4;\n                    }\n                ", ""},
	{"provideFixCases#6", nil, "<?php\n                    if (true)\n                        echo 3;\n                    \n                    ?><?php\n                echo 4;\n                ", "<?php\n                    if (true)\n                        echo 3;\n                    else\n                    ?><?php\n                echo 4;\n                "},
	{"provideFixCases#7", nil, "<?php\nif (true)\n    echo 4;\n?><?php echo 5;", "<?php\nif (true)\n    echo 4;\nelse?><?php echo 5;"},
	{"provideFixCases#8", nil, "<?php\n                if ($a)\n                    echo 1789;\n                else if($b)\n                    echo 256;\n                elseif($c)\n                    echo 3;\n                    if ($a) {\n\n                    }elseif($d) {\n                        return 1;\n                    }\n                else\n                    echo 4;\n            ", ""},
	{"provideFixCases#9", nil, "<?php\n                if ($a)\n                    echo 1789;\n                else if($b) {\n                    echo 256;\n                } elseif($c) {\n                    echo 3;\n                    if ($d) {\n                        echo 4;\n                    } elseif($e)\n                        return 1;\n                } else\n                    echo 4;\n            ", ""},
	{"provideFixCases#10", nil, "<?php\n                if ($a) {\n                    GOTO jump;\n                }  \n                    echo 1789;\n                \n\n                jump:\n            ", "<?php\n                if ($a) {\n                    GOTO jump;\n                } else {\n                    echo 1789;\n                }\n\n                jump:\n            "},
	{"provideFixCases/nested if", nil, "<?php\n                    if ($x) {\n                        if ($y) {\n                            return 1;\n                        }  \n                            return 2;\n                        \n                    }  \n                        return 3;\n                    \n                ", "<?php\n                    if ($x) {\n                        if ($y) {\n                            return 1;\n                        } else {\n                            return 2;\n                        }\n                    } else {\n                        return 3;\n                    }\n                "},
	{"provideFixCases#12", nil, "<?php\n                    if (false)\n                        echo 1;\n                    \n                ", "<?php\n                    if (false)\n                        echo 1;\n                    else{}\n                "},
	{"provideFixCases#13", nil, "<?php if($a){}", "<?php if($a){}else{}"},
	{"provideFixCases#14", nil, "<?php if($a){ $a = ($b); }  ", "<?php if($a){ $a = ($b); } else {}"},
	{"provideFixCases#15", nil, "<?php if ($a) {;}   if ($a) {;}  /**/ if($a){}", "<?php if ($a) {;} else {} if ($a) {;} else {/**/} if($a){}else{}"},
	{"provideFixCases#16", nil, "<?php\n                    if /**/($a) /**/{ //\n                        /**/\n                        /**/return/**/1/**/;\n                        //\n                    }/**/  /**/\n                        /**/\n                        //\n                    /**/\n                ", "<?php\n                    if /**/($a) /**/{ //\n                        /**/\n                        /**/return/**/1/**/;\n                        //\n                    }/**/ else /**/{\n                        /**/\n                        //\n                    }/**/\n                "},
	{"provideFixCases#17", nil, "<?php\n                    if ($a) {\n                        if ($b) {\n                            if ($c) {\n                            } elseif ($d) {\n                                return;\n                            }  //\n                            //\n                            return;\n                        }  //\n                        //\n                        return;\n                    }  //\n                    //\n                ", "<?php\n                    if ($a) {\n                        if ($b) {\n                            if ($c) {\n                            } elseif ($d) {\n                                return;\n                            } else {//\n                            }//\n                            return;\n                        } else {//\n                        }//\n                        return;\n                    } else {//\n                    }//\n                "},
	{"provideFixCases#18", nil, "<?php\n                    if ($a0) {\n                        //\n                    } else {\n                        echo 0;\n                    }\n                ", ""},
	{"provideFixCases#19", nil, "<?php\n                    if (false)\n                        echo \"a\";\n                    else\n\n                    echo \"a\";\n                ", ""},
	{"provideFixCases#20", nil, "<?php if($a2){;} else {echo 27;}", ""},
	{"provideFixCases#21", nil, "<?php if ($a3) {test();} else {echo 3;}", ""},
	{"provideFixCases#22", nil, "<?php if ($a4) {$b = function () {};} else {echo 4;}", ""},
	{"provideFixCases#23", nil, "<?php if ($a5) {$b = function () use ($a){};} else {echo 5;}", ""},
	{"provideFixCases#24", nil, "<?php\n                    if ($a) {\n                        if ($b) return;\n                    } else {\n                        echo 1;\n                    }\n                ", ""},
	{"provideFixCases#25", nil, "<?php\n                    if ($a) {\n                        if ($b) throw new \\Exception();\n                    } else {\n                        echo 1;\n                    }\n                ", ""},
	{"provideFixCases#26", nil, "<?php\n                    if ($a) {\n                        if ($b) { throw new \\Exception(); }\n                    } else {\n                        echo 1;\n                    }\n                ", ""},
	{"provideFixCases#27", nil, "<?php\n                    $a = true; // 6\n                    if (true === $a)\n                        $b = true === $a ? 1 : die;\n                    else\n                        echo 40;\n\n                    echo \"end\";\n                ", ""},
	{"provideFixCases#28", nil, "<?php\n                    $a = true; // 6\n                    if (true === $a)\n                        $b = true === $a ? 1 : exit(1);\n                    else\n                        echo 40;\n\n                    echo \"end\";\n                ", ""},
	{"provideFixCases#29", nil, "<?php\n                    $a = true; // 6\n                    if (true === $a)\n                        $b = true === $a ? 1 : exit(1);\n                    else\n                        echo 4;\n\n                    echo \"end\";\n                ", ""},
	{"provideFixCases#30", nil, "<?php\n                    if (false)\n                        die;\n                    elseif (true)\n                        if(true)echo 777;else die;\n                    else if (true)\n                        die;\n                    elseif (false)\n                        die;\n                    else\n                        echo 7;\n                ", ""},
	{"provideFixCases#31", nil, "<?php\n                    $tmp = function($b){$b();};\n                    $a =1;\n                    return $tmp(function () use ($a) {\n                        if ($a) {\n                            $a++;\n                        } else {\n                            $a--;\n                        }\n                    });\n                ", ""},
	{"provideFixCases#32", nil, "<?php\n                    $tmp = function($b){$b();};\n                    $a =1;\n                    return $tmp(function () use ($a) {\n                        if ($a) {\n                            $a++;\n                        } elseif($a > 2) {\n                            return 1;\n                        } else {\n                            $a--;\n                        }\n                    });\n                ", ""},
	{"provideFixCases#33", nil, "<?php\n                    return function() {\n                        if (false) {\n\n                        } elseif (3 > 2) {\n\n                        } else {\n                            echo 1;\n                        }\n                    };", ""},
	{"provideFixCases#34", nil, "<?php\n                    return function() {\n                        if (false) {\n                            return 1;\n                        } elseif (3 > 2) {\n\n                        } else {\n                            echo 1;\n                        }\n                    };", ""},
	{"provideFixCases#35", nil, "<?php\n                if ($a === false)\n                {\n                    if ($v) { $ret = \"foo\"; if($d){return 1;}echo $a;}\n                }\n                else\n                    $ret .= $value;\n\n                return $ret;", "<?php\n                if ($a === false)\n                {\n                    if ($v) { $ret = \"foo\"; if($d){return 1;}else{echo $a;}}\n                }\n                else\n                    $ret .= $value;\n\n                return $ret;"},
	{"provideFixCases#36", nil, "<?php if ($a) {return null;}   return function () {}; ", "<?php if ($a) {return null;} else { return function () {}; }"},
	{"provideFixCases#37", nil, "<?php if ($a) {return null;}   return new class {}; ", "<?php if ($a) {return null;} else { return new class {}; }"},
	{"provideFixCases#38", nil, "<?php if ($a) {return;} else { foo(); function bar() {} }", ""},
	{"provideFixCases#39", nil, "<?php if ($a) {return;} else { abstract class Foo {} }", ""},
	{"provideFixCases#40", nil, "<?php if ($a) {return;} else { interface Foo {} }", ""},
}

// The provider cases PHP-CS-Fixer generates from templates (generateCases,
// generateConditionsWithoutBracesCase and the PHP 8 negative cases).
func TestNoUselessElseUpstreamGenerated(t *testing.T) {
	t.Parallel()
	var cases []upstreamCase
	fill := func(template, statement string) string { return strings.Replace(template, "%s", statement, 1) }
	generateCases := func(expected, input string) {
		for _, c := range nueUpstreamExitStatements {
			variants := []string{c, strings.ToUpper(c)}
			if c != strings.ToLower(c) {
				variants = append(variants, strings.ToLower(c))
			}
			for _, v := range variants {
				in := ""
				if input != "" {
					in = fill(input, v)
				}
				cases = append(cases, upstreamCase{"generateCases/" + v, nil, fill(expected, v), in})
			}
		}
	}
	conditionsWithoutBraces := func(statement string) {
		for _, tpl := range []string{nueUpstreamIfTemplate, nueUpstreamIfElseTemplate, nueUpstreamIfElseIfTemplate} {
			cases = append(cases, upstreamCase{"generateConditionsWithoutBracesCase/" + statement, nil, fill(tpl, statement), ""})
		}
	}
	generateCases("<?php\n                while(true) {\n                    while(true) {\n                        if ($provideFixIfElseIfElseCases) {\n                            return;\n                        } elseif($a1) {\n                            if ($b) {echo 1; die;}  echo 552;\n                            return 1;\n                        } elseif($b) {\n                            %s\n                        }  \n                            echo 662;\n                        \n                    }\n                }\n            ", "<?php\n                while(true) {\n                    while(true) {\n                        if ($provideFixIfElseIfElseCases) {\n                            return;\n                        } elseif($a1) {\n                            if ($b) {echo 1; die;} else {echo 552;}\n                            return 1;\n                        } elseif($b) {\n                            %s\n                        } else {\n                            echo 662;\n                        }\n                    }\n                }\n            ")
	generateCases("<?php\n                while(true) {\n                    while(true) {\n                        if($a) {\n                            echo 100;\n                        } elseif($b) {\n                            %s\n                        } else {\n                            echo 3;\n                        }\n                    }\n                }\n            ", "")
	generateCases("<?php\n                while(true) {\n                    while(true) {\n                        if ($a) {\n                            echo 100;\n                        } elseif  ($a1) {\n                            echo 99887;\n                        } elseif  ($b) {\n                            echo $b+1; //\n                            /* test */\n                            %s\n                        } else {\n                            echo 321;\n                        }\n                    }\n                }\n            ", "")
	generateCases("<?php\n            while(true) {\n                while(true) {\n                    if ($a) {\n                        %s\n                    }  \n                        echo 1;\n                    \n                }\n            }\n        ", "<?php\n            while(true) {\n                while(true) {\n                    if ($a) {\n                        %s\n                    } else {\n                        echo 1;\n                    }\n                }\n            }\n        ")
	for _, statement := range nueUpstreamStatements {
		conditionsWithoutBraces(statement)
	}
	conditionsWithoutBraces("throw new class extends Exception{};")
	conditionsWithoutBraces("throw new class ($a, 9) extends Exception{ public function z($a, $b){ echo 7;} };")
	for _, c := range nueUpstreamPHP8Negative {
		cases = append(cases, upstreamCase{"PHP8 negative", nil, fill(nueUpstreamPHP8Template, c), ""})
	}
	conditionsWithoutBraces("$b = $a ?? throw new Exception($i);")
	runUpstreamCases(t, NoUselessElse{}, cases)
}

var nueUpstreamExitStatements = []string{
	"exit;",
	"exit();",
	"exit(1);",
	"die;",
	"die();",
	"die(1);",
	"break;",
	"break 2;",
	"break (2);",
	"continue;",
	"continue 2;",
	"continue (2);",
	"return;",
	"return 1;",
	"return (1);",
	"return \"a\";",
	"return 8+2;",
	"return null;",
	"return sum(1+8*6, 2);",
	"throw $e;",
	"throw ($e);",
	"throw new \\Exception;",
	"throw new \\Exception();",
	"throw new \\Exception((string)12+1);",
}

var nueUpstreamStatements = []string{
	"die;",
	"throw new Exception($i);",
	"while($i < 1) throw/*{}*/new Exception($i);",
	"while($i < 1){throw new Exception($i);}",
	"do{throw new Exception($i);}while($i < 1);",
	"foreach($a as $b)throw new Exception($i);",
	"foreach($a as $b){throw new Exception($i);}",
}

var nueUpstreamPHP8Negative = []string{
	"$bar = $foo1 ?? throw new \\Exception($e);",
	"$callable = fn() => throw new Exception();",
	"$value = $falsableValue ?: throw new InvalidArgumentException();",
	"$value = !empty($array)\n                    ? reset($array)\n                    : throw new InvalidArgumentException();",
	"$a = $condition && throw new Exception();",
	"$a = $condition || throw new Exception();",
	"$a = $condition and throw new Exception();",
	"$a = $condition or throw new Exception();",
}

const nueUpstreamIfTemplate = "<?php\n            if ($a === false)\n            {\n                if ($v) %s\n            }\n            else\n                $ret .= $value;\n\n            return $ret;"

const nueUpstreamIfElseIfTemplate = "<?php\n            if ($a === false)\n            {\n                if ($v) { $ret = \"foo\"; }\n                elseif($a)\n                    %s\n            }\n            else\n                $ret .= $value;\n\n            return $ret;"

const nueUpstreamIfElseTemplate = "<?php\n            if ($a === false)\n            {\n                if ($v) { $ret = \"foo\"; }\n                else\n                    %s\n            }\n            else\n                $ret .= $value;\n\n            return $ret;"

const nueUpstreamPHP8Template = "<?php\n                if ($foo) {\n                    %s\n                } else {\n                    echo 123;\n                }\n            "
