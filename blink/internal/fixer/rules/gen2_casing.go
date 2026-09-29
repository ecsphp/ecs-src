package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// builtinClassNames maps the lowercased name of a well-known internal PHP class
// to its canonical casing. Only reserved/built-in names appear here: a user
// class cannot be declared with any of these names, so a fully-qualified
// reference to one is unambiguous.
var builtinClassNames = map[string]string{
	// core
	"stdclass":          "stdClass",
	"closure":           "Closure",
	"generator":         "Generator",
	"fiber":             "Fiber",
	"weakmap":           "WeakMap",
	"weakreference":     "WeakReference",
	"stringable":        "Stringable",
	"throwable":         "Throwable",
	"traversable":       "Traversable",
	"iterator":          "Iterator",
	"iteratoraggregate": "IteratorAggregate",
	"arrayaccess":       "ArrayAccess",
	"countable":         "Countable",
	"jsonserializable":  "JsonSerializable",
	"unitenum":          "UnitEnum",
	"backedenum":        "BackedEnum",
	"attribute":         "Attribute",
	// exceptions / errors
	"exception":                "Exception",
	"errorexception":           "ErrorException",
	"error":                    "Error",
	"typeerror":                "TypeError",
	"valueerror":               "ValueError",
	"argumentcounterror":       "ArgumentCountError",
	"arithmeticerror":          "ArithmeticError",
	"divisionbyzeroerror":      "DivisionByZeroError",
	"unhandledmatcherror":      "UnhandledMatchError",
	"jsonexception":            "JsonException",
	"logicexception":           "LogicException",
	"badfunctioncallexception": "BadFunctionCallException",
	"badmethodcallexception":   "BadMethodCallException",
	"domainexception":          "DomainException",
	"invalidargumentexception": "InvalidArgumentException",
	"lengthexception":          "LengthException",
	"outofrangeexception":      "OutOfRangeException",
	"runtimeexception":         "RuntimeException",
	"outofboundsexception":     "OutOfBoundsException",
	"overflowexception":        "OverflowException",
	"rangeexception":           "RangeException",
	"underflowexception":       "UnderflowException",
	"unexpectedvalueexception": "UnexpectedValueException",
	// SPL data structures / iterators
	"arrayobject":         "ArrayObject",
	"arrayiterator":       "ArrayIterator",
	"splstack":            "SplStack",
	"splqueue":            "SplQueue",
	"spldoublylinkedlist": "SplDoublyLinkedList",
	"splfixedarray":       "SplFixedArray",
	"splheap":             "SplHeap",
	"splminheap":          "SplMinHeap",
	"splmaxheap":          "SplMaxHeap",
	"splpriorityqueue":    "SplPriorityQueue",
	"splobjectstorage":    "SplObjectStorage",
	"splfileinfo":         "SplFileInfo",
	"splfileobject":       "SplFileObject",
	"spltempfileobject":   "SplTempFileObject",
	// date / time
	"datetime":          "DateTime",
	"datetimeimmutable": "DateTimeImmutable",
	"datetimeinterface": "DateTimeInterface",
	"dateinterval":      "DateInterval",
	"dateperiod":        "DatePeriod",
	"datetimezone":      "DateTimeZone",
	// reflection
	"reflectionclass":     "ReflectionClass",
	"reflectionmethod":    "ReflectionMethod",
	"reflectionproperty":  "ReflectionProperty",
	"reflectionfunction":  "ReflectionFunction",
	"reflectionparameter": "ReflectionParameter",
	"reflectionnamedtype": "ReflectionNamedType",
	"reflectionexception": "ReflectionException",
	"reflectionenum":      "ReflectionEnum",
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Casing/ClassReferenceNameCasingFixer.php
//
// ClassReferenceNameCasing normalises the casing of a reference to an internal
// PHP class to its canonical spelling ("\exception" -> "\Exception"). To stay
// provable on flat tokens it only touches a fully-qualified reference ("\Name")
// to a known built-in class, which is global and unaffected by namespace or use
// imports, so the correct casing is unambiguous.
type ClassReferenceNameCasing struct{}

func (ClassReferenceNameCasing) Name() string {
	return `PhpCsFixer\Fixer\Casing\ClassReferenceNameCasingFixer`
}

func (ClassReferenceNameCasing) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Casing/ClassReferenceNameCasingFixer.php"
}

func (ClassReferenceNameCasing) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Ident {
			continue
		}
		canonical, ok := builtinClassNames[strings.ToLower(t.Value)]
		if !ok || canonical == t.Value {
			continue
		}
		// only a fully-qualified reference "\Name" is unambiguously the global
		// built-in class
		bs := prevSignificantIndex(s, i)
		if bs < 0 || s.At(bs).Kind != token.Punct || s.At(bs).Value != `\` {
			continue
		}
		// "\Name\..." - the name heads a namespace, it is not the class itself
		if n := nextSignificantIndex(s, i); n >= 0 && s.At(n).Kind == token.Punct && s.At(n).Value == `\` {
			continue
		}
		// "Foo\Name" - part of a namespaced name, not a global built-in
		before := prevSignificantIndex(s, bs)
		if before >= 0 && s.At(before).Kind == token.Ident {
			continue
		}
		if !isClassReferencePosition(s, before, i) {
			continue
		}
		s.SetValue(i, canonical)
		changed = true
	}
	return changed
}

// isClassReferencePosition mirrors the remaining guards of the PHP fixer for the
// name at nameIdx, given the significant token before its leading "\" (before,
// or -1). It rejects positions where a same-spelled function call, constant or
// assignment target - rather than a class name - would sit.
func isClassReferencePosition(s *tokens.Stream, before, nameIdx int) bool {
	next := nextSignificantIndex(s, nameIdx)

	// a name wrapped by matching block edges (e.g. "[\Name]") is ambiguous
	if before >= 0 && next >= 0 {
		if isBlockOpenOrComma(s.At(before)) && isBlockCloseOrComma(s.At(next)) {
			return false
		}
	}

	// "new \Name" is always a class reference
	if before >= 0 && s.At(before).Kind == token.Keyword && strings.EqualFold(s.At(before).Value, "new") {
		return true
	}

	// "\Name(" / "\Name;" / "\Name=" / "\Name?>" reads as a call, constant or
	// target, never a class name
	if next >= 0 {
		nt := s.At(next)
		if nt.Kind == token.CloseTag {
			return false
		}
		switch nt.Value {
		case "(", ";", "=":
			return false
		}
	}
	return true
}

func isBlockOpenOrComma(t token.Token) bool {
	if t.Kind != token.Punct {
		return false
	}
	switch t.Value {
	case ",", "(", "[", "{":
		return true
	}
	return false
}

func isBlockCloseOrComma(t token.Token) bool {
	if t.Kind != token.Punct {
		return false
	}
	switch t.Value {
	case ",", ")", "]", "}":
		return true
	}
	return false
}
