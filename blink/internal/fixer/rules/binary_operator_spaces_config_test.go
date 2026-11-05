package rules

import "testing"

// The expected output of every case and configuration comes from the PHP engine.
func TestBinaryOperatorSpacesConfigs(t *testing.T) {
	t.Parallel()
	configs := []map[string]any{
		nil,
		{"default": "no_space"},
		{"default": "at_least_single_space"},
		{"default": "align"},
		{"default": "align_by_scope"},
		{"default": "align_single_space"},
		{"default": "align_single_space_by_scope"},
		{"default": "align_single_space_minimal"},
		{"default": "align_single_space_minimal_by_scope"},
		{"default": "single_space", "operators": map[string]any{"=>": "align_single_space_minimal"}},
		{"operators": map[string]any{"=": "align", "=>": "align"}},
		{"operators": map[string]any{"=": "align_single_space_minimal_by_scope", "=>": "align_single_space_by_scope"}},
		{"default": nil, "operators": map[string]any{"=": "single_space"}},
		{"default": nil, "operators": map[string]any{"=>": "align_single_space", "+": "no_space"}},
		{"default": "align_single_space_minimal", "operators": map[string]any{"=": nil, "|": "no_space", "xor": nil}},
	}
	cases := []struct {
		src  string
		want [15]string
	}{
		{"<?php\n$a=1;\n$bbb  =  2;\n$cc =3;\n\n$d = 4;\n$eeeee   = 5;\n", [15]string{
			"<?php\n$a = 1;\n$bbb = 2;\n$cc = 3;\n\n$d = 4;\n$eeeee = 5;\n",
			"<?php\n$a=1;\n$bbb=2;\n$cc=3;\n\n$d=4;\n$eeeee=5;\n",
			"<?php\n$a = 1;\n$bbb  =  2;\n$cc = 3;\n\n$d = 4;\n$eeeee   = 5;\n",
			"<?php\n$a    =1;\n$bbb  =  2;\n$cc   =3;\n\n$d       = 4;\n$eeeee   = 5;\n",
			"<?php\n$a       =1;\n$bbb     =  2;\n$cc      =3;\n\n$d       = 4;\n$eeeee   = 5;\n",
			"<?php\n$a    = 1;\n$bbb  =  2;\n$cc   = 3;\n\n$d       = 4;\n$eeeee   = 5;\n",
			"<?php\n$a       = 1;\n$bbb     =  2;\n$cc      = 3;\n\n$d       = 4;\n$eeeee   = 5;\n",
			"<?php\n$a   = 1;\n$bbb = 2;\n$cc  = 3;\n\n$d     = 4;\n$eeeee = 5;\n",
			"<?php\n$a     = 1;\n$bbb   = 2;\n$cc    = 3;\n\n$d     = 4;\n$eeeee = 5;\n",
			"<?php\n$a = 1;\n$bbb = 2;\n$cc = 3;\n\n$d = 4;\n$eeeee = 5;\n",
			"<?php\n$a    =1;\n$bbb  =  2;\n$cc   =3;\n\n$d       = 4;\n$eeeee   = 5;\n",
			"<?php\n$a     = 1;\n$bbb   = 2;\n$cc    = 3;\n\n$d     = 4;\n$eeeee = 5;\n",
			"<?php\n$a = 1;\n$bbb = 2;\n$cc = 3;\n\n$d = 4;\n$eeeee = 5;\n",
			"<?php\n$a=1;\n$bbb  =  2;\n$cc =3;\n\n$d = 4;\n$eeeee   = 5;\n",
			"<?php\n$a=1;\n$bbb  =  2;\n$cc =3;\n\n$d = 4;\n$eeeee   = 5;\n",
		}},
		{"<?php\n$x = [\n    'a' => 1,\n    'bbbb'   =>   2,\n    'cc'=>3,\n\n    'd' => [\n        'x'=>1,\n        'yyyy' => 2,\n    ],\n    'e'    => [1 => 2, 33 => 4],\n];\n", [15]string{
			"<?php\n$x = [\n    'a' => 1,\n    'bbbb' => 2,\n    'cc' => 3,\n\n    'd' => [\n        'x' => 1,\n        'yyyy' => 2,\n    ],\n    'e' => [1 => 2, 33 => 4],\n];\n",
			"<?php\n$x=[\n    'a'=>1,\n    'bbbb'=>2,\n    'cc'=>3,\n\n    'd'=>[\n        'x'=>1,\n        'yyyy'=>2,\n    ],\n    'e'=>[1=>2, 33=>4],\n];\n",
			"<?php\n$x = [\n    'a' => 1,\n    'bbbb'   =>   2,\n    'cc' => 3,\n\n    'd' => [\n        'x' => 1,\n        'yyyy' => 2,\n    ],\n    'e'    => [1 => 2, 33 => 4],\n];\n",
			"<?php\n$x = [\n    'a'      => 1,\n    'bbbb'   => 2,\n    'cc'     => 3,\n\n    'd' => [\n        'x'    => 1,\n        'yyyy' => 2,\n    ],\n    'e'    => [1 => 2, 33 => 4],\n];\n",
			"<?php\n$x = [\n    'a'      => 1,\n    'bbbb'   => 2,\n    'cc'     => 3,\n\n    'd'      => [\n        'x'    => 1,\n        'yyyy' => 2,\n    ],\n    'e'      => [1 => 2, 33 => 4],\n];\n",
			"<?php\n$x = [\n    'a'      => 1,\n    'bbbb'   => 2,\n    'cc'     => 3,\n\n    'd' => [\n        'x'    => 1,\n        'yyyy' => 2,\n    ],\n    'e'    => [1 => 2, 33 => 4],\n];\n",
			"<?php\n$x = [\n    'a'      => 1,\n    'bbbb'   => 2,\n    'cc'     => 3,\n\n    'd'      => [\n        'x'    => 1,\n        'yyyy' => 2,\n    ],\n    'e'      => [1 => 2, 33 => 4],\n];\n",
			"<?php\n$x = [\n    'a'    => 1,\n    'bbbb' => 2,\n    'cc'   => 3,\n\n    'd' => [\n        'x'    => 1,\n        'yyyy' => 2,\n    ],\n    'e' => [1 => 2, 33 => 4],\n];\n",
			"<?php\n$x = [\n    'a'    => 1,\n    'bbbb' => 2,\n    'cc'   => 3,\n\n    'd'    => [\n        'x'    => 1,\n        'yyyy' => 2,\n    ],\n    'e'    => [1 => 2, 33 => 4],\n];\n",
			"<?php\n$x = [\n    'a'    => 1,\n    'bbbb' => 2,\n    'cc'   => 3,\n\n    'd' => [\n        'x'    => 1,\n        'yyyy' => 2,\n    ],\n    'e' => [1 => 2, 33 => 4],\n];\n",
			"<?php\n$x = [\n    'a'      => 1,\n    'bbbb'   => 2,\n    'cc'     => 3,\n\n    'd' => [\n        'x'    => 1,\n        'yyyy' => 2,\n    ],\n    'e'    => [1 => 2, 33 => 4],\n];\n",
			"<?php\n$x = [\n    'a'      => 1,\n    'bbbb'   => 2,\n    'cc'     => 3,\n\n    'd'      => [\n        'x'    => 1,\n        'yyyy' => 2,\n    ],\n    'e'      => [1 => 2, 33 => 4],\n];\n",
			"<?php\n$x = [\n    'a' => 1,\n    'bbbb'   =>   2,\n    'cc'=>3,\n\n    'd' => [\n        'x'=>1,\n        'yyyy' => 2,\n    ],\n    'e'    => [1 => 2, 33 => 4],\n];\n",
			"<?php\n$x = [\n    'a'      => 1,\n    'bbbb'   => 2,\n    'cc'     => 3,\n\n    'd' => [\n        'x'    => 1,\n        'yyyy' => 2,\n    ],\n    'e'    => [1 => 2, 33 => 4],\n];\n",
			"<?php\n$x = [\n    'a'    => 1,\n    'bbbb' => 2,\n    'cc'   => 3,\n\n    'd' => [\n        'x'    => 1,\n        'yyyy' => 2,\n    ],\n    'e' => [1 => 2, 33 => 4],\n];\n",
		}},
		{"<?php\nfunction foo($a = 1, $bb = 2) {\n    $x = 1;\n    $yyy = 2;\n    if ($x) {\n        $z = 3;\n        $zzzzz = 4;\n    }\n    $w = fn ($a) => [\n        'k' => 1,\n        'kkk' => 2,\n    ];\n}\nclass C {\n    public $a = 1;\n    public $bbb = 2;\n    function m(\n        $p = 1,\n        $pppp = 2\n    ) {\n        $q = 1; $qq = 2;\n        $r =\n            3;\n    }\n}\n", [15]string{
			"<?php\nfunction foo($a = 1, $bb = 2) {\n    $x = 1;\n    $yyy = 2;\n    if ($x) {\n        $z = 3;\n        $zzzzz = 4;\n    }\n    $w = fn ($a) => [\n        'k' => 1,\n        'kkk' => 2,\n    ];\n}\nclass C {\n    public $a = 1;\n    public $bbb = 2;\n    function m(\n        $p = 1,\n        $pppp = 2\n    ) {\n        $q = 1; $qq = 2;\n        $r =\n            3;\n    }\n}\n",
			"<?php\nfunction foo($a=1, $bb=2) {\n    $x=1;\n    $yyy=2;\n    if ($x) {\n        $z=3;\n        $zzzzz=4;\n    }\n    $w=fn ($a)=>[\n        'k'=>1,\n        'kkk'=>2,\n    ];\n}\nclass C {\n    public $a=1;\n    public $bbb=2;\n    function m(\n        $p=1,\n        $pppp=2\n    ) {\n        $q=1; $qq=2;\n        $r=\n            3;\n    }\n}\n",
			"<?php\nfunction foo($a = 1, $bb = 2) {\n    $x = 1;\n    $yyy = 2;\n    if ($x) {\n        $z = 3;\n        $zzzzz = 4;\n    }\n    $w = fn ($a) => [\n        'k' => 1,\n        'kkk' => 2,\n    ];\n}\nclass C {\n    public $a = 1;\n    public $bbb = 2;\n    function m(\n        $p = 1,\n        $pppp = 2\n    ) {\n        $q = 1; $qq = 2;\n        $r =\n            3;\n    }\n}\n",
			"<?php\nfunction foo($a = 1, $bb = 2) {\n    $x   = 1;\n    $yyy = 2;\n    if ($x) {\n        $z     = 3;\n        $zzzzz = 4;\n    }\n    $w = fn ($a) => [\n        'k'   => 1,\n        'kkk' => 2,\n    ];\n}\nclass C {\n    public $a   = 1;\n    public $bbb = 2;\n    function m(\n        $p = 1,\n        $pppp = 2\n    ) {\n        $q = 1; $qq = 2;\n        $r =\n            3;\n    }\n}\n",
			"<?php\nfunction foo($a = 1, $bb = 2) {\n    $x   = 1;\n    $yyy = 2;\n    if ($x) {\n        $z     = 3;\n        $zzzzz = 4;\n    }\n    $w   = fn ($a) => [\n        'k'   => 1,\n        'kkk' => 2,\n    ];\n}\nclass C {\n    public $a   = 1;\n    public $bbb = 2;\n    function m(\n        $p = 1,\n        $pppp = 2\n    ) {\n        $q = 1; $qq = 2;\n        $r =\n            3;\n    }\n}\n",
			"<?php\nfunction foo($a = 1, $bb = 2) {\n    $x   = 1;\n    $yyy = 2;\n    if ($x) {\n        $z     = 3;\n        $zzzzz = 4;\n    }\n    $w = fn ($a) => [\n        'k'   => 1,\n        'kkk' => 2,\n    ];\n}\nclass C {\n    public $a   = 1;\n    public $bbb = 2;\n    function m(\n        $p = 1,\n        $pppp = 2\n    ) {\n        $q = 1; $qq = 2;\n        $r =\n            3;\n    }\n}\n",
			"<?php\nfunction foo($a = 1, $bb = 2) {\n    $x   = 1;\n    $yyy = 2;\n    if ($x) {\n        $z     = 3;\n        $zzzzz = 4;\n    }\n    $w   = fn ($a) => [\n        'k'   => 1,\n        'kkk' => 2,\n    ];\n}\nclass C {\n    public $a   = 1;\n    public $bbb = 2;\n    function m(\n        $p = 1,\n        $pppp = 2\n    ) {\n        $q = 1; $qq = 2;\n        $r =\n            3;\n    }\n}\n",
			"<?php\nfunction foo($a = 1, $bb = 2) {\n    $x   = 1;\n    $yyy = 2;\n    if ($x) {\n        $z     = 3;\n        $zzzzz = 4;\n    }\n    $w = fn ($a) => [\n        'k'   => 1,\n        'kkk' => 2,\n    ];\n}\nclass C {\n    public $a   = 1;\n    public $bbb = 2;\n    function m(\n        $p = 1,\n        $pppp = 2\n    ) {\n        $q = 1; $qq = 2;\n        $r = 3;\n    }\n}\n",
			"<?php\nfunction foo($a = 1, $bb = 2) {\n    $x   = 1;\n    $yyy = 2;\n    if ($x) {\n        $z     = 3;\n        $zzzzz = 4;\n    }\n    $w   = fn ($a) => [\n        'k'   => 1,\n        'kkk' => 2,\n    ];\n}\nclass C {\n    public $a   = 1;\n    public $bbb = 2;\n    function m(\n        $p = 1,\n        $pppp = 2\n    ) {\n        $q = 1; $qq = 2;\n        $r = 3;\n    }\n}\n",
			"<?php\nfunction foo($a = 1, $bb = 2) {\n    $x = 1;\n    $yyy = 2;\n    if ($x) {\n        $z = 3;\n        $zzzzz = 4;\n    }\n    $w = fn ($a) => [\n        'k'   => 1,\n        'kkk' => 2,\n    ];\n}\nclass C {\n    public $a = 1;\n    public $bbb = 2;\n    function m(\n        $p = 1,\n        $pppp = 2\n    ) {\n        $q = 1; $qq = 2;\n        $r =\n            3;\n    }\n}\n",
			"<?php\nfunction foo($a = 1, $bb = 2) {\n    $x   = 1;\n    $yyy = 2;\n    if ($x) {\n        $z     = 3;\n        $zzzzz = 4;\n    }\n    $w = fn ($a) => [\n        'k'   => 1,\n        'kkk' => 2,\n    ];\n}\nclass C {\n    public $a   = 1;\n    public $bbb = 2;\n    function m(\n        $p = 1,\n        $pppp = 2\n    ) {\n        $q = 1; $qq = 2;\n        $r =\n            3;\n    }\n}\n",
			"<?php\nfunction foo($a = 1, $bb = 2) {\n    $x   = 1;\n    $yyy = 2;\n    if ($x) {\n        $z     = 3;\n        $zzzzz = 4;\n    }\n    $w   = fn ($a) => [\n        'k'   => 1,\n        'kkk' => 2,\n    ];\n}\nclass C {\n    public $a   = 1;\n    public $bbb = 2;\n    function m(\n        $p = 1,\n        $pppp = 2\n    ) {\n        $q = 1; $qq = 2;\n        $r = 3;\n    }\n}\n",
			"<?php\nfunction foo($a = 1, $bb = 2) {\n    $x = 1;\n    $yyy = 2;\n    if ($x) {\n        $z = 3;\n        $zzzzz = 4;\n    }\n    $w = fn ($a) => [\n        'k' => 1,\n        'kkk' => 2,\n    ];\n}\nclass C {\n    public $a = 1;\n    public $bbb = 2;\n    function m(\n        $p = 1,\n        $pppp = 2\n    ) {\n        $q = 1; $qq = 2;\n        $r =\n            3;\n    }\n}\n",
			"<?php\nfunction foo($a = 1, $bb = 2) {\n    $x = 1;\n    $yyy = 2;\n    if ($x) {\n        $z = 3;\n        $zzzzz = 4;\n    }\n    $w = fn ($a) => [\n        'k'   => 1,\n        'kkk' => 2,\n    ];\n}\nclass C {\n    public $a = 1;\n    public $bbb = 2;\n    function m(\n        $p = 1,\n        $pppp = 2\n    ) {\n        $q = 1; $qq = 2;\n        $r =\n            3;\n    }\n}\n",
			"<?php\nfunction foo($a = 1, $bb = 2) {\n    $x = 1;\n    $yyy = 2;\n    if ($x) {\n        $z = 3;\n        $zzzzz = 4;\n    }\n    $w = fn ($a) => [\n        'k'   => 1,\n        'kkk' => 2,\n    ];\n}\nclass C {\n    public $a = 1;\n    public $bbb = 2;\n    function m(\n        $p = 1,\n        $pppp = 2\n    ) {\n        $q = 1; $qq = 2;\n        $r =\n            3;\n    }\n}\n",
		}},
		{"<?php\n$a = $b  ===  $c;\n$dd = $e===$f;\n$g = $h+$i - -1;\n$jjj = $k  *  $l / 2 % 3 ** 4;\n$o = $p&$q | $r^$s << 1 >> 2;\n$w = $x and $y or $z xor $q;\n$aa .= 'x';\n$b  ??=  1;\n$e = $f ?? $g <=> $h;\n$p = $q <> $r || $s != $t && $u !== $v;\n$u = $v >= 1 && $w <= 2 || $x < 3 && $y > 4;\n$c -= 1; $c += 1; $c *= 1; $c /= 1; $c %= 1; $c **= 1;\n$c &= 1; $c |= 1; $c ^= 1; $c <<= 1; $c >>= 1;\n", [15]string{
			"<?php\n$a = $b === $c;\n$dd = $e === $f;\n$g = $h + $i - -1;\n$jjj = $k * $l / 2 % 3 ** 4;\n$o = $p & $q | $r ^ $s << 1 >> 2;\n$w = $x and $y or $z xor $q;\n$aa .= 'x';\n$b ??= 1;\n$e = $f ?? $g <=> $h;\n$p = $q <> $r || $s != $t && $u !== $v;\n$u = $v >= 1 && $w <= 2 || $x < 3 && $y > 4;\n$c -= 1; $c += 1; $c *= 1; $c /= 1; $c %= 1; $c **= 1;\n$c &= 1; $c |= 1; $c ^= 1; $c <<= 1; $c >>= 1;\n",
			"<?php\n$a=$b===$c;\n$dd=$e===$f;\n$g=$h+$i--1;\n$jjj=$k*$l/2%3**4;\n$o=$p&$q|$r^$s<<1>>2;\n$w=$xand$yor$zxor$q;\n$aa.='x';\n$b??=1;\n$e=$f??$g<=>$h;\n$p=$q<>$r||$s!=$t&&$u!==$v;\n$u=$v>=1&&$w<=2||$x<3&&$y>4;\n$c-=1; $c+=1; $c*=1; $c/=1; $c%=1; $c**=1;\n$c&=1; $c|=1; $c^=1; $c<<=1; $c>>=1;\n",
			"<?php\n$a = $b  ===  $c;\n$dd = $e === $f;\n$g = $h + $i - -1;\n$jjj = $k  *  $l / 2 % 3 ** 4;\n$o = $p & $q | $r ^ $s << 1 >> 2;\n$w = $x and $y or $z xor $q;\n$aa .= 'x';\n$b  ??=  1;\n$e = $f ?? $g <=> $h;\n$p = $q <> $r || $s != $t && $u !== $v;\n$u = $v >= 1 && $w <= 2 || $x < 3 && $y > 4;\n$c -= 1; $c += 1; $c *= 1; $c /= 1; $c %= 1; $c **= 1;\n$c &= 1; $c |= 1; $c ^= 1; $c <<= 1; $c >>= 1;\n",
			"<?php\n$a   = $b  ===  $c;\n$dd  = $e  ===$f;\n$g   = $h+$i - -1;\n$jjj = $k  *  $l / 2 % 3 ** 4;\n$o   = $p&$q | $r^$s << 1 >> 2;\n$w   = $x and $y or $z xor $q;\n$aa .= 'x';\n$b  ??=  1;\n$e = $f ?? $g <=> $h;\n$p = $q <> $r                        || $s != $t && $u !== $v;\n$u = $v >= 1              && $w <= 2 || $x < 3 && $y > 4;\n$c -= 1; $c += 1; $c *= 1; $c /= 1; $c %= 1; $c **= 1;\n$c &= 1; $c |= 1; $c ^= 1; $c <<= 1; $c >>= 1;\n",
			"<?php\n$a   = $b  ===  $c;\n$dd  = $e  ===$f;\n$g   = $h+$i - -1;\n$jjj = $k  *  $l / 2 % 3 ** 4;\n$o   = $p&$q | $r^$s << 1 >> 2;\n$w   = $x and $y or $z xor $q;\n$aa .= 'x';\n$b  ??=  1;\n$e   = $f ?? $g <=> $h;\n$p   = $q <> $r                        || $s != $t && $u !== $v;\n$u   = $v >= 1              && $w <= 2 || $x < 3 && $y > 4;\n$c -= 1; $c += 1; $c *= 1; $c /= 1; $c %= 1; $c **= 1;\n$c &= 1; $c |= 1; $c ^= 1; $c <<= 1; $c >>= 1;\n",
			"<?php\n$a   = $b  ===  $c;\n$dd  = $e  === $f;\n$g   = $h + $i - -1;\n$jjj = $k  *  $l / 2 % 3 ** 4;\n$o   = $p & $q | $r ^ $s << 1 >> 2;\n$w   = $x and $y or $z xor $q;\n$aa .= 'x';\n$b  ??=  1;\n$e = $f ?? $g <=> $h;\n$p = $q <> $r                        || $s != $t && $u !== $v;\n$u = $v >= 1              && $w <= 2 || $x < 3 && $y > 4;\n$c -= 1; $c += 1; $c *= 1; $c /= 1; $c %= 1; $c **= 1;\n$c &= 1; $c |= 1; $c ^= 1; $c <<= 1; $c >>= 1;\n",
			"<?php\n$a   = $b  ===  $c;\n$dd  = $e  === $f;\n$g   = $h + $i - -1;\n$jjj = $k  *  $l / 2 % 3 ** 4;\n$o   = $p & $q | $r ^ $s << 1 >> 2;\n$w   = $x and $y or $z xor $q;\n$aa .= 'x';\n$b  ??=  1;\n$e   = $f ?? $g <=> $h;\n$p   = $q <> $r                        || $s != $t && $u !== $v;\n$u   = $v >= 1              && $w <= 2 || $x < 3 && $y > 4;\n$c -= 1; $c += 1; $c *= 1; $c /= 1; $c %= 1; $c **= 1;\n$c &= 1; $c |= 1; $c ^= 1; $c <<= 1; $c >>= 1;\n",
			"<?php\n$a   = $b === $c;\n$dd  = $e === $f;\n$g   = $h + $i - -1;\n$jjj = $k * $l / 2 % 3 ** 4;\n$o   = $p & $q | $r ^ $s << 1 >> 2;\n$w   = $x and $y or $z xor $q;\n$aa .= 'x';\n$b ??= 1;\n$e = $f ?? $g <=> $h;\n$p = $q <> $r                        || $s != $t && $u !== $v;\n$u = $v >= 1              && $w <= 2 || $x < 3 && $y > 4;\n$c -= 1; $c += 1; $c *= 1; $c /= 1; $c %= 1; $c **= 1;\n$c &= 1; $c |= 1; $c ^= 1; $c <<= 1; $c >>= 1;\n",
			"<?php\n$a   = $b === $c;\n$dd  = $e === $f;\n$g   = $h + $i - -1;\n$jjj = $k * $l / 2 % 3 ** 4;\n$o   = $p & $q | $r ^ $s << 1 >> 2;\n$w   = $x and $y or $z xor $q;\n$aa .= 'x';\n$b ??= 1;\n$e   = $f ?? $g <=> $h;\n$p   = $q <> $r                        || $s != $t && $u !== $v;\n$u   = $v >= 1              && $w <= 2 || $x < 3 && $y > 4;\n$c -= 1; $c += 1; $c *= 1; $c /= 1; $c %= 1; $c **= 1;\n$c &= 1; $c |= 1; $c ^= 1; $c <<= 1; $c >>= 1;\n",
			"<?php\n$a = $b === $c;\n$dd = $e === $f;\n$g = $h + $i - -1;\n$jjj = $k * $l / 2 % 3 ** 4;\n$o = $p & $q | $r ^ $s << 1 >> 2;\n$w = $x and $y or $z xor $q;\n$aa .= 'x';\n$b ??= 1;\n$e = $f ?? $g <=> $h;\n$p = $q <> $r || $s != $t && $u !== $v;\n$u = $v >= 1 && $w <= 2 || $x < 3 && $y > 4;\n$c -= 1; $c += 1; $c *= 1; $c /= 1; $c %= 1; $c **= 1;\n$c &= 1; $c |= 1; $c ^= 1; $c <<= 1; $c >>= 1;\n",
			"<?php\n$a   = $b === $c;\n$dd  = $e === $f;\n$g   = $h + $i - -1;\n$jjj = $k * $l / 2 % 3 ** 4;\n$o   = $p & $q | $r ^ $s << 1 >> 2;\n$w   = $x and $y or $z xor $q;\n$aa .= 'x';\n$b ??= 1;\n$e = $f ?? $g <=> $h;\n$p = $q <> $r || $s != $t && $u !== $v;\n$u = $v >= 1 && $w <= 2 || $x < 3 && $y > 4;\n$c -= 1; $c += 1; $c *= 1; $c /= 1; $c %= 1; $c **= 1;\n$c &= 1; $c |= 1; $c ^= 1; $c <<= 1; $c >>= 1;\n",
			"<?php\n$a   = $b === $c;\n$dd  = $e === $f;\n$g   = $h + $i - -1;\n$jjj = $k * $l / 2 % 3 ** 4;\n$o   = $p & $q | $r ^ $s << 1 >> 2;\n$w   = $x and $y or $z xor $q;\n$aa .= 'x';\n$b ??= 1;\n$e   = $f ?? $g <=> $h;\n$p   = $q <> $r || $s != $t && $u !== $v;\n$u   = $v >= 1 && $w <= 2 || $x < 3 && $y > 4;\n$c -= 1; $c += 1; $c *= 1; $c /= 1; $c %= 1; $c **= 1;\n$c &= 1; $c |= 1; $c ^= 1; $c <<= 1; $c >>= 1;\n",
			"<?php\n$a = $b  ===  $c;\n$dd = $e===$f;\n$g = $h+$i - -1;\n$jjj = $k  *  $l / 2 % 3 ** 4;\n$o = $p&$q | $r^$s << 1 >> 2;\n$w = $x and $y or $z xor $q;\n$aa .= 'x';\n$b  ??=  1;\n$e = $f ?? $g <=> $h;\n$p = $q <> $r || $s != $t && $u !== $v;\n$u = $v >= 1 && $w <= 2 || $x < 3 && $y > 4;\n$c -= 1; $c += 1; $c *= 1; $c /= 1; $c %= 1; $c **= 1;\n$c &= 1; $c |= 1; $c ^= 1; $c <<= 1; $c >>= 1;\n",
			"<?php\n$a = $b  ===  $c;\n$dd = $e===$f;\n$g = $h+$i - -1;\n$jjj = $k  *  $l / 2 % 3 ** 4;\n$o = $p&$q | $r^$s << 1 >> 2;\n$w = $x and $y or $z xor $q;\n$aa .= 'x';\n$b  ??=  1;\n$e = $f ?? $g <=> $h;\n$p = $q <> $r || $s != $t && $u !== $v;\n$u = $v >= 1 && $w <= 2 || $x < 3 && $y > 4;\n$c -= 1; $c += 1; $c *= 1; $c /= 1; $c %= 1; $c **= 1;\n$c &= 1; $c |= 1; $c ^= 1; $c <<= 1; $c >>= 1;\n",
			"<?php\n$a = $b  === $c;\n$dd = $e === $f;\n$g = $h + $i - -1;\n$jjj = $k * $l / 2 % 3 ** 4;\n$o = $p & $q|$r ^ $s << 1 >> 2;\n$w = $x and $y or $z xor $q;\n$aa .= 'x';\n$b ??= 1;\n$e = $f ?? $g <=> $h;\n$p = $q <> $r                        || $s != $t && $u !== $v;\n$u = $v >= 1              && $w <= 2 || $x < 3 && $y > 4;\n$c -= 1; $c += 1; $c *= 1; $c /= 1; $c %= 1; $c **= 1;\n$c &= 1; $c |= 1; $c ^= 1; $c <<= 1; $c >>= 1;\n",
		}},
		{"<?php\n$a = -$b + +1;\n$c = &$d;\n$c =&$d;\nforeach ($a as $k => &$v) {}\nfunction f(A&B $x, int|string $y, ?Foo &$z = null): static|array {}\ntry {} catch (A|\\B $e) {}\n$e = $f&$g;\n$t = (int) $x+(int) $y;\n$fn = fn&($x) => $x;\n", [15]string{
			"<?php\n$a = -$b + +1;\n$c = &$d;\n$c = &$d;\nforeach ($a as $k => &$v) {}\nfunction f(A&B $x, int|string $y, ?Foo &$z = null): static|array {}\ntry {} catch (A|\\B $e) {}\n$e = $f & $g;\n$t = (int) $x + (int) $y;\n$fn = fn&($x) => $x;\n",
			"<?php\n$a=-$b++1;\n$c=&$d;\n$c=&$d;\nforeach ($a as $k=>&$v) {}\nfunction f(A&B $x, int|string $y, ?Foo &$z=null): static|array {}\ntry {} catch (A|\\B $e) {}\n$e=$f&$g;\n$t=(int) $x+(int) $y;\n$fn=fn&($x)=>$x;\n",
			"<?php\n$a = -$b + +1;\n$c = &$d;\n$c = &$d;\nforeach ($a as $k => &$v) {}\nfunction f(A&B $x, int|string $y, ?Foo &$z = null): static|array {}\ntry {} catch (A|\\B $e) {}\n$e = $f & $g;\n$t = (int) $x + (int) $y;\n$fn = fn&($x) => $x;\n",
			"<?php\n$a = -$b + +1;\n$c = &$d;\n$c =&$d;\nforeach ($a as $k => &$v) {}\nfunction f(A&B $x, int|string $y, ?Foo &$z = null): static|array {}\ntry {} catch (A|\\B $e) {}\n$e  = $f&$g;\n$t  = (int) $x+(int) $y;\n$fn = fn&($x) => $x;\n",
			"<?php\n$a  = -$b     + +1;\n$c  = &$d;\n$c  =&$d;\nforeach ($a as $k => &$v) {}\nfunction f(A&B $x, int|string $y, ?Foo &$z = null): static|array {}\ntry {} catch (A|\\B $e) {}\n$e  = $f&$g;\n$t  = (int) $x+(int) $y;\n$fn = fn&($x) => $x;\n",
			"<?php\n$a = -$b + +1;\n$c = &$d;\n$c = &$d;\nforeach ($a as $k => &$v) {}\nfunction f(A&B $x, int|string $y, ?Foo &$z = null): static|array {}\ntry {} catch (A|\\B $e) {}\n$e  = $f & $g;\n$t  = (int) $x + (int) $y;\n$fn = fn&($x) => $x;\n",
			"<?php\n$a  = -$b      + +1;\n$c  = &$d;\n$c  = &$d;\nforeach ($a as $k => &$v) {}\nfunction f(A&B $x, int|string $y, ?Foo &$z = null): static|array {}\ntry {} catch (A|\\B $e) {}\n$e  = $f & $g;\n$t  = (int) $x + (int) $y;\n$fn = fn&($x) => $x;\n",
			"<?php\n$a = -$b + +1;\n$c = &$d;\n$c = &$d;\nforeach ($a as $k => &$v) {}\nfunction f(A&B $x, int|string $y, ?Foo &$z = null): static|array {}\ntry {} catch (A|\\B $e) {}\n$e  = $f & $g;\n$t  = (int) $x + (int) $y;\n$fn = fn&($x) => $x;\n",
			"<?php\n$a  = -$b      + +1;\n$c  = &$d;\n$c  = &$d;\nforeach ($a as $k => &$v) {}\nfunction f(A&B $x, int|string $y, ?Foo &$z = null): static|array {}\ntry {} catch (A|\\B $e) {}\n$e  = $f & $g;\n$t  = (int) $x + (int) $y;\n$fn = fn&($x) => $x;\n",
			"<?php\n$a = -$b + +1;\n$c = &$d;\n$c = &$d;\nforeach ($a as $k => &$v) {}\nfunction f(A&B $x, int|string $y, ?Foo &$z = null): static|array {}\ntry {} catch (A|\\B $e) {}\n$e = $f & $g;\n$t = (int) $x + (int) $y;\n$fn = fn&($x) => $x;\n",
			"<?php\n$a = -$b + +1;\n$c = &$d;\n$c =&$d;\nforeach ($a as $k => &$v) {}\nfunction f(A&B $x, int|string $y, ?Foo &$z = null): static|array {}\ntry {} catch (A|\\B $e) {}\n$e  = $f & $g;\n$t  = (int) $x + (int) $y;\n$fn = fn&($x) => $x;\n",
			"<?php\n$a  = -$b + +1;\n$c  = &$d;\n$c  = &$d;\nforeach ($a as $k => &$v) {}\nfunction f(A&B $x, int|string $y, ?Foo &$z = null): static|array {}\ntry {} catch (A|\\B $e) {}\n$e  = $f & $g;\n$t  = (int) $x + (int) $y;\n$fn = fn&($x) => $x;\n",
			"<?php\n$a = -$b + +1;\n$c = &$d;\n$c = &$d;\nforeach ($a as $k => &$v) {}\nfunction f(A&B $x, int|string $y, ?Foo &$z = null): static|array {}\ntry {} catch (A|\\B $e) {}\n$e = $f&$g;\n$t = (int) $x+(int) $y;\n$fn = fn&($x) => $x;\n",
			"<?php\n$a = -$b++1;\n$c = &$d;\n$c =&$d;\nforeach ($a as $k => &$v) {}\nfunction f(A&B $x, int|string $y, ?Foo &$z = null): static|array {}\ntry {} catch (A|\\B $e) {}\n$e = $f&$g;\n$t = (int) $x+(int) $y;\n$fn = fn&($x) => $x;\n",
			"<?php\n$a = -$b + +1;\n$c = &$d;\n$c =&$d;\nforeach ($a as $k => &$v) {}\nfunction f(A&B $x, int|string $y, ?Foo &$z = null): static|array {}\ntry {} catch (A|\\B $e) {}\n$e = $f & $g;\n$t = (int) $x + (int) $y;\n$fn = fn&($x) => $x;\n",
		}},
		{"<?php\nfunction g() {\n    yield 'a' => 1;\n    yield 'bbb' => 2;\n}\n$m = match ($x) {\n    1 => 'a',\n    222 => 'b',\n    default => 'c',\n};\n[\n    'a' => $x,\n    'bbb' => $y,\n] = $arr;\nlist('a' => $q, 'bb' => $r) = $arr;\n$arr = array(\n    'a' => 1,\n    'bbb' => array(\n        'c' => 1,\n        'dddd' => 2,\n    ),\n);\n", [15]string{
			"<?php\nfunction g() {\n    yield 'a' => 1;\n    yield 'bbb' => 2;\n}\n$m = match ($x) {\n    1 => 'a',\n    222 => 'b',\n    default => 'c',\n};\n[\n    'a' => $x,\n    'bbb' => $y,\n] = $arr;\nlist('a' => $q, 'bb' => $r) = $arr;\n$arr = array(\n    'a' => 1,\n    'bbb' => array(\n        'c' => 1,\n        'dddd' => 2,\n    ),\n);\n",
			"<?php\nfunction g() {\n    yield 'a'=>1;\n    yield 'bbb'=>2;\n}\n$m=match ($x) {\n    1=>'a',\n    222=>'b',\n    default=>'c',\n};\n[\n    'a'=>$x,\n    'bbb'=>$y,\n]=$arr;\nlist('a'=>$q, 'bb'=>$r)=$arr;\n$arr=array(\n    'a'=>1,\n    'bbb'=>array(\n        'c'=>1,\n        'dddd'=>2,\n    ),\n);\n",
			"<?php\nfunction g() {\n    yield 'a' => 1;\n    yield 'bbb' => 2;\n}\n$m = match ($x) {\n    1 => 'a',\n    222 => 'b',\n    default => 'c',\n};\n[\n    'a' => $x,\n    'bbb' => $y,\n] = $arr;\nlist('a' => $q, 'bb' => $r) = $arr;\n$arr = array(\n    'a' => 1,\n    'bbb' => array(\n        'c' => 1,\n        'dddd' => 2,\n    ),\n);\n",
			"<?php\nfunction g() {\n    yield 'a' => 1;\n    yield 'bbb' => 2;\n}\n$m = match ($x) {\n    1       => 'a',\n    222     => 'b',\n    default => 'c',\n};\n[\n    'a'   => $x,\n    'bbb' => $y,\n]                           = $arr;\nlist('a' => $q, 'bb' => $r) = $arr;\n$arr                        = array(\n    'a'   => 1,\n    'bbb' => array(\n        'c'    => 1,\n        'dddd' => 2,\n    ),\n);\n",
			"<?php\nfunction g() {\n    yield 'a' => 1;\n    yield 'bbb' => 2;\n}\n$m                          = match ($x) {\n    1       => 'a',\n    222     => 'b',\n    default => 'c',\n};\n[\n    'a'   => $x,\n    'bbb' => $y,\n]                           = $arr;\nlist('a' => $q, 'bb' => $r) = $arr;\n$arr                        = array(\n    'a'   => 1,\n    'bbb' => array(\n        'c'    => 1,\n        'dddd' => 2,\n    ),\n);\n",
			"<?php\nfunction g() {\n    yield 'a' => 1;\n    yield 'bbb' => 2;\n}\n$m = match ($x) {\n    1       => 'a',\n    222     => 'b',\n    default => 'c',\n};\n[\n    'a'   => $x,\n    'bbb' => $y,\n]                           = $arr;\nlist('a' => $q, 'bb' => $r) = $arr;\n$arr                        = array(\n    'a'   => 1,\n    'bbb' => array(\n        'c'    => 1,\n        'dddd' => 2,\n    ),\n);\n",
			"<?php\nfunction g() {\n    yield 'a' => 1;\n    yield 'bbb' => 2;\n}\n$m                          = match ($x) {\n    1       => 'a',\n    222     => 'b',\n    default => 'c',\n};\n[\n    'a'   => $x,\n    'bbb' => $y,\n]                           = $arr;\nlist('a' => $q, 'bb' => $r) = $arr;\n$arr                        = array(\n    'a'   => 1,\n    'bbb' => array(\n        'c'    => 1,\n        'dddd' => 2,\n    ),\n);\n",
			"<?php\nfunction g() {\n    yield 'a' => 1;\n    yield 'bbb' => 2;\n}\n$m = match ($x) {\n    1       => 'a',\n    222     => 'b',\n    default => 'c',\n};\n[\n    'a'   => $x,\n    'bbb' => $y,\n]                           = $arr;\nlist('a' => $q, 'bb' => $r) = $arr;\n$arr                        = array(\n    'a'   => 1,\n    'bbb' => array(\n        'c'    => 1,\n        'dddd' => 2,\n    ),\n);\n",
			"<?php\nfunction g() {\n    yield 'a' => 1;\n    yield 'bbb' => 2;\n}\n$m                          = match ($x) {\n    1       => 'a',\n    222     => 'b',\n    default => 'c',\n};\n[\n    'a'   => $x,\n    'bbb' => $y,\n]                           = $arr;\nlist('a' => $q, 'bb' => $r) = $arr;\n$arr                        = array(\n    'a'   => 1,\n    'bbb' => array(\n        'c'    => 1,\n        'dddd' => 2,\n    ),\n);\n",
			"<?php\nfunction g() {\n    yield 'a' => 1;\n    yield 'bbb' => 2;\n}\n$m = match ($x) {\n    1       => 'a',\n    222     => 'b',\n    default => 'c',\n};\n[\n    'a'   => $x,\n    'bbb' => $y,\n] = $arr;\nlist('a' => $q, 'bb' => $r) = $arr;\n$arr = array(\n    'a'   => 1,\n    'bbb' => array(\n        'c'    => 1,\n        'dddd' => 2,\n    ),\n);\n",
			"<?php\nfunction g() {\n    yield 'a' => 1;\n    yield 'bbb' => 2;\n}\n$m = match ($x) {\n    1       => 'a',\n    222     => 'b',\n    default => 'c',\n};\n[\n    'a'   => $x,\n    'bbb' => $y,\n]                           = $arr;\nlist('a' => $q, 'bb' => $r) = $arr;\n$arr                        = array(\n    'a'   => 1,\n    'bbb' => array(\n        'c'    => 1,\n        'dddd' => 2,\n    ),\n);\n",
			"<?php\nfunction g() {\n    yield 'a' => 1;\n    yield 'bbb' => 2;\n}\n$m                          = match ($x) {\n    1       => 'a',\n    222     => 'b',\n    default => 'c',\n};\n[\n    'a'   => $x,\n    'bbb' => $y,\n]                           = $arr;\nlist('a' => $q, 'bb' => $r) = $arr;\n$arr                        = array(\n    'a'   => 1,\n    'bbb' => array(\n        'c'    => 1,\n        'dddd' => 2,\n    ),\n);\n",
			"<?php\nfunction g() {\n    yield 'a' => 1;\n    yield 'bbb' => 2;\n}\n$m = match ($x) {\n    1 => 'a',\n    222 => 'b',\n    default => 'c',\n};\n[\n    'a' => $x,\n    'bbb' => $y,\n] = $arr;\nlist('a' => $q, 'bb' => $r) = $arr;\n$arr = array(\n    'a' => 1,\n    'bbb' => array(\n        'c' => 1,\n        'dddd' => 2,\n    ),\n);\n",
			"<?php\nfunction g() {\n    yield 'a' => 1;\n    yield 'bbb' => 2;\n}\n$m = match ($x) {\n    1       => 'a',\n    222     => 'b',\n    default => 'c',\n};\n[\n    'a'   => $x,\n    'bbb' => $y,\n] = $arr;\nlist('a' => $q, 'bb' => $r) = $arr;\n$arr = array(\n    'a'   => 1,\n    'bbb' => array(\n        'c'    => 1,\n        'dddd' => 2,\n    ),\n);\n",
			"<?php\nfunction g() {\n    yield 'a' => 1;\n    yield 'bbb' => 2;\n}\n$m = match ($x) {\n    1       => 'a',\n    222     => 'b',\n    default => 'c',\n};\n[\n    'a'   => $x,\n    'bbb' => $y,\n] = $arr;\nlist('a' => $q, 'bb' => $r) = $arr;\n$arr = array(\n    'a'   => 1,\n    'bbb' => array(\n        'c'    => 1,\n        'dddd' => 2,\n    ),\n);\n",
		}},
		{"<?php\n$x = ['a' => 1, 'bb' => 2,\n    'ccc' => 3, 'd' => 4];\n$a = [\n    'a' => 1, // c\n    'bbb' => 2,\n    'c'\n        => 3,\n    'dd' =>\n        4,\n    'e'\t=>\t5,\n    1,\n    'ffffff' => [\n        'g' => 1,\n    ], 'h' => 2,\n];\n", [15]string{
			"<?php\n$x = ['a' => 1, 'bb' => 2,\n    'ccc' => 3, 'd' => 4];\n$a = [\n    'a' => 1, // c\n    'bbb' => 2,\n    'c'\n        => 3,\n    'dd' =>\n        4,\n    'e' => 5,\n    1,\n    'ffffff' => [\n        'g' => 1,\n    ], 'h' => 2,\n];\n",
			"<?php\n$x=['a'=>1, 'bb'=>2,\n    'ccc'=>3, 'd'=>4];\n$a=[\n    'a'=>1, // c\n    'bbb'=>2,\n    'c'\n        =>3,\n    'dd'=>\n        4,\n    'e'=>5,\n    1,\n    'ffffff'=>[\n        'g'=>1,\n    ], 'h'=>2,\n];\n",
			"<?php\n$x = ['a' => 1, 'bb' => 2,\n    'ccc' => 3, 'd' => 4];\n$a = [\n    'a' => 1, // c\n    'bbb' => 2,\n    'c'\n        => 3,\n    'dd' =>\n        4,\n    'e'\t=>\t5,\n    1,\n    'ffffff' => [\n        'g' => 1,\n    ], 'h' => 2,\n];\n",
			"<?php\n$x = ['a' => 1, 'bb' => 2,\n    'ccc' => 3, 'd' => 4];\n$a = [\n    'a'   => 1, // c\n    'bbb' => 2,\n    'c'\n         => 3,\n    'dd' =>\n        4,\n    'e'\t=> 5,\n    1,\n    'ffffff' => [\n        'g' => 1,\n    ], 'h' => 2,\n];\n",
			"<?php\n$x = ['a' => 1, 'bb' => 2,\n    'ccc' => 3, 'd' => 4];\n$a = [\n    'a'      => 1, // c\n    'bbb'    => 2,\n    'c'\n             => 3,\n    'dd'     =>\n        4,\n    'e'\t     => 5,\n    1,\n    'ffffff' => [\n        'g' => 1,\n    ], 'h' => 2,\n];\n",
			"<?php\n$x = ['a' => 1, 'bb' => 2,\n    'ccc' => 3, 'd' => 4];\n$a = [\n    'a'   => 1, // c\n    'bbb' => 2,\n    'c'\n         => 3,\n    'dd' =>\n        4,\n    'e'\t => 5,\n    1,\n    'ffffff' => [\n        'g' => 1,\n    ], 'h' => 2,\n];\n",
			"<?php\n$x = ['a' => 1, 'bb' => 2,\n    'ccc' => 3, 'd' => 4];\n$a = [\n    'a'      => 1, // c\n    'bbb'    => 2,\n    'c'\n             => 3,\n    'dd'     =>\n        4,\n    'e'\t     => 5,\n    1,\n    'ffffff' => [\n        'g' => 1,\n    ], 'h' => 2,\n];\n",
			"<?php\n$x = ['a' => 1, 'bb' => 2,\n    'ccc' => 3, 'd' => 4];\n$a = [\n    'a'   => 1, // c\n    'bbb' => 2,\n    'c'\n         => 3,\n    'dd' => 4,\n    'e'  => 5,\n    1,\n    'ffffff' => [\n        'g' => 1,\n    ], 'h' => 2,\n];\n",
			"<?php\n$x = ['a' => 1, 'bb' => 2,\n    'ccc' => 3, 'd' => 4];\n$a = [\n    'a'      => 1, // c\n    'bbb'    => 2,\n    'c'\n             => 3,\n    'dd'     => 4,\n    'e'      => 5,\n    1,\n    'ffffff' => [\n        'g' => 1,\n    ], 'h' => 2,\n];\n",
			"<?php\n$x = ['a' => 1, 'bb' => 2,\n    'ccc' => 3, 'd' => 4];\n$a = [\n    'a'   => 1, // c\n    'bbb' => 2,\n    'c'\n         => 3,\n    'dd' => 4,\n    'e'  => 5,\n    1,\n    'ffffff' => [\n        'g' => 1,\n    ], 'h' => 2,\n];\n",
			"<?php\n$x = ['a' => 1, 'bb' => 2,\n    'ccc' => 3, 'd' => 4];\n$a = [\n    'a'   => 1, // c\n    'bbb' => 2,\n    'c'\n         => 3,\n    'dd' =>\n        4,\n    'e'\t=> 5,\n    1,\n    'ffffff' => [\n        'g' => 1,\n    ], 'h' => 2,\n];\n",
			"<?php\n$x = ['a' => 1, 'bb' => 2,\n    'ccc' => 3, 'd' => 4];\n$a = [\n    'a'      => 1, // c\n    'bbb'    => 2,\n    'c'\n             => 3,\n    'dd'     =>\n        4,\n    'e'\t     => 5,\n    1,\n    'ffffff' => [\n        'g' => 1,\n    ], 'h' => 2,\n];\n",
			"<?php\n$x = ['a' => 1, 'bb' => 2,\n    'ccc' => 3, 'd' => 4];\n$a = [\n    'a' => 1, // c\n    'bbb' => 2,\n    'c'\n        => 3,\n    'dd' =>\n        4,\n    'e'\t=>\t5,\n    1,\n    'ffffff' => [\n        'g' => 1,\n    ], 'h' => 2,\n];\n",
			"<?php\n$x = ['a' => 1, 'bb' => 2,\n    'ccc' => 3, 'd' => 4];\n$a = [\n    'a'   => 1, // c\n    'bbb' => 2,\n    'c'\n         => 3,\n    'dd' =>\n        4,\n    'e'\t => 5,\n    1,\n    'ffffff' => [\n        'g' => 1,\n    ], 'h' => 2,\n];\n",
			"<?php\n$x = ['a' => 1, 'bb' => 2,\n    'ccc' => 3, 'd' => 4];\n$a = [\n    'a'   => 1, // c\n    'bbb' => 2,\n    'c'\n         => 3,\n    'dd' => 4,\n    'e'  => 5,\n    1,\n    'ffffff' => [\n        'g' => 1,\n    ], 'h' => 2,\n];\n",
		}},
		{"<?php\ndeclare(strict_types=1);\ndeclare(ticks=1, encoding='UTF-8');\n$bb  =  /* c */  2;\n$ddd    /* y */ = 4;\n$e =\n    5;\n$ffff =// c\n    6;\n$中文 = 1;\n$hh = 2;\n$s = \"a=$b\";\n", [15]string{
			"<?php\ndeclare(strict_types=1);\ndeclare(ticks=1, encoding = 'UTF-8');\n$bb =  /* c */  2;\n$ddd    /* y */ = 4;\n$e =\n    5;\n$ffff = // c\n    6;\n$中文 = 1;\n$hh = 2;\n$s = \"a=$b\";\n",
			"<?php\ndeclare(strict_types=1);\ndeclare(ticks=1, encoding='UTF-8');\n$bb=  /* c */  2;\n$ddd    /* y */ =4;\n$e=\n    5;\n$ffff=// c\n    6;\n$中文=1;\n$hh=2;\n$s=\"a=$b\";\n",
			"<?php\ndeclare(strict_types=1);\ndeclare(ticks=1, encoding = 'UTF-8');\n$bb  =  /* c */  2;\n$ddd    /* y */ = 4;\n$e =\n    5;\n$ffff = // c\n    6;\n$中文 = 1;\n$hh = 2;\n$s = \"a=$b\";\n",
			"<?php\ndeclare(strict_types=1);\ndeclare(ticks=1, encoding='UTF-8');\n$bb             =  /* c */  2;\n$ddd    /* y */ = 4;\n$e              =\n    5;\n$ffff =// c\n    6;\n$中文 = 1;\n$hh   = 2;\n$s    = \"a=$b\";\n",
			"<?php\ndeclare(strict_types=1);\ndeclare(ticks=1, encoding='UTF-8');\n$bb             =  /* c */  2;\n$ddd    /* y */ = 4;\n$e              =\n    5;\n$ffff           =// c\n    6;\n$中文           = 1;\n$hh             = 2;\n$s              = \"a=$b\";\n",
			"<?php\ndeclare(strict_types=1);\ndeclare(ticks=1, encoding = 'UTF-8');\n$bb             =  /* c */  2;\n$ddd    /* y */ = 4;\n$e              =\n    5;\n$ffff = // c\n    6;\n$中文 = 1;\n$hh   = 2;\n$s    = \"a=$b\";\n",
			"<?php\ndeclare(strict_types=1);\ndeclare(ticks=1, encoding = 'UTF-8');\n$bb             =  /* c */  2;\n$ddd    /* y */ = 4;\n$e              =\n    5;\n$ffff           = // c\n    6;\n$中文           = 1;\n$hh             = 2;\n$s              = \"a=$b\";\n",
			"<?php\ndeclare(strict_types=1);\ndeclare(ticks=1, encoding = 'UTF-8');\n$bb             = /* c */  2;\n$ddd    /* y */ = 4;\n$e              = 5;\n$ffff           = // c\n    6;\n$中文 = 1;\n$hh   = 2;\n$s    = \"a=$b\";\n",
			"<?php\ndeclare(strict_types=1);\ndeclare(ticks=1, encoding = 'UTF-8');\n$bb             = /* c */  2;\n$ddd    /* y */ = 4;\n$e              = 5;\n$ffff           = // c\n    6;\n$中文           = 1;\n$hh             = 2;\n$s              = \"a=$b\";\n",
			"<?php\ndeclare(strict_types=1);\ndeclare(ticks=1, encoding = 'UTF-8');\n$bb =  /* c */  2;\n$ddd    /* y */ = 4;\n$e =\n    5;\n$ffff = // c\n    6;\n$中文 = 1;\n$hh = 2;\n$s = \"a=$b\";\n",
			"<?php\ndeclare(strict_types=1);\ndeclare(ticks=1, encoding='UTF-8');\n$bb             =  /* c */  2;\n$ddd    /* y */ = 4;\n$e              =\n    5;\n$ffff =// c\n    6;\n$中文 = 1;\n$hh   = 2;\n$s    = \"a=$b\";\n",
			"<?php\ndeclare(strict_types=1);\ndeclare(ticks=1, encoding = 'UTF-8');\n$bb             = /* c */  2;\n$ddd    /* y */ = 4;\n$e              = 5;\n$ffff           = // c\n    6;\n$中文           = 1;\n$hh             = 2;\n$s              = \"a=$b\";\n",
			"<?php\ndeclare(strict_types=1);\ndeclare(ticks=1, encoding = 'UTF-8');\n$bb =  /* c */  2;\n$ddd    /* y */ = 4;\n$e =\n    5;\n$ffff = // c\n    6;\n$中文 = 1;\n$hh = 2;\n$s = \"a=$b\";\n",
			"<?php\ndeclare(strict_types=1);\ndeclare(ticks=1, encoding='UTF-8');\n$bb  =  /* c */  2;\n$ddd    /* y */ = 4;\n$e =\n    5;\n$ffff =// c\n    6;\n$中文 = 1;\n$hh = 2;\n$s = \"a=$b\";\n",
			"<?php\ndeclare(strict_types=1);\ndeclare(ticks=1, encoding='UTF-8');\n$bb  =  /* c */  2;\n$ddd    /* y */ = 4;\n$e =\n    5;\n$ffff =// c\n    6;\n$中文 = 1;\n$hh = 2;\n$s = \"a=$b\";\n",
		}},
		{"<?php\nclass P {\n    public int $x {\n        get => $this->x;\n        set => $value;\n    }\n}\n$o->{'a'} = 1;\n$o->bb = 2;\n${'v'} = 3;\n$vv = 4;\nuse A\\{B, C};\n$x = (new Foo)->bar;\n$zz = 2;\n", [15]string{
			"<?php\nclass P {\n    public int $x {\n        get => $this->x;\n        set => $value;\n    }\n}\n$o->{'a'} = 1;\n$o->bb = 2;\n${'v'} = 3;\n$vv = 4;\nuse A\\{B, C};\n$x = (new Foo)->bar;\n$zz = 2;\n",
			"<?php\nclass P {\n    public int $x {\n        get=>$this->x;\n        set=>$value;\n    }\n}\n$o->{'a'}=1;\n$o->bb=2;\n${'v'}=3;\n$vv=4;\nuse A\\{B, C};\n$x=(new Foo)->bar;\n$zz=2;\n",
			"<?php\nclass P {\n    public int $x {\n        get => $this->x;\n        set => $value;\n    }\n}\n$o->{'a'} = 1;\n$o->bb = 2;\n${'v'} = 3;\n$vv = 4;\nuse A\\{B, C};\n$x = (new Foo)->bar;\n$zz = 2;\n",
			"<?php\nclass P {\n    public int $x {\n        get => $this->x;\n        set => $value;\n    }\n}\n$o->{'a'} = 1;\n$o->bb    = 2;\n${'v'}    = 3;\n$vv       = 4;\nuse A\\{B, C};\n$x  = (new Foo)->bar;\n$zz = 2;\n",
			"<?php\nclass P {\n    public int $x {\n        get => $this->x;\n        set => $value;\n    }\n}\n$o->{'a'} = 1;\n$o->bb    = 2;\n${'v'}    = 3;\n$vv       = 4;\nuse A\\{B, C};\n$x        = (new Foo)->bar;\n$zz       = 2;\n",
			"<?php\nclass P {\n    public int $x {\n        get => $this->x;\n        set => $value;\n    }\n}\n$o->{'a'} = 1;\n$o->bb    = 2;\n${'v'}    = 3;\n$vv       = 4;\nuse A\\{B, C};\n$x  = (new Foo)->bar;\n$zz = 2;\n",
			"<?php\nclass P {\n    public int $x {\n        get => $this->x;\n        set => $value;\n    }\n}\n$o->{'a'} = 1;\n$o->bb    = 2;\n${'v'}    = 3;\n$vv       = 4;\nuse A\\{B, C};\n$x        = (new Foo)->bar;\n$zz       = 2;\n",
			"<?php\nclass P {\n    public int $x {\n        get => $this->x;\n        set => $value;\n    }\n}\n$o->{'a'} = 1;\n$o->bb    = 2;\n${'v'}    = 3;\n$vv       = 4;\nuse A\\{B, C};\n$x  = (new Foo)->bar;\n$zz = 2;\n",
			"<?php\nclass P {\n    public int $x {\n        get => $this->x;\n        set => $value;\n    }\n}\n$o->{'a'} = 1;\n$o->bb    = 2;\n${'v'}    = 3;\n$vv       = 4;\nuse A\\{B, C};\n$x        = (new Foo)->bar;\n$zz       = 2;\n",
			"<?php\nclass P {\n    public int $x {\n        get => $this->x;\n        set => $value;\n    }\n}\n$o->{'a'} = 1;\n$o->bb = 2;\n${'v'} = 3;\n$vv = 4;\nuse A\\{B, C};\n$x = (new Foo)->bar;\n$zz = 2;\n",
			"<?php\nclass P {\n    public int $x {\n        get => $this->x;\n        set => $value;\n    }\n}\n$o->{'a'} = 1;\n$o->bb    = 2;\n${'v'}    = 3;\n$vv       = 4;\nuse A\\{B, C};\n$x  = (new Foo)->bar;\n$zz = 2;\n",
			"<?php\nclass P {\n    public int $x {\n        get => $this->x;\n        set => $value;\n    }\n}\n$o->{'a'} = 1;\n$o->bb    = 2;\n${'v'}    = 3;\n$vv       = 4;\nuse A\\{B, C};\n$x        = (new Foo)->bar;\n$zz       = 2;\n",
			"<?php\nclass P {\n    public int $x {\n        get => $this->x;\n        set => $value;\n    }\n}\n$o->{'a'} = 1;\n$o->bb = 2;\n${'v'} = 3;\n$vv = 4;\nuse A\\{B, C};\n$x = (new Foo)->bar;\n$zz = 2;\n",
			"<?php\nclass P {\n    public int $x {\n        get => $this->x;\n        set => $value;\n    }\n}\n$o->{'a'} = 1;\n$o->bb = 2;\n${'v'} = 3;\n$vv = 4;\nuse A\\{B, C};\n$x = (new Foo)->bar;\n$zz = 2;\n",
			"<?php\nclass P {\n    public int $x {\n        get => $this->x;\n        set => $value;\n    }\n}\n$o->{'a'} = 1;\n$o->bb = 2;\n${'v'} = 3;\n$vv = 4;\nuse A\\{B, C};\n$x = (new Foo)->bar;\n$zz = 2;\n",
		}},
		{"<?php\nif ($a == 1\n    && $b == 22\n    || $ccc == 3) {\n}\n$x = $a\n    + $b\n    + $ccc;\n$y = $a +\n    $bbb +\n    $c;\nfor ($i = 0, $jj = 1; $i < 10; $i++) {}\n$a = 1;\n    $bb = 2;\n$ccc = 3;\n", [15]string{
			"<?php\nif ($a == 1\n    && $b == 22\n    || $ccc == 3) {\n}\n$x = $a\n    + $b\n    + $ccc;\n$y = $a +\n    $bbb +\n    $c;\nfor ($i = 0, $jj = 1; $i < 10; $i++) {}\n$a = 1;\n    $bb = 2;\n$ccc = 3;\n",
			"<?php\nif ($a==1\n    &&$b==22\n    ||$ccc==3) {\n}\n$x=$a\n    +$b\n    +$ccc;\n$y=$a+\n    $bbb+\n    $c;\nfor ($i=0, $jj=1; $i<10; $i++) {}\n$a=1;\n    $bb=2;\n$ccc=3;\n",
			"<?php\nif ($a == 1\n    && $b == 22\n    || $ccc == 3) {\n}\n$x = $a\n    + $b\n    + $ccc;\n$y = $a +\n    $bbb +\n    $c;\nfor ($i = 0, $jj = 1; $i < 10; $i++) {}\n$a = 1;\n    $bb = 2;\n$ccc = 3;\n",
			"<?php\nif ($a      == 1\n    && $b   == 22\n    || $ccc == 3) {\n}\n$x = $a\n         + $b\n         + $ccc;\n$y = $a  +\n    $bbb +\n    $c;\nfor ($i = 0, $jj = 1; $i < 10; $i++) {}\n$a      = 1;\n    $bb = 2;\n$ccc    = 3;\n",
			"<?php\nif ($a      == 1\n    && $b   == 22\n    || $ccc == 3) {\n}\n$x      = $a\n             + $b\n             + $ccc;\n$y      = $a +\n    $bbb     +\n    $c;\nfor ($i = 0, $jj = 1; $i < 10; $i++) {}\n$a      = 1;\n    $bb = 2;\n$ccc    = 3;\n",
			"<?php\nif ($a      == 1\n    && $b   == 22\n    || $ccc == 3) {\n}\n$x = $a\n         + $b\n         + $ccc;\n$y = $a  +\n    $bbb +\n    $c;\nfor ($i = 0, $jj = 1; $i < 10; $i++) {}\n$a      = 1;\n    $bb = 2;\n$ccc    = 3;\n",
			"<?php\nif ($a      == 1\n    && $b   == 22\n    || $ccc == 3) {\n}\n$x      = $a\n             + $b\n             + $ccc;\n$y      = $a +\n    $bbb     +\n    $c;\nfor ($i = 0, $jj = 1; $i < 10; $i++) {}\n$a      = 1;\n    $bb = 2;\n$ccc    = 3;\n",
			"<?php\nif ($a      == 1\n    && $b   == 22\n    || $ccc == 3) {\n}\n$x = $a\n        + $b\n        + $ccc;\n$y = $a + $bbb + $c;\nfor ($i = 0, $jj = 1; $i < 10; $i++) {}\n$a      = 1;\n    $bb = 2;\n$ccc    = 3;\n",
			"<?php\nif ($a      == 1\n    && $b   == 22\n    || $ccc == 3) {\n}\n$x      = $a\n             + $b\n             + $ccc;\n$y      = $a + $bbb + $c;\nfor ($i = 0, $jj = 1; $i < 10; $i++) {}\n$a      = 1;\n    $bb = 2;\n$ccc    = 3;\n",
			"<?php\nif ($a == 1\n    && $b == 22\n    || $ccc == 3) {\n}\n$x = $a\n    + $b\n    + $ccc;\n$y = $a +\n    $bbb +\n    $c;\nfor ($i = 0, $jj = 1; $i < 10; $i++) {}\n$a = 1;\n    $bb = 2;\n$ccc = 3;\n",
			"<?php\nif ($a == 1\n    && $b == 22\n    || $ccc == 3) {\n}\n$x = $a\n    + $b\n    + $ccc;\n$y = $a +\n    $bbb +\n    $c;\nfor ($i = 0, $jj = 1; $i < 10; $i++) {}\n$a      = 1;\n    $bb = 2;\n$ccc    = 3;\n",
			"<?php\nif ($a == 1\n    && $b == 22\n    || $ccc == 3) {\n}\n$x      = $a\n    + $b\n    + $ccc;\n$y      = $a +\n    $bbb +\n    $c;\nfor ($i = 0, $jj = 1; $i < 10; $i++) {}\n$a      = 1;\n    $bb = 2;\n$ccc    = 3;\n",
			"<?php\nif ($a == 1\n    && $b == 22\n    || $ccc == 3) {\n}\n$x = $a\n    + $b\n    + $ccc;\n$y = $a +\n    $bbb +\n    $c;\nfor ($i = 0, $jj = 1; $i < 10; $i++) {}\n$a = 1;\n    $bb = 2;\n$ccc = 3;\n",
			"<?php\nif ($a == 1\n    && $b == 22\n    || $ccc == 3) {\n}\n$x = $a\n    +$b\n    +$ccc;\n$y = $a+\n    $bbb+\n    $c;\nfor ($i = 0, $jj = 1; $i < 10; $i++) {}\n$a = 1;\n    $bb = 2;\n$ccc = 3;\n",
			"<?php\nif ($a      == 1\n    && $b   == 22\n    || $ccc == 3) {\n}\n$x = $a\n        + $b\n        + $ccc;\n$y = $a + $bbb + $c;\nfor ($i = 0, $jj = 1; $i < 10; $i++) {}\n$a = 1;\n    $bb = 2;\n$ccc = 3;\n",
		}},
		{"<?php\n$a = function ($x) use ($y) {\n    $aa = 1;\n    $b = 2;\n};\n$c = 3;\n$dddd = 4;\n$e = new class(1) {\n    public $a = 1;\n    public $bb = 2;\n};\nenum E: string {\n    case A = 'a';\n    case BBB = 'b';\n}\n", [15]string{
			"<?php\n$a = function ($x) use ($y) {\n    $aa = 1;\n    $b = 2;\n};\n$c = 3;\n$dddd = 4;\n$e = new class(1) {\n    public $a = 1;\n    public $bb = 2;\n};\nenum E: string {\n    case A = 'a';\n    case BBB = 'b';\n}\n",
			"<?php\n$a=function ($x) use ($y) {\n    $aa=1;\n    $b=2;\n};\n$c=3;\n$dddd=4;\n$e=new class(1) {\n    public $a=1;\n    public $bb=2;\n};\nenum E: string {\n    case A='a';\n    case BBB='b';\n}\n",
			"<?php\n$a = function ($x) use ($y) {\n    $aa = 1;\n    $b = 2;\n};\n$c = 3;\n$dddd = 4;\n$e = new class(1) {\n    public $a = 1;\n    public $bb = 2;\n};\nenum E: string {\n    case A = 'a';\n    case BBB = 'b';\n}\n",
			"<?php\n$a = function ($x) use ($y) {\n    $aa = 1;\n    $b  = 2;\n};\n$c    = 3;\n$dddd = 4;\n$e    = new class(1) {\n    public $a  = 1;\n    public $bb = 2;\n};\nenum E: string {\n    case A   = 'a';\n    case BBB = 'b';\n}\n",
			"<?php\n$a    = function ($x) use ($y) {\n    $aa = 1;\n    $b  = 2;\n};\n$c    = 3;\n$dddd = 4;\n$e    = new class(1) {\n    public $a  = 1;\n    public $bb = 2;\n};\nenum E: string {\n    case A   = 'a';\n    case BBB = 'b';\n}\n",
			"<?php\n$a = function ($x) use ($y) {\n    $aa = 1;\n    $b  = 2;\n};\n$c    = 3;\n$dddd = 4;\n$e    = new class(1) {\n    public $a  = 1;\n    public $bb = 2;\n};\nenum E: string {\n    case A   = 'a';\n    case BBB = 'b';\n}\n",
			"<?php\n$a    = function ($x) use ($y) {\n    $aa = 1;\n    $b  = 2;\n};\n$c    = 3;\n$dddd = 4;\n$e    = new class(1) {\n    public $a  = 1;\n    public $bb = 2;\n};\nenum E: string {\n    case A   = 'a';\n    case BBB = 'b';\n}\n",
			"<?php\n$a = function ($x) use ($y) {\n    $aa = 1;\n    $b  = 2;\n};\n$c    = 3;\n$dddd = 4;\n$e    = new class(1) {\n    public $a  = 1;\n    public $bb = 2;\n};\nenum E: string {\n    case A   = 'a';\n    case BBB = 'b';\n}\n",
			"<?php\n$a    = function ($x) use ($y) {\n    $aa = 1;\n    $b  = 2;\n};\n$c    = 3;\n$dddd = 4;\n$e    = new class(1) {\n    public $a  = 1;\n    public $bb = 2;\n};\nenum E: string {\n    case A   = 'a';\n    case BBB = 'b';\n}\n",
			"<?php\n$a = function ($x) use ($y) {\n    $aa = 1;\n    $b = 2;\n};\n$c = 3;\n$dddd = 4;\n$e = new class(1) {\n    public $a = 1;\n    public $bb = 2;\n};\nenum E: string {\n    case A = 'a';\n    case BBB = 'b';\n}\n",
			"<?php\n$a = function ($x) use ($y) {\n    $aa = 1;\n    $b  = 2;\n};\n$c    = 3;\n$dddd = 4;\n$e    = new class(1) {\n    public $a  = 1;\n    public $bb = 2;\n};\nenum E: string {\n    case A   = 'a';\n    case BBB = 'b';\n}\n",
			"<?php\n$a    = function ($x) use ($y) {\n    $aa = 1;\n    $b  = 2;\n};\n$c    = 3;\n$dddd = 4;\n$e    = new class(1) {\n    public $a  = 1;\n    public $bb = 2;\n};\nenum E: string {\n    case A   = 'a';\n    case BBB = 'b';\n}\n",
			"<?php\n$a = function ($x) use ($y) {\n    $aa = 1;\n    $b = 2;\n};\n$c = 3;\n$dddd = 4;\n$e = new class(1) {\n    public $a = 1;\n    public $bb = 2;\n};\nenum E: string {\n    case A = 'a';\n    case BBB = 'b';\n}\n",
			"<?php\n$a = function ($x) use ($y) {\n    $aa = 1;\n    $b = 2;\n};\n$c = 3;\n$dddd = 4;\n$e = new class(1) {\n    public $a = 1;\n    public $bb = 2;\n};\nenum E: string {\n    case A = 'a';\n    case BBB = 'b';\n}\n",
			"<?php\n$a = function ($x) use ($y) {\n    $aa = 1;\n    $b = 2;\n};\n$c = 3;\n$dddd = 4;\n$e = new class(1) {\n    public $a = 1;\n    public $bb = 2;\n};\nenum E: string {\n    case A = 'a';\n    case BBB = 'b';\n}\n",
		}},
		{"<?php $a=[1=>2, 'b'  =>  3]; $out = $items |> array_values(...) |> count(...);\n", [15]string{
			"<?php $a = [1 => 2, 'b' => 3]; $out = $items |> array_values(...) |> count(...);\n",
			"<?php $a=[1=>2, 'b'=>3]; $out=$items |> array_values(...) |> count(...);\n",
			"<?php $a = [1 => 2, 'b'  =>  3]; $out = $items |> array_values(...) |> count(...);\n",
			"<?php $a=[1=>2, 'b'  =>  3]; $out = $items |> array_values(...) |> count(...);\n",
			"<?php $a=[1=>2, 'b'  =>  3]; $out = $items |> array_values(...) |> count(...);\n",
			"<?php $a = [1 => 2, 'b'  =>  3]; $out = $items |> array_values(...) |> count(...);\n",
			"<?php $a = [1 => 2, 'b'  =>  3]; $out = $items |> array_values(...) |> count(...);\n",
			"<?php $a = [1 => 2, 'b' => 3]; $out = $items |> array_values(...) |> count(...);\n",
			"<?php $a = [1 => 2, 'b' => 3]; $out = $items |> array_values(...) |> count(...);\n",
			"<?php $a = [1 => 2, 'b' => 3]; $out = $items |> array_values(...) |> count(...);\n",
			"<?php $a=[1=>2, 'b'  =>  3]; $out = $items |> array_values(...) |> count(...);\n",
			"<?php $a = [1 => 2, 'b'  =>  3]; $out = $items |> array_values(...) |> count(...);\n",
			"<?php $a = [1=>2, 'b'  =>  3]; $out = $items |> array_values(...) |> count(...);\n",
			"<?php $a=[1 => 2, 'b'  =>  3]; $out = $items |> array_values(...) |> count(...);\n",
			"<?php $a=[1 => 2, 'b' => 3]; $out = $items |> array_values(...) |> count(...);\n",
		}},
	}
	for _, c := range cases {
		for i, config := range configs {
			var f fixerRule = BinaryOperatorSpaces{}
			if config != nil {
				f = BinaryOperatorSpaces{}.WithConfig(config).(fixerRule)
			}
			got, changed := apply(t, f, c.src)
			if got != c.want[i] || changed != (c.src != c.want[i]) {
				t.Errorf("config %d: %q\n got=%q changed=%v\nwant=%q", i, c.src, got, changed, c.want[i])
			}
		}
	}
}

