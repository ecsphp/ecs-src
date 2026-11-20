package rules

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/lexer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/VersionControl/SubversionPropertiesSniff.php
//
// SubversionProperties reports a file whose Subversion properties, read from
// the pre-1.7 working copy files .svn/props and .svn/prop-base next to it, lack
// or differ from svn:keywords "Author Id Revision" and svn:eol-style "native".
// A file without them is not under version control. It fixes nothing.
type SubversionProperties struct{}

func (SubversionProperties) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\VersionControl\SubversionPropertiesSniff`
}

func (SubversionProperties) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/VersionControl/SubversionPropertiesSniff.php"
}

func (SubversionProperties) Fix(*tokens.Stream) bool { return false }

var subversionRequiredProperties = [][2]string{
	{"svn:keywords", "Author Id Revision"},
	{"svn:eol-style", "native"},
}

func (r SubversionProperties) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	keys, props, ok := subversionPropertiesRead(s.Path())
	if !ok {
		return nil
	}
	f := newSlvFile(lexer.Lex(s.Render()))
	openTag := -1
	for i := range f.toks {
		if phpcsIsOpenTag(f, i) {
			openTag = i
			break
		}
	}
	if openTag < 0 {
		return nil
	}
	required := map[string]string{}
	for _, kv := range subversionRequiredProperties {
		required[kv[0]] = kv[1]
		if _, ok := props[kv[0]]; !ok {
			keys = append(keys, kv[0])
		}
	}
	var errs []fixer.CodingStandardError
	add := func(code, msg string) {
		errs = append(errs, fixer.CodingStandardError{Line: f.toks[openTag].line, Message: msg, Checker: r.Name() + "." + code})
	}
	for _, key := range keys {
		value, has := props[key]
		want, isRequired := required[key]
		switch {
		case has && !isRequired:
			add("Unexpected", `Unexpected Subversion property "`+key+`" = "`+value+`"`)
		case !has:
			add("Missing", `Missing Subversion property "`+key+`" = "`+want+`"`)
		case value != want:
			add("NoMatch", `Subversion property "`+key+`" = "`+value+`" does not match "`+want+`"`)
		}
	}
	return errs
}

// subversionPropertiesRead parses the svn hash files of path, in the order
// PHP_CodeSniffer reads them; ok is false when neither exists.
func subversionPropertiesRead(path string) (keys []string, props map[string]string, ok bool) {
	if path == "" {
		return nil, nil, false
	}
	dir, base := filepath.Dir(path), filepath.Base(path)
	props = map[string]string{}
	for _, p := range []string{
		dir + "/.svn/props/" + base + ".svn-work",
		dir + "/.svn/prop-base/" + base + ".svn-base",
	} {
		file, err := os.Open(p)
		if err != nil {
			continue
		}
		ok = true
		r := bufio.NewReader(file)
		readLen := func() int {
			line, _ := r.ReadString('\n')
			line = strings.TrimSpace(line)
			if line == "END" || len(line) < 2 {
				return -1
			}
			n, _ := strconv.Atoi(line[2:])
			return n
		}
		readN := func(n int) string {
			b := make([]byte, n)
			got, _ := io.ReadFull(r, b)
			_, _ = r.ReadByte()
			return string(b[:got])
		}
		for {
			n := readLen()
			if n < 0 {
				break
			}
			key := readN(n)
			m := readLen()
			if m < 0 {
				break
			}
			if _, seen := props[key]; !seen {
				keys = append(keys, key)
			}
			props[key] = readN(m)
		}
		_ = file.Close()
	}
	return keys, props, ok
}
