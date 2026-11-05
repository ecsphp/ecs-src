package rules

import (
	"maps"
	"slices"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/token"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Alias/NoMixedEchoPrintFixer.php
//
// NoMixedEchoPrint writes output with one language construct. With the default
// `use: echo`, "print" becomes "echo" when it is used as a statement (after ;
// { } ) the open tag or "else"); print as an expression ("$x = print 'y'") is
// left alone. With `use: print`, "echo" becomes "print" when it has a single
// argument: no comma up to its ";" or close tag, outside parentheses and array
// literals. "<?=" is never touched.
type NoMixedEchoPrint struct {
	Print bool // use: print (default echo)
}

func (NoMixedEchoPrint) Name() string {
	return `PhpCsFixer\Fixer\Alias\NoMixedEchoPrintFixer`
}

func (NoMixedEchoPrint) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Alias/NoMixedEchoPrintFixer.php"
}

// WithConfig reads `use` (`echo` or `print`).
func (f NoMixedEchoPrint) WithConfig(config map[string]any) fixer.Fixer {
	if v, ok := config["use"].(string); ok {
		f.Print = v == "print"
	}
	return f
}

// IsCandidate rules out a file without the construct to replace.
func (f NoMixedEchoPrint) IsCandidate(s *tokens.Stream) bool {
	if f.Print {
		return s.Index().HasWord("echo")
	}
	return s.Index().HasWord("print")
}

func (f NoMixedEchoPrint) Fix(s *tokens.Stream) bool {
	from, to := "print", "echo"
	if f.Print {
		from, to = "echo", "print"
	}
	changed := false
	for i := 0; i < s.Len(); i++ {
		if !kwIs(s.At(i), from) {
			continue
		}
		if f.Print && !singleEchoArgument(s, i) || !f.Print && !printIsStatement(s, i) {
			continue
		}
		s.SetValue(i, to)
		changed = true
	}
	return changed
}

// printIsStatement reports whether the "print" at i starts a statement: it
// follows ; { } ) an open tag, or "else".
func printIsStatement(s *tokens.Stream, i int) bool {
	prev, ok := prevSignificant(s, i)
	if !ok {
		return false
	}
	return prev.Kind == token.OpenTag ||
		(prev.Kind == token.Punct && (prev.Value == ";" || prev.Value == "{" || prev.Value == "}" || prev.Value == ")")) ||
		kwIs(prev, "else")
}

// singleEchoArgument reports whether the "echo" at i has no comma up to the
// next ";" or close tag, skipping parentheses and array literals as upstream
// does (other brackets, such as a match body, are not skipped).
func singleEchoArgument(s *tokens.Stream, i int) bool {
	end := s.Len()
	for j := i + 1; j < s.Len(); j++ {
		if s.At(j).Kind == token.CloseTag || isPunctVal(s, j, ";") {
			end = j
			break
		}
	}
	for j := sigNext(s, i); j >= 0 && j < end; j++ {
		if isPunctVal(s, j, "(") || isPunctVal(s, j, "[") && isArrayLiteralOpen(s, j) {
			if j = s.MatchForward(j); j < 0 {
				return true
			}
		}
		if isPunctVal(s, j, ",") {
			return false
		}
	}
	return true
}

// PHP-CS-Fixer: https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Alias/NoAliasFunctionsFixer.php
//
// NoAliasFunctions replaces function aliases with their master function
// ("sizeof" -> "count", "join" -> "implode") for the configured `sets`
// (default @internal, @IMAP, @pg; @all takes every set). Only global function
// calls are renamed, leaving methods, declarations and namespaced names alone.
// The @time set renames mktime() and gmmktime() to time() only when called
// without arguments. @snmp, which upstream still accepts but no longer
// defines (configuring it fails there), renames nothing.
type NoAliasFunctions struct {
	aliases map[string]functionAlias // nil: the default sets
	names   []string                 // the keys of aliases, for IsCandidate
}

// functionAlias is the master function an alias is renamed to; noArgs limits
// the rename to calls without arguments.
type functionAlias struct {
	master string
	noArgs bool
}

