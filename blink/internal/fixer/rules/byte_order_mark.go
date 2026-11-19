package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Files/ByteOrderMarkSniff.php
//
// ByteOrderMark reports a byte order mark at the start of the file. It fixes
// nothing.
type ByteOrderMark struct{}

func (ByteOrderMark) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\Files\ByteOrderMarkSniff`
}

func (ByteOrderMark) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Files/ByteOrderMarkSniff.php"
}

func (ByteOrderMark) Fix(*tokens.Stream) bool { return false }

// byteOrderMarks are PHP_CodeSniffer's BOM_DEFINITIONS, in its order.
var byteOrderMarks = []struct{ name, bom string }{
	{"UTF-8", "\xef\xbb\xbf"},
	{"UTF-16 (BE)", "\xfe\xff"},
	{"UTF-16 (LE)", "\xff\xfe"},
}

func (b ByteOrderMark) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	if s.Len() == 0 || s.At(0).Kind != token.InlineHTML {
		return nil
	}
	for _, m := range byteOrderMarks {
		if strings.HasPrefix(s.At(0).Value, m.bom) {
			return []fixer.CodingStandardError{{
				Line:    1,
				Message: "File contains " + m.name + " byte order mark, which may corrupt your application",
				Checker: b.Name() + ".Found",
			}}
		}
	}
	return nil
}
