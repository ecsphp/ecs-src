package rules

import (
	"strings"

	"blink/internal/fixer"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/DoctrineAnnotation/DoctrineAnnotationSpacesFixer.php
//
// DoctrineAnnotationSpaces fixes whitespace in Doctrine annotations: around
// parentheses, commas, argument assignments and array assignments, each as its
// option says.
type DoctrineAnnotationSpaces struct {
	// opts is the configured options; nil keeps all the defaults.
	opts *doctrineSpacesOptions
}

// doctrineSpacing is a nullable bool option: add the space, remove it, or leave
// it as it is (null).
type doctrineSpacing int

const (
	doctrineSpacingIgnore doctrineSpacing = iota
	doctrineSpacingAdd
	doctrineSpacingRemove
)

type doctrineSpacesOptions struct {
	ignoredTags                  map[string]bool
	aroundParentheses            bool
	aroundCommas                 bool
	beforeArgumentAssignments    doctrineSpacing
	afterArgumentAssignments     doctrineSpacing
	beforeArrayAssignmentsEquals doctrineSpacing
	afterArrayAssignmentsEquals  doctrineSpacing
	beforeArrayAssignmentsColon  doctrineSpacing
	afterArrayAssignmentsColon   doctrineSpacing
}

// defaultDoctrineSpacesOptions are the upstream defaults of every option.
func defaultDoctrineSpacesOptions() doctrineSpacesOptions {
	return doctrineSpacesOptions{
		ignoredTags:                  doctrineIgnoredTags,
		aroundParentheses:            true,
		aroundCommas:                 true,
		beforeArgumentAssignments:    doctrineSpacingRemove,
		afterArgumentAssignments:     doctrineSpacingRemove,
		beforeArrayAssignmentsEquals: doctrineSpacingAdd,
		afterArrayAssignmentsEquals:  doctrineSpacingAdd,
		beforeArrayAssignmentsColon:  doctrineSpacingAdd,
		afterArrayAssignmentsColon:   doctrineSpacingAdd,
	}
}

func (DoctrineAnnotationSpaces) Name() string {
	return `PhpCsFixer\Fixer\DoctrineAnnotation\DoctrineAnnotationSpacesFixer`
}

func (DoctrineAnnotationSpaces) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/DoctrineAnnotation/DoctrineAnnotationSpacesFixer.php"
}

func (f DoctrineAnnotationSpaces) WithConfig(config map[string]any) fixer.Fixer {
	opts := defaultDoctrineSpacesOptions()
	if f.opts != nil {
		opts = *f.opts
	}
	if tags, ok := doctrineIgnoredTagsOption(config); ok {
		opts.ignoredTags = tags
	}
	if v, ok := config["around_parentheses"].(bool); ok {
		opts.aroundParentheses = v
	}
	if v, ok := config["around_commas"].(bool); ok {
		opts.aroundCommas = v
	}
	for key, field := range map[string]*doctrineSpacing{
		"before_argument_assignments":     &opts.beforeArgumentAssignments,
		"after_argument_assignments":      &opts.afterArgumentAssignments,
		"before_array_assignments_equals": &opts.beforeArrayAssignmentsEquals,
		"after_array_assignments_equals":  &opts.afterArrayAssignmentsEquals,
		"before_array_assignments_colon":  &opts.beforeArrayAssignmentsColon,
		"after_array_assignments_colon":   &opts.afterArrayAssignmentsColon,
	} {
		v, ok := config[key]
		if !ok {
			continue
		}
		switch v := v.(type) {
		case nil:
			*field = doctrineSpacingIgnore
		case bool:
			if v {
				*field = doctrineSpacingAdd
			} else {
				*field = doctrineSpacingRemove
			}
		}
	}
	f.opts = &opts
	return f
}

func (f DoctrineAnnotationSpaces) Fix(s *tokens.Stream) bool {
	opts := defaultDoctrineSpacesOptions()
	if f.opts != nil {
		opts = *f.opts
	}
	return applyToDoctrineAnnotations(s, opts.ignoredTags, func(toks []daToken) []daToken {
		if opts.aroundParentheses {
			daFixSpacesAroundParentheses(toks)
		}
		if opts.aroundCommas {
			toks = daFixSpacesAroundCommas(toks)
		}
		return daFixAroundAssignments(toks, opts)
	})
}

// daInsertAt inserts t at index, like Doctrine Annotation Tokens::insertAt.
func daInsertAt(toks []daToken, index int, t daToken) []daToken {
	toks = append(toks, daToken{})
	copy(toks[index+1:], toks[index:])
	toks[index] = t
	return toks
}

