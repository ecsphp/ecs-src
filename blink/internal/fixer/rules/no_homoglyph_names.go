package rules

import (
	"strings"
	"unicode/utf8"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Naming/NoHomoglyphNamesFixer.php
//
// NoHomoglyphNames replaces non-ASCII look-alikes of ASCII letters, digits and
// "_" with the ASCII character in names, also inside strings. A name that is not
// valid UTF-8 is left alone, as is a named argument's name.
type NoHomoglyphNames struct{}

func (NoHomoglyphNames) Name() string {
	return `PhpCsFixer\Fixer\Naming\NoHomoglyphNamesFixer`
}

func (NoHomoglyphNames) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Naming/NoHomoglyphNamesFixer.php"
}

func (NoHomoglyphNames) Fix(s *tokens.Stream) bool {
	changed := false
	for i := range s.Len() {
		t := s.At(i)
		if !homoglyphNonASCII(t.Value) {
			continue
		}
		v := t.Value
		switch t.Kind {
		case token.Variable:
			v = homoglyphReplace(v)
		case token.Ident:
			if !homoglyphNamedArgument(s, i) {
				v = homoglyphReplace(v)
			}
		case token.String:
			parts := encapsedParts(v)
			repl := map[int]string{}
			for k, p := range parts {
				if p.kind != encVariable && p.kind != encName {
					continue
				}
				if r := homoglyphReplace(v[p.start:p.end]); r != v[p.start:p.end] {
					repl[k] = r
				}
			}
			if len(repl) > 0 {
				v = encRebuild(v, parts, repl)
			}
		}
		if v != t.Value {
			s.SetValue(i, v)
			changed = true
		}
	}
	return changed
}

// homoglyphNamedArgument reports a name PHP-CS-Fixer retypes as
// CT::T_NAMED_ARGUMENT_NAME: followed by ":" and preceded by "(" or ",".
func homoglyphNamedArgument(s *tokens.Stream, i int) bool {
	if !isPunctVal(s, nextMeaningfulIndex(s, i), ":") {
		return false
	}
	prev := prevMeaningfulIndex(s, i)
	return isPunctVal(s, prev, "(") || isPunctVal(s, prev, ",")
}

func homoglyphNonASCII(v string) bool {
	for i := 0; i < len(v); i++ {
		if v[i] >= 0x80 {
			return true
		}
	}
	return false
}

// homoglyphReplace is the Preg::replaceCallback over [^[:ascii:]] with the u
// modifier: a value that is not valid UTF-8 is left as it is.
func homoglyphReplace(v string) string {
	if !utf8.ValidString(v) {
		return v
	}
	var b strings.Builder
	b.Grow(len(v))
	for _, r := range v {
		if c, ok := homoglyphReplacements[r]; ok {
			b.WriteByte(c)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// homoglyphReplacements is NoHomoglyphNamesFixer::REPLACEMENTS without its
// ASCII keys, which the [^[:ascii:]] match never reaches.
var homoglyphReplacements = func() map[rune]byte {
	m := map[rune]byte{}
	for ascii, glyphs := range map[byte]string{
		'0': "０",
		'1': "１",
		'2': "２",
		'3': "３",
		'4': "４",
		'5': "５",
		'6': "６",
		'7': "７",
		'8': "８",
		'9': "９",
		'A': "ΑАＡ",
		'B': "ʙΒВＢ",
		'C': "ϹСⅭＣ",
		'D': "ⅮＤ",
		'E': "ΕЕＥ",
		'F': "ϜＦ",
		'G': "ɢԌＧ",
		'H': "ʜΗНＨ",
		'I': "ΙІⅠＩ",
		'J': "ЈＪ",
		'K': "ΚКＫ",
		'L': "ʟⅬＬ",
		'M': "ΜМⅯＭ",
		'N': "ɴΝＮ",
		'O': "ΟОＯ",
		'P': "ΡРＰ",
		'Q': "Ｑ",
		'R': "ʀＲ",
		'S': "ЅＳ",
		'T': "ΤТＴ",
		'U': "Ｕ",
		'V': "ѴⅤＶ",
		'W': "Ｗ",
		'X': "ΧХⅩＸ",
		'Y': "ʏΥҮＹ",
		'Z': "ΖＺ",
		'_': "＿",
		'a': "ɑаａ",
		'b': "Ьｂ",
		'c': "ϲсⅽｃ",
		'd': "ԁⅾｄ",
		'e': "еｅ",
		'f': "ｆ",
		'g': "ɡｇ",
		'h': "һｈ",
		'i': "ɩіⅰｉ",
		'j': "јｊ",
		'k': "ｋ",
		'l': "ⅼｌ",
		'm': "ⅿｍ",
		'n': "ｎ",
		'o': "οоｏ",
		'p': "рｐ",
		'q': "ｑ",
		'r': "ｒ",
		's': "ѕｓ",
		't': "ｔ",
		'u': "ｕ",
		'v': "νѵⅴｖ",
		'w': "ѡｗ",
		'x': "хⅹｘ",
		'y': "уｙ",
		'z': "ｚ",
	} {
		for _, r := range glyphs {
			m[r] = ascii
		}
	}
	return m
}()
