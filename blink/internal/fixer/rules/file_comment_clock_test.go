package rules

import "time"

// the FileComment fixtures embed the year ECS generated them in; regenerating
// them with bin/phpcs-fixtures.sh in a later year means bumping this one too
func init() {
	fileCommentNow = func() time.Time { return time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC) }
}