// daFixSpacesAroundParentheses mirrors fixSpacesAroundParentheses.
func daFixSpacesAroundParentheses(toks []daToken) {
	inAnnotationUntil := -1
	for index := range toks {
		if inAnnotationUntil >= 0 {
			if index == inAnnotationUntil {
				inAnnotationUntil = -1
				continue
			}
		} else if toks[index].typ == daTAt {
			if end := daGetAnnotationEnd(toks, index); end >= 0 {
				inAnnotationUntil = end + 1
			}
			continue
		}
		if inAnnotationUntil < 0 {
			continue
		}
		var target int
		switch toks[index].typ {
		case daTOpenParen:
			if toks[index-1].typ == daTNone {
				toks[index-1].content = ""
			}
			target = index + 1
		case daTCloseParen:
			target = index - 1
		default:
			continue
		}
		if target < len(toks) && toks[target].typ == daTNone {
			if strings.Contains(toks[target].content, "\n") {
				continue
			}
			toks[target].content = ""
		}
	}
}

// daFixSpacesAroundCommas mirrors fixSpacesAroundCommas. Like upstream, the
// annotation end found at "@" is not shifted by the spaces inserted after it.
func daFixSpacesAroundCommas(toks []daToken) []daToken {
	inAnnotationUntil := -1
	for index := 0; index < len(toks); index++ {
		if inAnnotationUntil >= 0 {
			if index == inAnnotationUntil {
				inAnnotationUntil = -1
				continue
			}
		} else if toks[index].typ == daTAt {
			if end := daGetAnnotationEnd(toks, index); end >= 0 {
				inAnnotationUntil = end
			}
			continue
		}
		if inAnnotationUntil < 0 || toks[index].typ != daTComma {
			continue
		}
		if toks[index-1].typ == daTNone {
			toks[index-1].content = ""
		}
		if index < len(toks)-1 && !daStartsWithSpace(toks[index+1].content) {
			toks = daInsertAt(toks, index+1, daToken{typ: daTNone, content: " "})
		}
	}
	return toks
}

// daStartsWithSpace mirrors Preg::match('/^\s/', $content).
func daStartsWithSpace(content string) bool {
	return content != "" && daIsSpaceByte(content[0])
}

// daFixAroundAssignments mirrors fixAroundAssignments. An insertion before the
// operator shifts it to index+1, which the loop then visits again (upstream
// iterates the live token list); the second visit is a no-op.
func daFixAroundAssignments(toks []daToken, opts doctrineSpacesOptions) []daToken {
	var scopes []int
	for index := 0; index < len(toks); index++ {
		typ := toks[index].typ
		if len(scopes) > 0 && typ == scopes[len(scopes)-1] {
			scopes = scopes[:len(scopes)-1]
			continue
		}
		if typ == daTAt {
			scopes = append(scopes, daTCloseParen)
			continue
		}
		if typ == daTOpenCurly {
			scopes = append(scopes, daTCloseCurly)
			continue
		}
		if len(scopes) == 0 {
			continue
		}
		var before, after doctrineSpacing
		switch end := scopes[len(scopes)-1]; {
		case end == daTCloseParen && typ == daTEquals:
			before, after = opts.beforeArgumentAssignments, opts.afterArgumentAssignments
		case end == daTCloseCurly && typ == daTEquals:
			before, after = opts.beforeArrayAssignmentsEquals, opts.afterArrayAssignmentsEquals
		case end == daTCloseCurly && typ == daTColon:
			before, after = opts.beforeArrayAssignmentsColon, opts.afterArrayAssignmentsColon
		default:
			continue
		}
		toks = daUpdateSpacesAt(toks, index+1, index+1, after)
		toks = daUpdateSpacesAt(toks, index-1, index, before)
	}
	return toks
}

// daUpdateSpacesAt mirrors updateSpacesAt.
func daUpdateSpacesAt(toks []daToken, index, insertIndex int, spacing doctrineSpacing) []daToken {
	if index < 0 || index >= len(toks) {
		return toks // upstream throws on an out-of-range index; leave the block alone
	}
	switch spacing {
	case doctrineSpacingAdd:
		if toks[index].typ != daTNone {
			toks = daInsertAt(toks, insertIndex, daToken{typ: daTNone})
			index = insertIndex
		}
		toks[index].content = " "
	case doctrineSpacingRemove:
		if toks[index].typ == daTNone {
			toks[index].content = ""
		}
	}
	return toks
}
