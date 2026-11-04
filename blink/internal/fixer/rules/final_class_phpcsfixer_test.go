package rules

import "testing"

// Cases ported from PHP-CS-Fixer's FinalClassFixerTest.
func TestFinalClassPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, FinalClass{}, []aliasUpstreamCase{
		{"doctrine entity", nil, "<?php /** @Entity */ class MyEntity {}", "<?php /** @Entity */ class MyEntity {}"},
		{"orm entity", nil,
			"<?php use Doctrine\\ORM\\Mapping as ORM; /** @ORM\\Entity */ class MyEntity {}",
			"<?php use Doctrine\\ORM\\Mapping as ORM; /** @ORM\\Entity */ class MyEntity {}"},
		{"orm entity with args", nil,
			"<?php use Doctrine\\ORM\\Mapping as ORM; /** @ORM\\Entity(repositoryClass=\"MyRepository\") */ class MyEntity {}",
			"<?php use Doctrine\\ORM\\Mapping as ORM; /** @ORM\\Entity(repositoryClass=\"MyRepository\") */ class MyEntity {}"},
		{"document", nil, "<?php /** @Document */ class MyDocument {}", "<?php /** @Document */ class MyDocument {}"},
		{"entity lowercase", nil, "<?php /** @entity */ class MyEntity {}", "<?php /** @entity */ class MyEntity {}"},
		{"abstract", nil, "<?php abstract class MyAbstract {}", "<?php abstract class MyAbstract {}"},
		{"trait", nil, "<?php trait MyTrait {}", "<?php trait MyTrait {}"},
		{"interface", nil, "<?php interface MyInterface {}", "<?php interface MyInterface {}"},
		{"class constant", nil, "<?php echo Exception::class;", "<?php echo Exception::class;"},
		{"plain class", nil, "<?php class MyClass {}", "<?php final class MyClass {}"},
		{"extends", nil, "<?php class MyClass extends MyAbstract {}", "<?php final class MyClass extends MyAbstract {}"},
		{"implements", nil, "<?php class MyClass implements MyInterface {}", "<?php final class MyClass implements MyInterface {}"},
		{"non-excluded phpdoc", nil,
			"<?php /** @codeCoverageIgnore */ class MyEntity {}",
			"<?php /** @codeCoverageIgnore */ final class MyEntity {}"},
		{"mixed", nil,
			"<?php class A {} abstract class B {} class C {}",
			"<?php final class A {} abstract class B {} final class C {}"},
		{"entity inside description", nil,
			"<?php /** @internal Map my app to an @Entity */ class MyMapper {}",
			"<?php /** @internal Map my app to an @Entity */ final class MyMapper {}"},
		{"anonymous class", nil, "<?php $anonymClass = new class {};", "<?php $anonymClass = new class {};"},
		{"attribute entity", nil, "<?php #[Entity] class MyEntity {}", "<?php #[Entity] class MyEntity {}"},
		{"attribute orm entity", nil,
			"<?php use Doctrine\\ORM\\Mapping as ORM; #[ORM\\Entity] class MyEntity {}",
			"<?php use Doctrine\\ORM\\Mapping as ORM; #[ORM\\Entity] class MyEntity {}"},
		{"attribute two, one excluded", nil, "<?php #[IgnoredAttribute] #[Entity] class MyEntity {}", "<?php #[IgnoredAttribute] #[Entity] class MyEntity {}"},
		{"attribute group, one excluded", nil, "<?php #[IgnoredAttribute(\"Some-Value\"), Entity] class MyEntity {}", "<?php #[IgnoredAttribute(\"Some-Value\"), Entity] class MyEntity {}"},
		{"attribute then comment", nil, "<?php #[Entity] /* some comment */ class MyEntity {}", "<?php #[Entity] /* some comment */ class MyEntity {}"},
		{"comment before attribute", nil, "<?php /* some comment */ #[Entity] class MyEntity {}", "<?php /* some comment */ #[Entity] class MyEntity {}"},
		{"readonly entity attribute", nil, "<?php #[Entity] readonly class MyEntity {}", "<?php #[Entity] readonly class MyEntity {}"},
	})
}
