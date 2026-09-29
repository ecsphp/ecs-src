package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// nativeFunctions is a curated set of common built-in functions. It is
// intentionally partial: names not listed are left untouched (never miscased).
var nativeFunctions = map[string]bool{
	"strlen": true, "count": true, "sizeof": true, "is_array": true, "is_string": true,
	"is_int": true, "is_integer": true, "is_bool": true, "is_null": true, "is_object": true,
	"is_callable": true, "is_numeric": true, "is_float": true, "is_a": true, "is_iterable": true,
	"array_map": true, "array_filter": true, "array_merge": true, "array_keys": true,
	"array_values": true, "array_key_exists": true, "in_array": true, "implode": true,
	"explode": true, "str_replace": true, "str_repeat": true, "substr": true, "strpos": true,
	"stripos": true, "strrpos": true, "strtolower": true, "strtoupper": true, "ucfirst": true,
	"lcfirst": true, "ucwords": true, "trim": true, "ltrim": true, "rtrim": true,
	"sprintf": true, "printf": true, "vsprintf": true, "number_format": true,
	"json_encode": true, "json_decode": true, "preg_match": true, "preg_replace": true,
	"preg_split": true, "preg_match_all": true, "preg_quote": true, "sort": true, "rsort": true,
	"usort": true, "uasort": true, "uksort": true, "ksort": true, "krsort": true, "asort": true,
	"arsort": true, "array_push": true, "array_pop": true, "array_shift": true,
	"array_unshift": true, "array_slice": true, "array_splice": true, "array_reverse": true,
	"array_unique": true, "array_flip": true, "array_combine": true, "array_column": true,
	"array_sum": true, "array_product": true, "array_reduce": true, "array_search": true,
	"array_fill": true, "array_diff": true, "array_intersect": true, "array_pad": true,
	"array_chunk": true, "array_key_first": true, "array_key_last": true, "max": true,
	"min": true, "abs": true, "ceil": true, "floor": true, "round": true, "intval": true,
	"floatval": true, "strval": true, "boolval": true, "gettype": true, "settype": true,
	"function_exists": true, "method_exists": true, "class_exists": true, "interface_exists": true,
	"property_exists": true, "defined": true, "define": true, "constant": true,
	"call_user_func": true, "call_user_func_array": true, "func_get_args": true,
	"func_num_args": true, "compact": true, "extract": true, "print_r": true, "var_dump": true,
	"var_export": true, "serialize": true, "unserialize": true, "base64_encode": true,
	"base64_decode": true, "md5": true, "sha1": true, "hash": true, "dechex": true, "hexdec": true,
	"date": true, "time": true, "mktime": true, "strtotime": true, "microtime": true,
	"str_pad": true, "str_split": true, "str_contains": true, "str_starts_with": true,
	"str_ends_with": true, "wordwrap": true, "nl2br": true, "htmlspecialchars": true,
	"htmlentities": true, "strip_tags": true, "addslashes": true, "stripslashes": true,
	"ord": true, "chr": true, "intdiv": true, "fmod": true, "pow": true,
	"sqrt": true, "rand": true, "mt_rand": true, "random_int": true, "array_is_list": true,
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Casing/NativeFunctionCasingFixer.php
//
// NativeFunctionCasing lowercases calls to native PHP functions (a curated
// subset; unknown names are left untouched, never miscased).
type NativeFunctionCasing struct{}

func (NativeFunctionCasing) Name() string {
	return `PhpCsFixer\Fixer\Casing\NativeFunctionCasingFixer`
}

func (NativeFunctionCasing) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Casing/NativeFunctionCasingFixer.php"
}

func (NativeFunctionCasing) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.Ident {
			continue
		}
		lower := strings.ToLower(t.Value)
		if !nativeFunctions[lower] || t.Value == lower {
			continue
		}
		// must be a function call, not a method or a namespaced name
		if prev, ok := prevSignificant(s, i); ok {
			switch strings.ToLower(prev.Value) {
			case "->", "?->", "::", `\`, "function", "new":
				continue
			}
		}
		if nextSignificantValue(s, i) != "(" {
			continue
		}
		s.SetValue(i, lower)
		changed = true
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Casing/IntegerLiteralCaseFixer.php
//
// IntegerLiteralCase lowercases the prefix and digits of hex/binary/octal
// integer literals ("0XFF" -> "0xff").
type IntegerLiteralCase struct{}

func (IntegerLiteralCase) Name() string {
	return `PhpCsFixer\Fixer\Casing\IntegerLiteralCaseFixer`
}

func (IntegerLiteralCase) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Casing/IntegerLiteralCaseFixer.php"
}

func (IntegerLiteralCase) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if t.Kind != token.Number || len(t.Value) < 2 || t.Value[0] != '0' {
			continue
		}
		var fixed string
		switch t.Value[1] {
		case 'x', 'X':
			// prefix lowercase, hex digits uppercase ("0Xff" -> "0xFF")
			fixed = "0x" + strings.ToUpper(t.Value[2:])
		case 'b', 'B':
			fixed = "0b" + t.Value[2:]
		case 'o', 'O':
			fixed = "0o" + t.Value[2:]
		default:
			continue
		}
		if fixed != t.Value {
			s.SetValue(i, fixed)
			changed = true
		}
	}
	return changed
}
