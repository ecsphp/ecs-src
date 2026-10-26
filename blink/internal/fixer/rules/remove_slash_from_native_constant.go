package rules

import (
	"blink/internal/token"
	"blink/internal/tokens"
)

// Symplify: https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/ConstantNotation/RemoveSlashFromNativeConstantFixer.php
//
// RemoveSlashFromNativeConstant strips the redundant leading backslash from the
// true, false and null constants (\true -> true).
type RemoveSlashFromNativeConstant struct{}

func (RemoveSlashFromNativeConstant) Name() string {
	return `Symplify\CodingStandard\Fixer\ConstantNotation\RemoveSlashFromNativeConstantFixer`
}

func (RemoveSlashFromNativeConstant) SourceURL() string {
	return "https://github.com/ecsphp/ecs-src/blob/main/packages/coding-standard/src/Fixer/ConstantNotation/RemoveSlashFromNativeConstantFixer.php"
}

func (RemoveSlashFromNativeConstant) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		if s.At(i).Kind != token.Punct || s.At(i).Value != `\` {
			continue
		}

		next := nextSignificantIndex(s, i)
		if next < 0 || s.At(next).Kind != token.Ident {
			continue
		}

		lower := lowerASCII(s.At(next).Value)
		if lower != "true" && lower != "false" && lower != "null" {
			continue
		}

		// the keyword must stand alone, not be a namespace segment like
		// "Foo\true" or "\true\Bar"
		if prev := prevSignificantIndex(s, i); prev >= 0 {
			pt := s.At(prev)
			if pt.Kind == token.Ident || (pt.Kind == token.Punct && pt.Value == `\`) {
				continue
			}
		}
		if after := nextSignificantIndex(s, next); after >= 0 {
			at := s.At(after)
			if at.Kind == token.Punct && at.Value == `\` {
				continue
			}
		}

		s.RemoveAt(i)
		changed = true
		i--
	}
	return changed
}
