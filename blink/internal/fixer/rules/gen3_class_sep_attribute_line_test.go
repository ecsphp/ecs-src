package rules

import "testing"

// Expectations are PHP-CS-Fixer's own output. A member with a docblock or an
// attribute directly above it gets exactly one line break below that block,
// and the blank line goes above the whole block, attribute arguments and
// attributes sharing a line included.
func TestClassAttributesSeparationAttributeLine(t *testing.T) {
	t.Parallel()
	// ECS's configuration: only methods are separated
	f := ClassAttributesSeparation{}.WithConfig(map[string]any{
		"elements": map[string]any{"method": "one"},
	})
	for _, c := range []struct{ name, src, want string }{
		{
			"first method of an interface",
			"<?php\n\ninterface Shape\n{\n    #[Deprecated] public function area(): float;\n}\n",
			"<?php\n\ninterface Shape\n{\n    #[Deprecated]\n public function area(): float;\n}\n",
		},
		{
			"later method of an interface",
			"<?php\ninterface I\n{\n    public function a(): void;\n    #[Deprecated] public function b(): void;\n}\n",
			"<?php\ninterface I\n{\n    public function a(): void;\n\n    #[Deprecated]\n public function b(): void;\n}\n",
		},
		{
			"first method with a body",
			"<?php\nclass C\n{\n    #[Deprecated] public function a()\n    {\n    }\n}\n",
			"<?php\nclass C\n{\n    #[Deprecated]\n public function a()\n    {\n    }\n}\n",
		},
		{
			"later method with a body",
			"<?php\nclass C\n{\n    public function a()\n    {\n    }\n    #[Deprecated] public function b()\n    {\n    }\n}\n",
			"<?php\nclass C\n{\n    public function a()\n    {\n    }\n\n    #[Deprecated]\n public function b()\n    {\n    }\n}\n",
		},
		{
			"abstract method",
			"<?php\nabstract class C\n{\n    abstract public function a(): void;\n    #[Deprecated] abstract protected function b(): void;\n}\n",
			"<?php\nabstract class C\n{\n    abstract public function a(): void;\n\n    #[Deprecated]\n abstract protected function b(): void;\n}\n",
		},
		{
			"enum method after a case",
			"<?php\nenum E\n{\n    case A;\n    #[Deprecated] public function label(): string\n    {\n        return 'a';\n    }\n}\n",
			"<?php\nenum E\n{\n    case A;\n\n    #[Deprecated]\n public function label(): string\n    {\n        return 'a';\n    }\n}\n",
		},
		{
			"trait method with several modifiers",
			"<?php\ntrait T\n{\n    #[A] final protected static function a()\n    {\n    }\n}\n",
			"<?php\ntrait T\n{\n    #[A]\n final protected static function a()\n    {\n    }\n}\n",
		},
		{
			"no whitespace before the method",
			"<?php\nclass C\n{\n    public function a()\n    {\n    }\n\n    #[Deprecated]public function b()\n    {\n    }\n}\n",
			"<?php\nclass C\n{\n    public function a()\n    {\n    }\n\n    #[Deprecated]\npublic function b()\n    {\n    }\n}\n",
		},
		{
			"attribute with arguments",
			"<?php\nclass C\n{\n    public function a()\n    {\n    }\n    #[Route('/b', methods: ['GET'])] public function b()\n    {\n    }\n}\n",
			"<?php\nclass C\n{\n    public function a()\n    {\n    }\n\n    #[Route('/b', methods: ['GET'])]\n public function b()\n    {\n    }\n}\n",
		},
		{
			"two attributes on the method line",
			"<?php\nclass C\n{\n    public function a()\n    {\n    }\n    #[A] #[B] public function b()\n    {\n    }\n}\n",
			"<?php\nclass C\n{\n    public function a()\n    {\n    }\n\n    #[A] #[B]\n public function b()\n    {\n    }\n}\n",
		},
		{
			"last of two attribute lines",
			"<?php\nclass C\n{\n    public function a()\n    {\n    }\n    #[A]\n    #[B] public function b()\n    {\n    }\n}\n",
			"<?php\nclass C\n{\n    public function a()\n    {\n    }\n\n    #[A]\n    #[B]\n public function b()\n    {\n    }\n}\n",
		},
		{
			"docblock above the attribute",
			"<?php\nclass C\n{\n    public function a()\n    {\n    }\n    /** B. */\n    #[Deprecated] public function b()\n    {\n    }\n}\n",
			"<?php\nclass C\n{\n    public function a()\n    {\n    }\n\n    /** B. */\n    #[Deprecated]\n public function b()\n    {\n    }\n}\n",
		},
		{
			"docblock and attribute on one line",
			"<?php\nclass C\n{\n    public function a()\n    {\n    }\n    /** D. */ #[A]\n    public function b()\n    {\n    }\n}\n",
			"<?php\nclass C\n{\n    public function a()\n    {\n    }\n\n    /** D. */ #[A]\n    public function b()\n    {\n    }\n}\n",
		},
		{
			"attribute and a comment on one line",
			"<?php\nclass C\n{\n    public function a()\n    {\n    }\n    #[A] // note\n    public function b()\n    {\n    }\n}\n",
			"<?php\nclass C\n{\n    public function a()\n    {\n    }\n\n    #[A] // note\n    public function b()\n    {\n    }\n}\n",
		},
		{
			"docblock on the method line",
			"<?php\ninterface I\n{\n    /** A. */ public function a(): void;\n}\n",
			"<?php\ninterface I\n{\n    /** A. */\n public function a(): void;\n}\n",
		},
		{
			"blank line below the attribute",
			"<?php\nclass C\n{\n    public function a()\n    {\n    }\n\n    #[Deprecated]\n\n    public function b()\n    {\n    }\n}\n",
			"<?php\nclass C\n{\n    public function a()\n    {\n    }\n\n    #[Deprecated]\n    public function b()\n    {\n    }\n}\n",
		},
		{
			"property and constant are not configured",
			"<?php\nclass C\n{\n    #[A] public $a;\n    #[B] public const B = 1;\n}\n",
			"<?php\nclass C\n{\n    #[A] public $a;\n    #[B] public const B = 1;\n}\n",
		},
		{
			"attribute already above the method",
			"<?php\nclass C\n{\n    public function a()\n    {\n    }\n\n    #[Route('/b', methods: ['GET'])]\n    public function b()\n    {\n    }\n}\n",
			"<?php\nclass C\n{\n    public function a()\n    {\n    }\n\n    #[Route('/b', methods: ['GET'])]\n    public function b()\n    {\n    }\n}\n",
		},
		{
			"attribute with arguments on its own line",
			"<?php\nclass C\n{\n    public function a()\n    {\n    }\n    #[Route('/b', methods: ['GET'])]\n    public function b()\n    {\n    }\n}\n",
			"<?php\nclass C\n{\n    public function a()\n    {\n    }\n\n    #[Route('/b', methods: ['GET'])]\n    public function b()\n    {\n    }\n}\n",
		},
		{
			"multi-line attribute",
			"<?php\nclass C\n{\n    public function a()\n    {\n    }\n\n    #[Route(\n        '/b',\n    )] public function b()\n    {\n    }\n}\n",
			"<?php\nclass C\n{\n    public function a()\n    {\n    }\n\n    #[Route(\n        '/b',\n    )]\n public function b()\n    {\n    }\n}\n",
		},
		{
			"anonymous class",
			"<?php\n$c = new class() {\n    #[A] public function a()\n    {\n    }\n};\n",
			"<?php\n$c = new class() {\n    #[A]\n public function a()\n    {\n    }\n};\n",
		},
		{
			"blank line above already there",
			"<?php\ninterface I\n{\n    public function a(): void;\n\n    #[Deprecated] public function b(): void;\n}\n",
			"<?php\ninterface I\n{\n    public function a(): void;\n\n    #[Deprecated]\n public function b(): void;\n}\n",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			assertFix(t, f, c.src, c.want, c.src != c.want)
		})
	}
}

// With the default configuration every member type is separated, so a
// property, a constant or an enum case also goes below its docblock or
// attribute.
func TestClassAttributesSeparationAttributeLineDefault(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ name, src, want string }{
		{
			"property and constant",
			"<?php\nclass C\n{\n    #[A] public $a;\n\n    /** B. */ public const B = 1;\n}\n",
			"<?php\nclass C\n{\n    #[A]\n public $a;\n\n    /** B. */\n public const B = 1;\n}\n",
		},
		{
			"enum cases",
			"<?php\nenum E\n{\n    #[A] case A;\n    #[B] case B;\n}\n",
			"<?php\nenum E\n{\n    #[A]\n case A;\n\n    #[B]\n case B;\n}\n",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			assertFix(t, ClassAttributesSeparation{}, c.src, c.want, true)
		})
	}
}
