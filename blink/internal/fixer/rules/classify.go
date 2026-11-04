package rules

import "github.com/ecsphp/ecs-src/blink/internal/tokens"

// Exported for the extend package.

func NextMeaningful(s *tokens.Stream, i int) int  { return sigNext(s, i) }
func PrevMeaningful(s *tokens.Stream, i int) int  { return sigPrev(s, i) }
func IsTypeColon(s *tokens.Stream, i int) bool    { return isTypeColon(s, i) }
func IsNullableType(s *tokens.Stream, i int) bool { return isNullableType(s, i) }
