package rules

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/PHP/CommentedOutCodeSniff.php
//
// CommentedOutCode reports a comment more than maxPercentage (35) percent of
// whose tokens are code, when tokenized as PHP. The sniff warns, but ECS
// escalates its warnings to errors. It fixes nothing.
type CommentedOutCode struct{}

const commentedOutCodeMaxPercentage = 35

var (
	commentedOutCodeAnnotation = regexp.MustCompile(`^\s*@[A-Za-z()._-]+\s*$`)
	commentedOutCodeFrames     = regexp.MustCompile(`[-=#*]{2,}`)
	commentedOutCodeDigits     = regexp.MustCompile(`\d+`)
)

func (CommentedOutCode) Name() string {
	return `PHP_CodeSniffer\Standards\Squiz\Sniffs\PHP\CommentedOutCodeSniff`
}

func (CommentedOutCode) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Squiz/Sniffs/PHP/CommentedOutCodeSniff.php"
}

func (CommentedOutCode) Fix(*tokens.Stream) bool { return false }

func (r CommentedOutCode) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	return squizReport(s, r.Name(), func(f *slvFile, _ *phpcsScope, report func(i int, code, msg string)) {
		for ptr := 0; ptr < len(f.toks); ptr++ {
			if f.toks[ptr].kind != token.Comment || isPhpcsAnnotationComment(f.toks[ptr].orig) {
				continue
			}
			if strings.HasPrefix(f.toks[ptr].content, "//end ") {
				continue
			}
			next, percent := commentedOutCodeScan(f, ptr)
			if percent > commentedOutCodeMaxPercentage {
				report(ptr, "Found", fmt.Sprintf("This comment is %d%% valid code; is this commented out code?", min(100, percent)))
			}
			ptr = next - 1
		}
	})
}

// commentedOutCodeScan gathers the comment block starting at ptr and returns
// the token after it with the percentage of code in it (0 when not measured).
func commentedOutCodeScan(f *slvFile, ptr int) (int, int) {
	content := ""
	lastLine := f.toks[ptr].line
	block := strings.HasPrefix(f.toks[ptr].content, "/*")
	last := ptr
	for i := ptr; i < len(f.toks); i++ {
		t := f.toks[i]
		if !f.isEmpty(i) {
			break
		}
		if t.kind == token.Whitespace {
			continue
		}
		if t.kind == token.Comment && isPhpcsAnnotationComment(t.orig) {
			lastLine = t.line
			continue
		}
		if !block && lastLine+1 <= t.line && strings.HasPrefix(t.content, "/*") {
			break
		}
		if !block && lastLine+1 < t.line {
			break
		}
		c := strings.Trim(t.content, " \t\n\r\x00\x0B")
		stop := false
		if !block {
			c = strings.TrimPrefix(c, "//")
			c = strings.TrimPrefix(c, "#")
		} else {
			c = strings.TrimPrefix(c, "/**")
			c = strings.TrimPrefix(c, "/*")
			if strings.HasSuffix(c, "*/") {
				c = c[:len(c)-2]
				stop = true
			}
			c = strings.TrimPrefix(c, "*")
		}
		content += c + f.eol
		lastLine = t.line
		last = i
		if stop {
			break
		}
	}
	next := last + 1
	if commentedOutCodeAnnotation.MatchString(content) {
		return next, 0
	}
	content = commentedOutCodeFrames.ReplaceAllString(content, "-")
	content = commentedOutCodeDigits.ReplaceAllString(content, "")
	content = strings.Trim(content, " \t\n\r\x00\x0B")
	if content == "" {
		return next, 0
	}
	return next, commentedOutCodePercent("<?php " + content + " ?>")
}

// commentedOutCodePercent tokenizes src as PHP_CodeSniffer would and returns
// the share of its tokens that look like code, or 0 when it does not parse
// into "<?php ... ?>".
func commentedOutCodePercent(src string) int {
	g := newSlvFile(lexer.Lex(src))
	toks := g.toks
	n := len(toks)
	if n < 2 || toks[0].kind != token.OpenTag || toks[1].kind != token.Whitespace {
		return 0
	}
	if toks[n-1].kind != token.CloseTag {
		return 0
	}
	codes := commentedOutCodeClassify(g)
	codes = codes[2 : len(codes)-1]
	if len(codes) == 0 || codes[len(codes)-1] != commentedOutCodeWhitespace && codes[len(codes)-1] != commentedOutCodeComment {
		return 0
	}
	if codes[len(codes)-1] == commentedOutCodeWhitespace {
		codes = codes[:len(codes)-1]
	}
	numCode, nonWhitespace := 0, 0
	for _, c := range codes {
		if c == commentedOutCodeCode {
			numCode++
		}
		if c != commentedOutCodeWhitespace {
			nonWhitespace++
		}
	}
	if nonWhitespace <= 2 {
		return 0
	}
	return int(math.Ceil(float64(numCode) / float64(len(codes)) * 100))
}

type commentedOutCodeKind uint8

const (
	commentedOutCodeWhitespace commentedOutCodeKind = iota
	commentedOutCodeComment                         // T_COMMENT and the annotations: empty, not code
	commentedOutCodeText                            // T_STRING, ".", goto labels, comparisons, arithmetic
	commentedOutCodeCode
)

