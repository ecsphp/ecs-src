<?php
$x = $foo ? function (): int {
    return 1;
} : null;
$y = $foo ? fn (): int => 1 : 2;
$z = $a ? static function ($b): ?string { return $b; } : $c;
$a = (string) -1;
$b = (int) +$c;
$d = (float)-1;
$e = (bool) - $x;
$f = $g - (int) -$h;
function foo((A&B)|C $x, (D&E)|null $y = null): (A&B)|C {
    return $x;
}
class K {
    public function bar(C|(A&B) $p, int $q = 1|2): void {}
    private (A&B)|C $prop;
}
$fn = fn ((A&B)|C $x): (A&B)|null => $x;
$v = $a & $b | $c;
$x = $foo ? function () use ($a): int { return $a; } : null;
$x = $foo ? function &(): array { return $a; } : null;
$x = $foo ? $bar ? function (): int { return 1; } : 2 : 3;
$x = $foo ?: function (): int { return 1; };
$x = [$foo ? function (): int { return 1; } : null, 1];
$x = $foo ? fn&(): int => $a : null;
$x = $foo ? fn () => function (): int { return 1; } : null;
$x = $a ? (string) -1 : (int) -2;
$x = [(string) -1, (array) -2];
$x = foo((string) -1);
$x = (string)-$a - (int) -$b;
$x = (unset) -1;
$x = ( string ) -1;
$x = (binary) -1;
$x = function &(): array { return $a; };
$x = fn&(): int => $a;
$x = $foo ? function &(): array { return $a; } : null;
$x = $foo ? function & (): array { return $a; } : null;
$x = $foo ? function &() use ($b): array { return $a; } : null;
$x = $foo ? fn&(): int => $a : null;
$x = $foo ? fn &(): int => $a : null;
$x = $foo ? fn& (): int => $a : null;
$x = $foo ? fn&($a): ?int => $a : null;
class K {
    public function &foo(): array { return $this->a ? $this->b : $this->c; }
    public function bar() { return $this->a ? function &(): int { return 1; } : 2; }
}
function &baz(): array { return []; }
$x = $foo ? function &(): array { return $a; }:null;
$x = $foo ? bar(function &(): int { return 1; }) : null;
$x = $foo ? fn&(): int => $a ? 1 : 2 : null;
$x = $a ?: fn&(): int => $a;
$x = $foo ? [fn&(): int => $a] : null;
$x = $foo ? function &($a): int { return $a; } : (fn&(): int => 1);
$y = $c ? $d : function &(): int { return 1; };
function q((A&B)|null $a, (C&D)|(E&F) $b, ?int $c = null, (G&H)|false &$d = null, (I&J)|K ...$e) {}
$r = fn((A&B)|C $x, (D&E)|null $y): int => 1;
$s = function ((A&B)|C $x, (D&E)|null $y) use ($z): int { return 1; };
