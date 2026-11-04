package rules

import "testing"

// Cases ported from PHP-CS-Fixer's SingleLineThrowFixerTest
// (tests/Fixer/FunctionNotation); every expected output is what PHP-CS-Fixer
// produces for the input (equal to the input when nothing changes).
func TestSingleLineThrowPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	same := func(code string) aliasUpstreamCase { return aliasUpstreamCase{"", nil, code, code} }
	fix := func(expected, input string) aliasUpstreamCase {
		return aliasUpstreamCase{"", nil, input, expected}
	}
	aliasRunUpstreamCases(t, SingleLineThrow{}, []aliasUpstreamCase{
		same("<?php throw new Exception; foo(\n                    \"Foo\"\n                );"),
		same("<?php throw new $exceptionName; foo(\n                    \"Foo\"\n                );"),
		same("<?php throw $exception; foo(\n                    \"Foo\"\n                );"),
		same(`<?php throw new Exception("Foo.", 0);`),
		fix(`<?php throw new Exception("Foo.", 0);`, "<?php throw new Exception(\n                \"Foo.\",\n                0\n            );"),
		fix(`<?php throw new Exception("Foo." . "Bar");`, "<?php throw new Exception(\n                \"Foo.\"\n                .\n                \"Bar\"\n            );"),
		fix(`<?php throw new Exception(new ExceptionReport("Foo"), 0);`, "<?php throw new Exception(\n                new\n                    ExceptionReport(\"Foo\"),\n                0\n            );"),
		fix(`<?php throw new Exception(sprintf('Error with number "%s".', 42));`, "<?php throw new Exception(sprintf(\n                'Error with number \"%s\".',\n                42\n            ));"),
		fix(`<?php throw new SomeVendor\Exception("Foo.");`, "<?php throw new SomeVendor\\Exception(\n                \"Foo.\"\n            );"),
		fix(`<?php throw new \SomeVendor\Exception("Foo.");`, "<?php throw new \\SomeVendor\\Exception(\n                \"Foo.\"\n            );"),
		fix(`<?php throw $this->exceptionFactory->createAnException("Foo");`, "<?php throw $this\n                ->exceptionFactory\n                ->createAnException(\n                    \"Foo\"\n                );"),
		fix(`<?php throw $this->getExceptionFactory()->createAnException("Foo");`, "<?php throw $this\n                ->getExceptionFactory()\n                ->createAnException(\n                    \"Foo\"\n                );"),
		fix(`<?php throw $this->getExceptionFactory()->createAnException(function ($x, $y) { return $x === $y + 2; });`, "<?php throw $this\n                ->getExceptionFactory()\n                ->createAnException(\n                    function\n                    (\n                        $x,\n                        $y\n                    )\n                    {\n                        return $x === $y + 2\n                        ;\n                    }\n                );"),
		fix(`<?php throw ExceptionFactory::createAnException("Foo");`, "<?php throw ExceptionFactory\n                    ::\n                    createAnException(\n                        \"Foo\"\n                    );"),
		fix(`<?php throw new Exception("Foo.", 0);`, "<?php throw\n                new\n                    Exception\n                        (\n                            \"Foo.\"\n                                ,\n                            0\n                        );"),
		fix(`<?php throw new $exceptionName("Foo.");`, "<?php throw new $exceptionName(\n                \"Foo.\"\n            );"),
		fix(`<?php throw new $exceptions[4];`, "<?php throw new $exceptions[\n                4\n            ];"),
		fix(`<?php throw clone $exceptionName("Foo.");`, "<?php throw clone $exceptionName(\n                \"Foo.\"\n            );"),
		fix(`<?php throw new WeirdException("Foo.", -20, "An elephant", 1, 2, 3, 4, 5, 6, 7, 8);`, "<?php throw new WeirdException(\"Foo.\", -20, \"An elephant\",\n\n                1,\n        2,\n                                    3, 4, 5, 6, 7, 8\n            );"),
		fix("<?php\n                if ($foo) {\n                    throw new Exception(\"It is foo.\", 1);\n                } else {\n                    throw new \\Exception(\"It is not foo.\", 0);\n                }\n            ", "<?php\n                if ($foo) {\n                    throw new Exception(\n                        \"It is foo.\",\n                        1\n                    );\n                } else {\n                    throw new \\Exception(\n                        \"It is not foo.\", 0\n                    );\n                }\n            "),
		fix("<?php throw new Exception( /* A */\"Foo\", /* 1 */0 /* 2 */); //3", "<?php throw new Exception( // A\n                \"Foo\", // 1\n                0 // 2\n            ); //3"),
		fix("<?php throw new Exception( /* 0123 */ \"Foo\", /* 1 */0 /* 2 */); //3", "<?php throw new Exception( /* 0123 */\n                \"Foo\", // 1\n                0 // 2\n            ); //3"),
		fix("<?php throw new Exception( /* X  */ \"Foo\", /* 1 */0 /* 2 */); //3", "<?php throw new Exception( /* X\n */\n                \"Foo\", // 1\n                0 // 2\n            ); //3"),
		fix("<?php throw new Exception(  0, /* 1 2 3 */ /*4*/              \"Foo\", /*5*/ /*6*/0 /*7*/);", "<?php throw new Exception(  0, /*\n1\n2\n3\n*/\n  /*4*/              \"Foo\", /*5*/\n/*6*/0 /*7*/);"),
		fix("<?php throw new Exception( /* 0 */${\"Foo\" /* a */}, /*b */fnx/*c */(/*d */0/*e */)/*f */);", "<?php throw new Exception( // 0\n    ${\"Foo\"\n    # a\n    }, #b\n    fnx#c\n    (#d\n    0#e\n    )#f\n);"),
		fix("<?php throw new Exception('Message.'. 1);", "<?php throw new Exception('Message.'.\n1\n);"),
		fix("<?php throw new class() extends Exception\n            {\n                protected $message = \"Custom message\";\n            }\n        ;", "<?php throw\n            new class()\n            extends Exception\n            {\n                protected $message = \"Custom message\";\n            }\n        ;"),
		fix("<?php throw new class extends Exception\n            {\n                protected $message = \"Custom message\";\n            }\n        ;", "<?php throw\n            new class\n            extends Exception\n            {\n                protected $message = \"Custom message\";\n            }\n        ;"),
		fix("<?php throw new Exception(\"Foo.\", 0)?>", "<?php throw new Exception(\n                \"Foo.\",\n                0\n            )?>"),
		// PHP 8.0
		fix(`<?php throw $this?->getExceptionFactory()?->createAnException("Foo");`, "<?php throw $this\n                    ?->getExceptionFactory()\n                    ?->createAnException(\n                    \"Foo\"\n                );"),
		same("<?php\n                match ($number) {\n                    1 => $function->one(),\n                    2 => $function->two(),\n                    default => throw new \\NotOneOrTwo()\n                };\n            "),
		same("<?php\n                match ($number) {\n                    1 => $function->one(),\n                    2 => throw new Exception(\"Number 2 is not allowed.\"),\n                    1 => $function->three(),\n                    default => throw new \\NotOneOrTwo()\n                };\n            "),
		fix(`<?php throw new Exception(match ($a) { 1 => "a", 3 => "b" });`, "<?php throw new Exception(match ($a) {\n                1 => \"a\",\n                3 => \"b\"\n            });"),
		fix("<?php\n$var = [\n    $something[1] ?? throw new Exception(123)\n];\n", "<?php\n$var = [\n    $something[1] ?? throw new Exception(\n\n    123\n\n    )\n];\n"),
		same("<?php\n$var = [\n    $something[1] ?? throw new Exception()\n];\n"),
	})
}
