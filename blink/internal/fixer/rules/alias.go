package rules

import (
	"strings"

	"blink/internal/token"
	"blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Alias/NoMixedEchoPrintFixer.php
//
// NoMixedEchoPrint rewrites the "print" language construct to "echo" when it is
// used as a statement (the default "use: echo"). Uses of print as an expression
// ("$x = print 'y'") are left alone.
type NoMixedEchoPrint struct{}

func (NoMixedEchoPrint) Name() string {
	return `PhpCsFixer\Fixer\Alias\NoMixedEchoPrintFixer`
}

func (NoMixedEchoPrint) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Alias/NoMixedEchoPrintFixer.php"
}

func (NoMixedEchoPrint) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Keyword || strings.ToLower(t.Value) != "print" {
			continue
		}
		prev, ok := prevSignificant(s, i)
		if !ok {
			continue
		}
		// "print" is a statement only after ; { } ) an open tag, or "else"
		convertible := prev.Kind == token.OpenTag ||
			(prev.Kind == token.Punct && (prev.Value == ";" || prev.Value == "{" || prev.Value == "}" || prev.Value == ")")) ||
			(prev.Kind == token.Keyword && strings.ToLower(prev.Value) == "else")
		if !convertible {
			continue
		}
		s.SetValue(i, "echo")
		changed = true
	}
	return changed
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Alias/NoAliasFunctionsFixer.php
//
// NoAliasFunctions replaces internal function aliases with their canonical name
// ("sizeof" -> "count", "join" -> "implode"). It covers the default sets
// (@internal, @IMAP, @pg) and only rewrites global function calls, leaving
// methods, declarations and namespaced names alone.
type NoAliasFunctions struct{}

func (NoAliasFunctions) Name() string {
	return `PhpCsFixer\Fixer\Alias\NoAliasFunctionsFixer`
}

func (NoAliasFunctions) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Alias/NoAliasFunctionsFixer.php"
}

// functionAliases are the default sets (@internal, @IMAP, @pg) from PHP-CS-Fixer.
var functionAliases = map[string]string{
	// @internal
	"diskfreespace":           "disk_free_space",
	"dns_check_record":        "checkdnsrr",
	"dns_get_mx":              "getmxrr",
	"session_commit":          "session_write_close",
	"stream_register_wrapper": "stream_wrapper_register",
	"set_file_buffer":         "stream_set_write_buffer",
	"socket_set_blocking":     "stream_set_blocking",
	"socket_get_status":       "stream_get_meta_data",
	"socket_set_timeout":      "stream_set_timeout",
	"socket_getopt":           "socket_get_option",
	"socket_setopt":           "socket_set_option",
	"chop":                    "rtrim",
	"close":                   "closedir",
	"doubleval":               "floatval",
	"fputs":                   "fwrite",
	"get_required_files":      "get_included_files",
	"ini_alter":               "ini_set",
	"is_double":               "is_float",
	"is_integer":              "is_int",
	"is_long":                 "is_int",
	"is_real":                 "is_float",
	"is_writeable":            "is_writable",
	"join":                    "implode",
	"key_exists":              "array_key_exists",
	"magic_quotes_runtime":    "set_magic_quotes_runtime",
	"pos":                     "current",
	"show_source":             "highlight_file",
	"sizeof":                  "count",
	"strchr":                  "strstr",
	"user_error":              "trigger_error",
	// @IMAP
	"imap_create":         "imap_createmailbox",
	"imap_fetchtext":      "imap_body",
	"imap_header":         "imap_headerinfo",
	"imap_listmailbox":    "imap_list",
	"imap_listsubscribed": "imap_lsub",
	"imap_rename":         "imap_renamemailbox",
	"imap_scan":           "imap_listscan",
	"imap_scanmailbox":    "imap_listscan",
	// @pg
	"pg_exec": "pg_query",
}

func (NoAliasFunctions) Fix(s *tokens.Stream) bool {
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Ident {
			continue
		}
		canonical, ok := functionAliases[strings.ToLower(t.Value)]
		if !ok {
			continue
		}
		if nextSignificantValue(s, i) != "(" {
			continue // not a call
		}
		if !isGlobalFunctionCall(s, i) {
			continue
		}
		s.SetValue(i, canonical)
		changed = true
	}
	return changed
}

// isGlobalFunctionCall reports whether the name at index i is a call to a global
// function - not a method, a declaration, a "new", or a namespaced name.
func isGlobalFunctionCall(s *tokens.Stream, i int) bool {
	prev, ok := prevSignificant(s, i)
	if !ok {
		return true
	}
	if prev.Kind == token.Punct {
		switch prev.Value {
		case "->", "?->", "::":
			return false // method or static call
		case `\`:
			// qualified name: namespaced when a name precedes the backslash,
			// global when the backslash is leading ("\sizeof(")
			bsIdx := prevSignificantIndex(s, i)
			if before, ok2 := prevSignificant(s, bsIdx); ok2 && before.Kind == token.Ident {
				return false
			}
			return true
		}
	}
	if prev.Kind == token.Keyword {
		switch strings.ToLower(prev.Value) {
		case "function", "const", "new", "goto":
			return false // declaration, constant, instantiation or label
		}
	}
	return true
}
