package rules

import "testing"

// TestDoctrineAnnotationBracesCandidates follows nextElementAcceptsDoctrineAnnotations:
// only a docblock before a class, or before a class element reached past
// MODIFIER_KINDS (no "static", "var", union type, "array", attribute or
// visibility-promoted parameter in between), is fixed.
func TestDoctrineAnnotationBracesCandidates(t *testing.T) {
	t.Parallel()
	runUpstreamCases(t, DoctrineAnnotationBraces{}, []upstreamCase{{
		name: "candidates",
		expected: `<?php

/**
 * @Entity
 */
final class Sample
{
    /** @A */
    public function plain() {}

    /** @B() */
    private static function staticOne() {}

    /** @C() */
    private int|string $union;

    /** @D() */
    private array $list;

    /** @E */
    private ?Foo $nullable;

    /** @F */
    public const X = 1;

    /** @G() */
    var $old;

    /** @H */
    use SomeTrait;

    public function __construct(
        /** @I() */
        public int $promoted,
        /** @J */
        readonly int $readonlyOnly,
        /** @K() */
        int $plainParam,
        /** @T */
        private(set) int $privateSet,
        /** @U */
        protected(set) int $protectedSet,
        /** @V */
        public(set) int $publicSet,
        /** @W() */
        public protected(set) int $promotedSet,
    ) {
        /** @L() */
        $local = 1;
    }

    /** @M() */
    #[Attr]
    public function withAttribute() {}

    /** @N() */
    public static $prop;

    /** @O */
    public \Foo\Bar $qualified;

    /** @X */
    public private(set) int $asymmetric;
}

/** @P() */
interface I
{
    /** @Q() */
    public function f();
}

/** @R */
abstract readonly class R {}

$x = new class {
    /** @S */
    public $p;
};
`,
		input: `<?php

/**
 * @Entity()
 */
final class Sample
{
    /** @A() */
    public function plain() {}

    /** @B() */
    private static function staticOne() {}

    /** @C() */
    private int|string $union;

    /** @D() */
    private array $list;

    /** @E() */
    private ?Foo $nullable;

    /** @F() */
    public const X = 1;

    /** @G() */
    var $old;

    /** @H() */
    use SomeTrait;

    public function __construct(
        /** @I() */
        public int $promoted,
        /** @J() */
        readonly int $readonlyOnly,
        /** @K() */
        int $plainParam,
        /** @T() */
        private(set) int $privateSet,
        /** @U() */
        protected(set) int $protectedSet,
        /** @V() */
        public(set) int $publicSet,
        /** @W() */
        public protected(set) int $promotedSet,
    ) {
        /** @L() */
        $local = 1;
    }

    /** @M() */
    #[Attr]
    public function withAttribute() {}

    /** @N() */
    public static $prop;

    /** @O() */
    public \Foo\Bar $qualified;

    /** @X() */
    public private(set) int $asymmetric;
}

/** @P() */
interface I
{
    /** @Q() */
    public function f();
}

/** @R() */
abstract readonly class R {}

$x = new class {
    /** @S() */
    public $p;
};
`,
	}})
}
