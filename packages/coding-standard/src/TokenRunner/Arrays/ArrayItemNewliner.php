<?php

declare(strict_types=1);

namespace Symplify\CodingStandard\TokenRunner\Arrays;

use PhpCsFixer\Tokenizer\Token;
use PhpCsFixer\Tokenizer\Tokens;
use PhpCsFixer\WhitespacesFixerConfig;
use Symplify\CodingStandard\TokenRunner\ValueObject\BlockInfo;

final readonly class ArrayItemNewliner
{
    public function __construct(
        private WhitespacesFixerConfig $whitespacesFixerConfig
    ) {
    }

    /**
     * @param Tokens<Token> $tokens
     */
    public function fixArrayOpener(Tokens $tokens, BlockInfo $blockInfo): void
    {
        $depth = 0;
        for ($position = $blockInfo->getEnd() - 1; $position >= $blockInfo->getStart() + 1; --$position) {
            $content = $tokens[$position]->getContent();

            // walking backwards: a closer enters a nested block, an opener leaves it -
            // only commas at the array's own level separate its items, so a comma
            // inside a nested closure body or "[$a, $b]" destructuring is left alone
            if (in_array($content, [')', ']', '}'], true)) {
                ++$depth;
                continue;
            }

            if (in_array($content, ['(', '[', '{'], true)) {
                --$depth;
                continue;
            }

            if ($depth !== 0 || $content !== ',') {
                continue;
            }

            $nextTokenPosition = $position + 1;
            $nextToken = $tokens[$nextTokenPosition] ?? null;
            if (! $nextToken instanceof Token) {
                continue;
            }

            if (\str_contains($nextToken->getContent(), "\n")) {
                continue;
            }

            $lookaheadPosition = $tokens->getNonWhitespaceSibling($position, 1, " \t\r\0\x0B");
            if ($lookaheadPosition !== null && $tokens[$lookaheadPosition]->isGivenKind(T_COMMENT)) {
                continue;
            }

            if ($nextToken->getContent() === '{') {
                continue;
            }

            $tokens->ensureWhitespaceAtIndex($nextTokenPosition, 0, $this->whitespacesFixerConfig->getLineEnding());
        }
    }
}
