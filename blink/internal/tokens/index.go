package tokens

import (
	"slices"

	"github.com/ecsphp/ecs-src/blink/internal/token"
)

// Index tells which token kinds and which words (Ident and Keyword values) a
// stream holds, so a rule can skip a file that cannot concern it without
// walking its tokens, like PHP-CS-Fixer's Tokens::isTokenKindFound.
//
// It may answer "present" for a word that is not there, never the reverse:
// words go into a Bloom filter, and a word with a non-ASCII byte makes every
// HasWord answer true (strings.EqualFold and strings.ToLower fold some
// non-ASCII letters to ASCII ones, the Kelvin sign to "k" for one).
type Index struct {
	kinds    uint32
	nonASCII bool
	words    [indexWords]uint64
}

// indexWords is the filter size in 64-bit words: 8192 bits keep false
// positives around 1% for a file of a thousand distinct words.
const indexWords = 128

type indexKey struct{}

// Index returns the stream's index, built again only when the tokens changed
// since it was last built.
func (s *Stream) Index() *Index {
	return s.Memo(indexKey{}, func() any { return buildIndex(s.toks) }).(*Index)
}

func buildIndex(toks []token.Token) *Index {
	x := &Index{}
	for i := range toks {
		t := &toks[i]
		x.kinds |= 1 << t.Kind
		if t.Kind != token.Ident && t.Kind != token.Keyword {
			continue
		}
		h, ascii := wordHash(t.Value)
		if !ascii {
			x.nonASCII = true
			continue
		}
		a, b := wordBits(h)
		x.words[a/64] |= 1 << (a % 64)
		x.words[b/64] |= 1 << (b % 64)
	}
	return x
}

// HasKind reports whether a token of kind k is in the stream.
func (x *Index) HasKind(k token.Kind) bool { return x.kinds&(1<<k) != 0 }

// HasWord reports whether an Ident or Keyword token may spell w, ignoring
// ASCII case. A false answer is certain; a true one may not be.
func (x *Index) HasWord(w string) bool {
	if x.nonASCII {
		return true
	}
	h, ascii := wordHash(w)
	if !ascii {
		return true
	}
	a, b := wordBits(h)
	return x.words[a/64]&(1<<(a%64)) != 0 && x.words[b/64]&(1<<(b%64)) != 0
}

// HasAnyWord reports whether HasWord holds for one of ws.
func (x *Index) HasAnyWord(ws ...string) bool {
	return slices.ContainsFunc(ws, x.HasWord)
}

// wordHash is FNV-1a over w with ASCII letters lowercased, and whether w is
// all ASCII.
func wordHash(w string) (uint64, bool) {
	h := uint64(14695981039346656037)
	for i := 0; i < len(w); i++ {
		c := w[i]
		if c >= 0x80 {
			return 0, false
		}
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		h ^= uint64(c)
		h *= 1099511628211
	}
	return h, true
}

// wordBits picks the word's two filter bits from two halves of its hash.
func wordBits(h uint64) (uint, uint) {
	const bits = indexWords * 64
	return uint(h % bits), uint((h >> 32) % bits)
}

type nonASCIIKey struct{}

// HasNonASCII reports whether a token holds a byte outside ASCII. It reads
// every byte of the file, so unlike Index it is only built when asked for.
func (s *Stream) HasNonASCII() bool {
	return s.Memo(nonASCIIKey{}, func() any {
		for i := range s.toks {
			v := s.toks[i].Value
			for j := 0; j < len(v); j++ {
				if v[j] >= 0x80 {
					return true
				}
			}
		}
		return false
	}).(bool)
}
