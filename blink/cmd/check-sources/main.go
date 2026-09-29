// Command check-sources verifies that every registered fixer links to a
// reachable PHP-CS-Fixer source file on GitHub. It is run in CI so a renamed or
// removed upstream rule fails the build.
package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"blink/internal/fixer/rules"
)

func main() {
	client := &http.Client{Timeout: 20 * time.Second}
	failed := false

	for _, f := range rules.All() {
		url := f.SourceURL()

		req, err := http.NewRequest(http.MethodHead, url, nil)
		if err != nil {
			fmt.Printf("FAIL %s: %v\n", f.Name(), err)
			failed = true
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("FAIL %s: %v\n", f.Name(), err)
			failed = true
			continue
		}
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("FAIL %s: %s -> %d\n", f.Name(), url, resp.StatusCode)
			failed = true
			continue
		}
		fmt.Printf("OK   %s\n", url)
	}

	if failed {
		os.Exit(1)
	}
	fmt.Println("All fixer sources reachable.")
}
