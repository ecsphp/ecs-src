package rules

import (
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitMockFixer.php
//
// PhpUnitMock replaces ->getMock and ->getMockWithoutInvokingTheOriginalConstructor
// with ->createMock / ->createPartialMock inside test classes. The default
// target (newest) also rewrites a two-argument getMock to createPartialMock.
// Risky: assumes PHPUnit's own methods.
type PhpUnitMock struct {
	target string
}

func (PhpUnitMock) Name() string {
	return `PhpCsFixer\Fixer\PhpUnit\PhpUnitMockFixer`
}

func (PhpUnitMock) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/PhpUnit/PhpUnitMockFixer.php"
}

func (f PhpUnitMock) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["target"].(string); ok {
		f.target = v
	}
	return f
}

func (f PhpUnitMock) Fix(s *tokens.Stream) bool {
	fixCreatePartial := phpUnitTargetFulfills(f.target, "5.5")
	changed := false
	for _, cls := range phpUnitTestClasses(s) {
		start, end := cls[0], cls[1]
		for index := start; index < end && index < s.Len(); index++ {
			if !puIsObjectOperator(s, index) {
				continue
			}
			index = sigNext(s, index)
			if index < 0 {
				break
			}
			if t := s.At(index); t.Kind != token.Ident {
				continue
			}
			name := s.At(index).Value
			switch {
			case strings.EqualFold(name, "getMockWithoutInvokingTheOriginalConstructor"):
				s.SetValue(index, "createMock")
				changed = true
			case strings.EqualFold(name, "getMock"):
				open := sigNext(s, index)
				if !isPunctVal(s, open, "(") {
					continue
				}
				n := len(aliasArguments(s, open, s.MatchForward(open)))
				if n == 1 {
					s.SetValue(index, "createMock")
					changed = true
				} else if n == 2 && fixCreatePartial {
					s.SetValue(index, "createPartialMock")
					changed = true
				}
			}
		}
	}
	return changed
}
