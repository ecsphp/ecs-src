// Command check-sources verifies that every registered fixer links to a
// reachable PHP-CS-Fixer source file on GitHub. It is run in CI so a renamed or
// removed upstream rule fails the build. GitHub rate-limits bursts of HEAD
// requests with 503/429, so each URL is retried with backoff before failing.
package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"blink/internal/fixer/rules"
)

const maxAttempts = 4

// checkURL returns the final status code (or an error) for url, retrying on a
// 5xx or 429 that GitHub returns when throttling a burst of requests.
func checkURL(client *http.Client, url string) (int, error) {
	var lastStatus int
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequest(http.MethodHead, url, nil)
		if err != nil {
			return 0, err
		}
		req.Header.Set("User-Agent", "blink-check-sources")
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
		} else {
			_ = resp.Body.Close()
			lastStatus, lastErr = resp.StatusCode, nil
			if resp.StatusCode == http.StatusOK {
				return resp.StatusCode, nil
			}
			// a 4xx other than 429 is a genuine missing/renamed source, not throttling
			if resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode < 500 {
				return resp.StatusCode, nil
			}
		}
		if attempt < maxAttempts {
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
		}
	}
	return lastStatus, lastErr
}

func main() {
	client := &http.Client{Timeout: 20 * time.Second}
	failed := false

	for _, f := range rules.All() {
		url := f.SourceURL()
		status, err := checkURL(client, url)
		switch {
		case err != nil:
			fmt.Printf("FAIL %s: %v\n", f.Name(), err)
			failed = true
		case status != http.StatusOK:
			fmt.Printf("FAIL %s: %s -> %d\n", f.Name(), url, status)
			failed = true
		default:
			fmt.Printf("OK   %s\n", url)
		}
		time.Sleep(150 * time.Millisecond) // throttle so GitHub does not rate-limit the burst
	}

	if failed {
		os.Exit(1)
	}
	fmt.Println("All fixer sources reachable.")
}
