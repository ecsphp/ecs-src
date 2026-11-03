package rules

import (
	"fmt"
	"strings"
	"testing"
)

// Cases ported from PHP-CS-Fixer's StringLineEndingFixerTest (its
// testWithWhitespacesConfig needs a "\r\n" line ending config blink lacks).
func TestStringLineEndingUpstream(t *testing.T) {
	t.Parallel()
	heredoc := "<?php\n$a=\n<<<EOT\n%s\n\nEOT;\n"
	nowdoc := "<?php\n$a=\n<<<'EOT'\n%s\n\nEOT;\n"
	input := `/**
* @SWG\Get(
*     path="/api/v0/cards",
*     operationId="listCards",
*     tags={"Банковские карты"},
*     summary="Возвращает список банковских карт."
*  )
*/`
	tpl := func(template, prefix, body string) string {
		return fmt.Sprintf(strings.Replace(template, "<<<", prefix+"<<<", 1), body)
	}
	cr := strings.ReplaceAll(input, "\n", "\r")
	crlf := strings.ReplaceAll(input, "\n", "\r\n")
	runUpstreamFixCases(t, StringLineEnding{}, []upstreamFixCase{
		{name: "#0", expected: "<?php $a = 'my\nmulti\nline\nstring';\r\n", input: "<?php $a = 'my\r\nmulti\nline\r\nstring';\r\n"},
		{name: "#1", expected: "<?php $a = \"my\nmulti\nline\nstring\";\r\n", input: "<?php $a = \"my\r\nmulti\nline\r\nstring\";\r\n"},
		{name: "#2", expected: "<?php $a = \"my\nmulti\nline\nstring\nwith\n$b\ninterpolation\";\r\n", input: "<?php $a = \"my\r\nmulti\nline\r\nstring\nwith\r\n$b\ninterpolation\";\r\n"},
		{name: "#3", expected: tpl(heredoc, "", input), input: tpl(heredoc, "", cr)},
		{name: "#4", expected: tpl(heredoc, "", input), input: tpl(heredoc, "", crlf)},
		{name: "#5", expected: tpl(nowdoc, "", input), input: tpl(nowdoc, "", cr)},
		{name: "#6", expected: tpl(nowdoc, "", input), input: tpl(nowdoc, "", crlf)},
		{name: "#7", expected: tpl(nowdoc, "b", input), input: tpl(nowdoc, "b", crlf)},
		{name: "#8", expected: tpl(nowdoc, "B", input), input: tpl(nowdoc, "B", crlf)},
		{name: "#9", expected: tpl(heredoc, "b", input), input: tpl(heredoc, "b", crlf)},
		{name: "#10", expected: tpl(heredoc, "B", input), input: tpl(heredoc, "B", crlf)},
		{name: "not T_CLOSE_TAG, do T_INLINE_HTML", expected: "<?php foo(); ?>\r\nA\n\n", input: "<?php foo(); ?>\r\nA\r\n\r\n"},
		{name: "#12", expected: "<?php $a = b'my\nmulti\nline\nstring';\r\n", input: "<?php $a = b'my\r\nmulti\nline\r\nstring';\r\n"},
	})
}

func TestStringLineEnding(t *testing.T) {
	t.Parallel()
	runUpstreamFixCases(t, StringLineEnding{}, []upstreamFixCase{
		// the heredoc start marker keeps its line ending, as PHP-CS-Fixer leaves T_START_HEREDOC
		{name: "heredoc start", expected: "<?php $a = <<<EOT\r\na\nb\nEOT;\n", input: "<?php $a = <<<EOT\r\na\r\nb\r\nEOT;\n"},
		// \R also matches VT, FF, NEL, LS and PS in valid UTF-8
		{name: "other breaks", expected: "<?php $a = 'a\nb\nc\nd\ne\nf';", input: "<?php $a = 'a\vb\fc\u0085d e f';"},
		// a byte 0x85 inside a valid UTF-8 character is no NEL
		{name: "utf-8 byte 0x85", expected: "<?php $a = 'Å\n';", input: "<?php $a = 'Å\r';"},
		// without valid UTF-8 the byte 0x85 is a NEL
		{name: "invalid utf-8", expected: "<?php $a = '\xff\n';", input: "<?php $a = '\xff\x85';"},
		// interpolated code and comments are not strings
		{name: "interpolation", expected: "<?php $a = \"x\n{$b['c\nd']}\n\"; // \r\n", input: "<?php $a = \"x\r\n{$b['c\r\nd']}\r\n\"; // \r\n"},
		{name: "heredoc interpolation", expected: "<?php $a = <<<EOT\nx\n{$b['c\nd']} $e\nEOT;\n", input: "<?php $a = <<<EOT\nx\r\n{$b['c\r\nd']} $e\r\nEOT;\n"},
		{name: "close tag newline", expected: "<?php ?>\r\n<?php ?>\rx\n", input: "<?php ?>\r\n<?php ?>\rx\r"},
	})
}