func (NoAliasFunctions) Name() string {
	return `PhpCsFixer\Fixer\Alias\NoAliasFunctionsFixer`
}

func (NoAliasFunctions) SourceURL() string {
	return "https://github.com/PHP-CS-Fixer/PHP-CS-Fixer/blob/master/src/Fixer/Alias/NoAliasFunctionsFixer.php"
}

// functionAliasSets are upstream's SETS, in its order.
var functionAliasSets = []struct {
	name    string
	aliases map[string]functionAlias
}{
	{"@internal", map[string]functionAlias{
		"diskfreespace":           {master: "disk_free_space"},
		"dns_check_record":        {master: "checkdnsrr"},
		"dns_get_mx":              {master: "getmxrr"},
		"session_commit":          {master: "session_write_close"},
		"stream_register_wrapper": {master: "stream_wrapper_register"},
		"set_file_buffer":         {master: "stream_set_write_buffer"},
		"socket_set_blocking":     {master: "stream_set_blocking"},
		"socket_get_status":       {master: "stream_get_meta_data"},
		"socket_set_timeout":      {master: "stream_set_timeout"},
		"socket_getopt":           {master: "socket_get_option"},
		"socket_setopt":           {master: "socket_set_option"},
		"chop":                    {master: "rtrim"},
		"close":                   {master: "closedir"},
		"doubleval":               {master: "floatval"},
		"fputs":                   {master: "fwrite"},
		"get_required_files":      {master: "get_included_files"},
		"ini_alter":               {master: "ini_set"},
		"is_double":               {master: "is_float"},
		"is_integer":              {master: "is_int"},
		"is_long":                 {master: "is_int"},
		"is_real":                 {master: "is_float"},
		"is_writeable":            {master: "is_writable"},
		"join":                    {master: "implode"},
		"key_exists":              {master: "array_key_exists"},
		"magic_quotes_runtime":    {master: "set_magic_quotes_runtime"},
		"pos":                     {master: "current"},
		"show_source":             {master: "highlight_file"},
		"sizeof":                  {master: "count"},
		"strchr":                  {master: "strstr"},
		"user_error":              {master: "trigger_error"},
	}},
	{"@IMAP", map[string]functionAlias{
		"imap_create":         {master: "imap_createmailbox"},
		"imap_fetchtext":      {master: "imap_body"},
		"imap_header":         {master: "imap_headerinfo"},
		"imap_listmailbox":    {master: "imap_list"},
		"imap_listsubscribed": {master: "imap_lsub"},
		"imap_rename":         {master: "imap_renamemailbox"},
		"imap_scan":           {master: "imap_listscan"},
		"imap_scanmailbox":    {master: "imap_listscan"},
	}},
	{"@ldap", map[string]functionAlias{
		"ldap_close":  {master: "ldap_unbind"},
		"ldap_modify": {master: "ldap_mod_replace"},
	}},
	{"@mysqli", map[string]functionAlias{
		"mysqli_execute":       {master: "mysqli_stmt_execute"},
		"mysqli_set_opt":       {master: "mysqli_options"},
		"mysqli_escape_string": {master: "mysqli_real_escape_string"},
	}},
	{"@pg", map[string]functionAlias{
		"pg_exec": {master: "pg_query"},
	}},
	{"@oci", map[string]functionAlias{
		"oci_free_cursor": {master: "oci_free_statement"},
	}},
	{"@odbc", map[string]functionAlias{
		"odbc_do":              {master: "odbc_exec"},
		"odbc_field_precision": {master: "odbc_field_len"},
	}},
	{"@mbreg", map[string]functionAlias{
		"mbereg":                {master: "mb_ereg"},
		"mbereg_match":          {master: "mb_ereg_match"},
		"mbereg_replace":        {master: "mb_ereg_replace"},
		"mbereg_search":         {master: "mb_ereg_search"},
		"mbereg_search_getpos":  {master: "mb_ereg_search_getpos"},
		"mbereg_search_getregs": {master: "mb_ereg_search_getregs"},
		"mbereg_search_init":    {master: "mb_ereg_search_init"},
		"mbereg_search_pos":     {master: "mb_ereg_search_pos"},
		"mbereg_search_regs":    {master: "mb_ereg_search_regs"},
		"mbereg_search_setpos":  {master: "mb_ereg_search_setpos"},
		"mberegi":               {master: "mb_eregi"},
		"mberegi_replace":       {master: "mb_eregi_replace"},
		"mbregex_encoding":      {master: "mb_regex_encoding"},
		"mbsplit":               {master: "mb_split"},
	}},
	{"@openssl", map[string]functionAlias{
		"openssl_get_publickey":  {master: "openssl_pkey_get_public"},
		"openssl_get_privatekey": {master: "openssl_pkey_get_private"},
	}},
	{"@sodium", map[string]functionAlias{
		"sodium_crypto_scalarmult_base": {master: "sodium_crypto_box_publickey_from_secretkey"},
	}},
	{"@exif", map[string]functionAlias{
		"read_exif_data": {master: "exif_read_data"},
	}},
	{"@ftp", map[string]functionAlias{
		"ftp_quit": {master: "ftp_close"},
	}},
	{"@posix", map[string]functionAlias{
		"posix_errno": {master: "posix_get_last_error"},
	}},
	{"@pcntl", map[string]functionAlias{
		"pcntl_errno": {master: "pcntl_get_last_error"},
	}},
	{"@time", map[string]functionAlias{
		"mktime":   {master: "time", noArgs: true},
		"gmmktime": {master: "time", noArgs: true},
	}},
}