// commentedOutCodeContexts are PHP::T_STRING_CONTEXTS: after one of them a
// keyword or true/false/null/self/parent is a T_STRING.
var commentedOutCodeContexts = phpcsSet("->", "?->", "function", "class", "interface", "trait", "enum",
	"extends", "implements", "#[", "new", "const", `\`, "use", "namespace", "::", "goto")

// commentedOutCodeClassify sorts g's tokens the way the sniff counts them.
func commentedOutCodeClassify(g *slvFile) []commentedOutCodeKind {
	out := make([]commentedOutCodeKind, 0, len(g.toks))
	lastCode := ""
	lastIdx := -1
	nextNonEmpty := func(i int) int { return g.nextEffective(i+1, len(g.toks)) }
	prevNonEmpty := func(i int) int { return g.prevEffective(i - 1) }
	for i, t := range g.toks {
		code := ""
		kind := commentedOutCodeCode
		lower := strings.ToLower(t.orig)
		inContext := commentedOutCodeContexts[lastCode]
		switch t.kind {
		case token.Whitespace:
			kind = commentedOutCodeWhitespace
		case token.Comment:
			kind = commentedOutCodeComment
		case token.Punct:
			code = t.orig
			switch {
			case t.orig == ".":
				kind = commentedOutCodeText
			case squizReportIsComparison(g, i), squizReportIsArithmetic(g, i):
				kind = commentedOutCodeText
			}
		case token.AttributeStart:
			code = "#["
		case token.Ident:
			code = "string"
			switch {
			case lower == `\`:
				code = `\`
			case strings.Contains(t.orig, `\`):
				code = "name"
			case lower == "exit" || lower == "die" || lower == "eval":
				if !inContext {
					code = lower
				}
			case lower == "true" || lower == "false" || lower == "null" || lower == "self" || lower == "parent":
				n := nextNonEmpty(i)
				preserve := false
				switch {
				case lastCode == "const":
					preserve = g.is(n, "=")
				case inContext:
					preserve = lastCode != "new" || lower != "self" && lower != "parent"
				case lastIdx >= 0 && g.is(lastIdx, "&"):
					preserve = g.isKeyword(prevNonEmpty(lastIdx), "function")
				default:
					preserve = g.is(n, "(")
				}
				if !preserve {
					code = lower
				}
			case strings.HasPrefix(t.orig, "__") && strings.HasSuffix(t.orig, "__") && commentedOutCodeMagic[lower]:
				code = lower
			}
			if code == "string" || code == "exit" || code == "die" {
				if n := nextNonEmpty(i); g.is(n, ":") {
					if p := prevNonEmpty(i); g.is(p, "(") || g.is(p, ",") {
						code = "param_name"
					}
				}
			}
			if code == "string" {
				kind = commentedOutCodeText
			}
		case token.Keyword:
			code = lower
			n := nextNonEmpty(i)
			if inContext {
				preserve := lastCode == "new" && (code == "class" || code == "static") ||
					(code == "extends" || code == "implements") && lastCode == "class" ||
					code == "namespace" && n >= 0 && g.toks[n].kind == token.Ident && strings.HasPrefix(g.toks[n].orig, `\`) ||
					lastCode == "const" && !g.is(n, "=") && !g.is(n, ";")
				if !preserve {
					code = "string"
				}
			}
			switch code {
			case "array":
				if n >= 0 && !g.is(n, "(") {
					code = "string"
				}
			case "function", "const":
				if lastCode == "use" {
					code = "string"
				}
			case "readonly":
				if g.is(n, "(") {
					code = "string"
				}
			case "fn", "match":
				if !g.is(n, "(") {
					code = "string"
				}
			case "enum":
				if n < 0 || g.toks[n].kind != token.Ident || n != i+2 {
					code = "string"
				}
			}
			if g.is(nextNonEmpty(i), ":") {
				if p := prevNonEmpty(i); g.is(p, "(") || g.is(p, ",") {
					code = "param_name"
				}
			}
			if code == "string" {
				kind = commentedOutCodeText
			}
		}
		if t.kind == token.String && strings.HasPrefix(t.orig, "`") {
			out = append(out, commentedOutCodeBackticks(t.orig, g.eol)...)
		} else {
			out = append(out, kind)
		}
		if kind != commentedOutCodeWhitespace && kind != commentedOutCodeComment {
			lastCode, lastIdx = code, i
		}
	}
	return out
}

// commentedOutCodeMagic are the magic constants PHP gives a token of their own.
var commentedOutCodeMagic = phpcsSet("__class__", "__dir__", "__file__", "__function__", "__line__",
	"__method__", "__namespace__", "__trait__", "__property__", "__halt_compiler")

// commentedOutCodeBackticks counts a shell-exec literal as PHP_CodeSniffer
// tokenizes it: T_BACKTICK, its text as T_ENCAPSED_AND_WHITESPACE (one token
// per line) around each interpolated T_VARIABLE, and a closing T_BACKTICK.
func commentedOutCodeBackticks(v, eol string) []commentedOutCodeKind {
	out := []commentedOutCodeKind{commentedOutCodeCode}
	inner := strings.TrimSuffix(strings.TrimPrefix(v, "`"), "`")
	text := func(piece string) {
		lines := strings.Split(piece, eol)
		for k, line := range lines {
			if k == len(lines)-1 && line == "" {
				break
			}
			out = append(out, commentedOutCodeText)
		}
	}
	at := 0
	for _, m := range commentedOutCodeVariable.FindAllStringIndex(inner, -1) {
		text(inner[at:m[0]])
		out = append(out, commentedOutCodeCode)
		at = m[1]
	}
	text(inner[at:])
	if len(v) > 1 && strings.HasSuffix(v, "`") {
		out = append(out, commentedOutCodeCode)
	}
	return out
}

var commentedOutCodeVariable = regexp.MustCompile(`\$[A-Za-z_\x80-\xff][A-Za-z0-9_\x80-\xff]*`)
