package rules

import (
	"slices"
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Alias/BacktickToShellExecFixer.php
//
// BacktickToShellExec converts `ls -lah` to shell_exec("ls -lah"); a lone
// `$var` becomes shell_exec($var).
type BacktickToShellExec struct{}

func (BacktickToShellExec) Name() string {
	return `PhpCsFixer\Fixer\Alias\BacktickToShellExecFixer`
}

func (BacktickToShellExec) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Alias/BacktickToShellExecFixer.php"
}

func (BacktickToShellExec) Fix(s *tokens.Stream) bool {
	// the outermost backtick literals, as [first, last] token ranges
	var literals [][2]int
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if !t.IsShellExec() || (t.Part != token.Whole && t.Part != token.Head) {
			continue
		}
		last := i
		if t.Part == token.Head {
			chunks := literalChunks(s, i)
			if chunks == nil {
				continue
			}
			last = chunks[len(chunks)-1]
		}
		literals = append(literals, [2]int{i, last})
		i = last
	}
	changed := false
	for _, literal := range slices.Backward(literals) {
		first, last := literal[0], literal[1]
		if !backtickConvertible(s, first, last) {
			continue
		}
		var arg string
		if first == last {
			arg = `"` + s.At(first).Value[1:len(s.At(first).Value)-1] + `"`
		} else if s.At(first).Value == "`" && s.At(last).Value == "`" && last == first+2 && s.At(first+1).Kind == token.Variable {
			// one T_VARIABLE between the backticks: PHP-CS-Fixer passes it as is
			arg = s.At(first + 1).Value
		} else {
			var b strings.Builder
			for j := first; j <= last; j++ {
				b.WriteString(s.At(j).Value)
			}
			v := b.String()
			arg = `"` + v[1:len(v)-1] + `"`
		}
		s.ReplaceRange(first, last, []token.Token{{Kind: token.Ident, Value: "shell_exec"}, aliasPunct("("), {Kind: token.String, Value: arg}, aliasPunct(")")})
		changed = true
	}
	if changed {
		aliasRelex(s)
	}
	return changed
}

// backtickConvertible reports whether PHP-CS-Fixer converts the literal
// first..last: no T_ENCAPSED_AND_WHITESPACE token in it (nested strings
// included) holds a quote or a backtick. A literal nested in "{$...}" is left
// alone too: PHP-CS-Fixer pairs backticks in order and would break the code.
func backtickConvertible(s *tokens.Stream, first, last int) bool {
	for j := first; j <= last; j++ {
		t := s.At(j)
		if t.Kind != token.String {
			continue
		}
		if j > first && j < last && t.IsShellExec() {
			return false
		}
		if strings.ContainsAny(backtickEncapsedText(t, j == first || j == last), "`\"'") {
			return false
		}
	}
	return true
}

// backtickEncapsedText returns the part of t that PHP lexes as
// T_ENCAPSED_AND_WHITESPACE, without quotes, backticks or heredoc markers ("" for
// a T_CONSTANT_ENCAPSED_STRING). outer marks the first and last chunks of the
// literal being converted.
func backtickEncapsedText(t token.Token, outer bool) string {
	v := t.Value
	if t.Part == token.Whole && !outer && !t.StartsHeredoc() {
		return ""
	}
	if t.OpensString() {
		if t.StartsHeredoc() {
			v = v[strings.IndexByte(v, '\n')+1:]
		} else {
			v = v[strings.IndexAny(v, "\"`")+1:]
		}
	}
	if t.ClosesString() && !t.EndsHeredoc() && v != "" {
		v = v[:len(v)-1]
	}
	return v
}