// The expected outputs come from the PHP engine.
func TestBinaryOperatorSpacesConfigCases(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		rule fixerRule
		src  string
		want string
	}{
		{"default single", BinaryOperatorSpaces{}, "<?php $a=[1=>2];", "<?php $a = [1 => 2];"},
		{"no space", BinaryOperatorSpaces{}.WithConfig(map[string]any{"default": "no_space"}).(fixerRule), "<?php $a = [1 => 2];", "<?php $a=[1=>2];"},
		{"operator override", BinaryOperatorSpaces{}.WithConfig(map[string]any{"default": "single_space", "operators": map[string]any{"=>": "no_space"}}).(fixerRule), "<?php $a=[1 => 2];", "<?php $a = [1=>2];"},
		{"align with nothing to align with", BinaryOperatorSpaces{}.WithConfig(map[string]any{"default": "align"}).(fixerRule), "<?php $a=1;", "<?php $a=1;"},
		{"nil default leaves rest alone", BinaryOperatorSpaces{}.WithConfig(map[string]any{"default": nil, "operators": map[string]any{"=": "single_space"}}).(fixerRule), "<?php $a=[1=>2];", "<?php $a = [1=>2];"},
		{"extra operator via map", BinaryOperatorSpaces{}.WithConfig(map[string]any{"operators": map[string]any{"+=": "single_space"}}).(fixerRule), "<?php $a+=1;", "<?php $a += 1;"},
		{"compound concat assignment spaced by default", BinaryOperatorSpaces{}, "<?php $a .='c';", "<?php $a .= 'c';"},
		{"compound add assignment spaced by default", BinaryOperatorSpaces{}, "<?php $a +=1;", "<?php $a += 1;"},
		{"compound assignment honors no_space default", BinaryOperatorSpaces{}.WithConfig(map[string]any{"default": "no_space"}).(fixerRule), "<?php $a .= 'c';", "<?php $a.='c';"},
		{"collapse aligned arrow inside attribute", BinaryOperatorSpaces{}, "<?php #[X([\n    'a'   => 1,\n    'bb'  => 2,\n])]\nclass C {}", "<?php #[X([\n    'a' => 1,\n    'bb' => 2,\n])]\nclass C {}"},
		{"arrow inside attribute string left alone", BinaryOperatorSpaces{}, "<?php #[X('a   => b')]\nclass C {}", "<?php #[X('a   => b')]\nclass C {}"},
		{"reference assignment is spaced around =", BinaryOperatorSpaces{}, "<?php $a   =&$b;", "<?php $a = &$b;"},
		{"compound assign follows default without override", BinaryOperatorSpaces{}.WithConfig(map[string]any{"default": "single_space", "operators": map[string]any{"=>": "single_space", "=": "single_space"}}).(fixerRule), "<?php $url .='c'; $a .=$b;", "<?php $url .= 'c'; $a .= $b;"},
		{"compound assign explicit no_space override", BinaryOperatorSpaces{}.WithConfig(map[string]any{"default": "single_space", "operators": map[string]any{".=": "no_space"}}).(fixerRule), "<?php $url .= 'c';", "<?php $url.='c';"},
	}
	for _, c := range cases {
		got, _ := apply(t, c.rule, c.src)
		if got != c.want {
			t.Errorf("%s: got=%q want=%q", c.name, got, c.want)
		}
	}
}

