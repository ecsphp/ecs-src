package rules

import (
	"strings"
	"testing"

	"blink/internal/lexer"
	"blink/internal/tokens"
)

// messyClass is one deliberately badly-formatted class; repeating it builds a
// larger file so the benchmark exercises the full rule set on realistic input.
const messyClass = `
class Example%d
{
    public    function compute($a,$b ,$c)
    {
        $values = array(1,2,   3) ;
        $sum=0;
        foreach( $values as $v ){
            $sum = $sum+$v;
        }
        if($a==$b) { return $sum  ; }
        return $sum+$c;
    }
}
`

// genSource builds a PHP file of n messy classes.
func genSource(n int) string {
	var b strings.Builder
	b.WriteString("<?php\n\nnamespace App;\n")
	for i := range n {
		b.WriteString(strings.Replace(messyClass, "%d", string(rune('A'+i%26)), 1))
	}
	return b.String()
}

// BenchmarkFixPipeline measures lex + all rules + render, the per-file hot path.
// Sizes grow geometrically so a super-linear cost in the stream mutations shows
// up as a growing ns/class across the sub-benchmarks.
func BenchmarkFixPipeline(b *testing.B) {
	all := All()
	for _, n := range []int{1, 8, 64, 256} {
		src := genSource(n)
		b.Run(sizeName(n), func(b *testing.B) {
			b.SetBytes(int64(len(src)))
			b.ReportAllocs()
			for b.Loop() {
				stream := tokens.New(lexer.Lex(src))
				for _, rule := range all {
					rule.Fix(stream)
				}
				_ = stream.Render()
			}
		})
	}
}

func sizeName(n int) string {
	switch {
	case n >= 256:
		return "256classes"
	case n >= 64:
		return "64classes"
	case n >= 8:
		return "8classes"
	default:
		return "1class"
	}
}
