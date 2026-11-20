package rules

import (
	"os"
	"strconv"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/tokens"
)

// PHP_CodeSniffer: https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Files/ExecutableFileSniff.php
//
// ExecutableFile reports a file with any execute permission bit set. It reads
// the file's mode from disk and fixes nothing.
type ExecutableFile struct{}

func (ExecutableFile) Name() string {
	return `PHP_CodeSniffer\Standards\Generic\Sniffs\Files\ExecutableFileSniff`
}

func (ExecutableFile) SourceURL() string {
	return "https://github.com/PHPCSStandards/PHP_CodeSniffer/blob/4.x/src/Standards/Generic/Sniffs/Files/ExecutableFileSniff.php"
}

func (ExecutableFile) Fix(*tokens.Stream) bool { return false }

// phpcsReadsFileSystem keeps the sniff out of TestPhpcsUpstreamErrors: the mode
// ECS saw depends on the umask of the checkout.
func (ExecutableFile) phpcsReadsFileSystem() {}

func (e ExecutableFile) Errors(s *tokens.Stream) []fixer.CodingStandardError {
	if s.Path() == "" {
		return nil
	}
	info, err := os.Stat(s.Path())
	if err != nil {
		return nil
	}
	perm := info.Mode().Perm()
	if perm&0o111 == 0 {
		return nil
	}
	// PHP's substr(sprintf('%o', fileperms()), -4): the setuid/setgid/sticky
	// digit and the three permission digits
	mode := uint32(perm)
	if info.Mode()&os.ModeSetuid != 0 {
		mode |= 0o4000
	}
	if info.Mode()&os.ModeSetgid != 0 {
		mode |= 0o2000
	}
	if info.Mode()&os.ModeSticky != 0 {
		mode |= 0o1000
	}
	octal := strconv.FormatUint(uint64(mode), 8)
	for len(octal) < 4 {
		octal = "0" + octal
	}
	return []fixer.CodingStandardError{{
		Line:    1,
		Message: "A PHP file should not be executable; found file permissions set to " + octal,
		Checker: e.Name() + ".Executable",
	}}
}