// PHP-CS-Fixer rejects an unsupported operator or strategy, so there is no
// expected output to match: blink ignores the entry and keeps the rest.
func TestBinaryOperatorSpacesRejectedEntriesKeepDefault(t *testing.T) {
	t.Parallel()
	src := "<?php $a=1 . 2; $b = [1  =>  2];\n"
	want := "<?php $a = 1 . 2; $b = [1=>2];\n"
	f := BinaryOperatorSpaces{}.WithConfig(map[string]any{
		"default":   "bogus",
		"operators": map[string]any{".": "no_space", "|>": "align", "=": 1, "=>": "no_space"},
	}).(fixerRule)
	if got, _ := apply(t, f, src); got != want {
		t.Errorf("\n got=%q\nwant=%q", got, want)
	}
}

// A "|" or "&" joining types is not an operator: return types (after a use
// clause too), variadic and by-reference parameters, DNF and promoted
// properties, typed constants, catch lists. The expected outputs come from the
// PHP engine.
func TestBinaryOperatorSpacesTypePositions(t *testing.T) {
	t.Parallel()
	src := "<?php\n$f = static function ($a) use ($b): float|string { return 1; };\n$g = function () use ($x): array|true {};\n$h = function ($a): float|string { return 1; };\nfunction f(string $a, int|float|null ...$rest) {}\n$i = fn (Foo $c, string|int ...$v): ?float => 1;\nfunction g(null|A|B ...$t): self {}\nclass K {\n    public function and(A|B ...$others): static {}\n    public function __construct(private (A&B)|null $c = null) {}\n    public A&B $p;\n    public (A&B)|C $q;\n    const int|string X = 1;\n    function &r(): A|B {}\n}\nfunction h(int &...$x, A&B ...$y, A & $z, A&B &$w) {}\ntry {} catch (A|B $e) {} catch (C|D) {}\n$j = static fn (A|B $a): C&D => $a & $b | $c;\nenum E: string { case A = 'a'; }\n$k = $a | $b & $c;\n$l = X|Y;\n$m = foo()|$n;\n"
	for i, f := range []fixerRule{
		BinaryOperatorSpaces{},
		BinaryOperatorSpaces{}.WithConfig(map[string]any{"default": "no_space"}).(fixerRule),
	} {
		want := []string{
			"<?php\n$f = static function ($a) use ($b): float|string { return 1; };\n$g = function () use ($x): array|true {};\n$h = function ($a): float|string { return 1; };\nfunction f(string $a, int|float|null ...$rest) {}\n$i = fn (Foo $c, string|int ...$v): ?float => 1;\nfunction g(null|A|B ...$t): self {}\nclass K {\n    public function and(A|B ...$others): static {}\n    public function __construct(private (A&B)|null $c = null) {}\n    public A&B $p;\n    public (A&B)|C $q;\n    const int|string X = 1;\n    function &r(): A|B {}\n}\nfunction h(int &...$x, A&B ...$y, A & $z, A&B &$w) {}\ntry {} catch (A|B $e) {} catch (C|D) {}\n$j = static fn (A|B $a): C&D => $a & $b | $c;\nenum E: string { case A = 'a'; }\n$k = $a | $b & $c;\n$l = X | Y;\n$m = foo() | $n;\n",
			"<?php\n$f=static function ($a) use ($b): float|string { return 1; };\n$g=function () use ($x): array|true {};\n$h=function ($a): float|string { return 1; };\nfunction f(string $a, int|float|null ...$rest) {}\n$i=fn (Foo $c, string|int ...$v): ?float=>1;\nfunction g(null|A|B ...$t): self {}\nclass K {\n    public function and(A|B ...$others): static {}\n    public function __construct(private (A&B)|null $c=null) {}\n    public A&B $p;\n    public (A&B)|C $q;\n    const int|string X=1;\n    function &r(): A|B {}\n}\nfunction h(int &...$x, A&B ...$y, A & $z, A&B &$w) {}\ntry {} catch (A|B $e) {} catch (C|D) {}\n$j=static fn (A|B $a): C&D=>$a&$b|$c;\nenum E: string { case A='a'; }\n$k=$a|$b&$c;\n$l=X|Y;\n$m=foo()|$n;\n",
		}[i]
		if got, _ := apply(t, f, src); got != want {
			t.Errorf("config %d\n got=%q\nwant=%q", i, got, want)
		}
	}
}