// defaultFunctionAliases is the default configuration: @internal, @IMAP, @pg.
var defaultFunctionAliases = NoAliasFunctions{}.withSets([]string{"@internal", "@IMAP", "@pg"})

// WithConfig reads `sets`.
func (f NoAliasFunctions) WithConfig(config map[string]any) fixer.Fixer {
	if sets, ok := phpdocTagsStringList(config["sets"]); ok {
		return f.withSets(sets)
	}
	// a PHP list with gaps, as array_diff gives, is dumped as an object
	if sets, ok := config["sets"].(map[string]any); ok {
		names := make([]string, 0, len(sets))
		for _, set := range sets {
			if name, ok := set.(string); ok {
				names = append(names, name)
			}
		}
		return f.withSets(names)
	}
	return f
}

// withSets merges the named sets, as configurePostNormalisation does.
func (NoAliasFunctions) withSets(sets []string) NoAliasFunctions {
	f := NoAliasFunctions{aliases: map[string]functionAlias{}}
	for _, name := range sets {
		for _, set := range functionAliasSets {
			if name == "@all" || set.name == name {
				maps.Copy(f.aliases, set.aliases)
			}
		}
		if name == "@all" {
			break
		}
	}
	f.names = slices.Sorted(maps.Keys(f.aliases))
	return f
}

// config returns f, or the default configuration for the zero value.
func (f NoAliasFunctions) config() NoAliasFunctions {
	if f.aliases == nil {
		return defaultFunctionAliases
	}
	return f
}

// IsCandidate rules out a file without the trigger: only a call to one of the aliases is renamed.
func (f NoAliasFunctions) IsCandidate(s *tokens.Stream) bool {
	return s.Index().HasAnyWord(f.config().names...)
}

func (f NoAliasFunctions) Fix(s *tokens.Stream) bool {
	aliases := f.config().aliases
	changed := false
	for i := 0; i < s.Len(); i++ {
		t := s.At(i)
		if t.Kind != token.Ident {
			continue
		}
		alias, ok := aliases[strings.ToLower(t.Value)]
		if !ok {
			continue
		}
		open := sigNext(s, i)
		if open < 0 || !isPunctVal(s, open, "(") {
			continue // not a call
		}
		if !isGlobalFunctionCall(s, i) {
			continue
		}
		if alias.noArgs && !isPunctVal(s, sigNext(s, open), ")") {
			continue
		}
		s.SetValue(i, alias.master)
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
