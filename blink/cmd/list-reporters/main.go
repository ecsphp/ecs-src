// Command list-reporters prints every rule that reports errors it cannot fix,
// one FQCN per line, for the report-only parity check.
package main

import (
	"fmt"

	"github.com/ecsphp/ecs-src/blink/internal/fixer"
	"github.com/ecsphp/ecs-src/blink/internal/fixer/rules"
)

func main() {
	for _, rule := range rules.All() {
		if _, ok := rule.(fixer.ErrorReporter); ok {
			fmt.Println(rule.Name())
		}
	}
}
