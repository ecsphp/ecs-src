<?php

declare(strict_types=1);

namespace Symplify\CodingStandard\Fixer\Annotation;

use PhpCsFixer\FixerDefinition\FixerDefinition;
use PhpCsFixer\FixerDefinition\FixerDefinitionInterface;
use PhpCsFixer\Tokenizer\Token;
use PhpCsFixer\Tokenizer\Tokens;
use SplFileInfo;
use Symplify\CodingStandard\Fixer\AbstractSymplifyFixer;
use Symplify\CodingStandard\Fixer\Naming\MethodNameResolver;
use Symplify\CodingStandard\TokenRunner\Traverser\TokenReverser;

/**
 * @see \Symplify\CodingStandard\Tests\Fixer\Annotation\RemoveEventSubscriberDescriptionFixer\RemoveEventSubscriberDescriptionFixerTest
 */
final class RemoveEventSubscriberDescriptionFixer extends AbstractSymplifyFixer
{
    private const string ERROR_MESSAGE = 'Remove event subscriber docblock description that only repeats the method name words plus "event"';

    private readonly MethodNameResolver $methodNameResolver;

    public function __construct(
        private readonly TokenReverser $tokenReverser
    ) {
        $this->methodNameResolver = new MethodNameResolver();
    }

    public function getDefinition(): FixerDefinitionInterface
    {
        return new FixerDefinition(self::ERROR_MESSAGE, []);
    }

    /**
     * @param Tokens<Token> $tokens
     */
    public function isCandidate(Tokens $tokens): bool
    {
        if (! $tokens->isTokenKindFound(T_FUNCTION)) {
            return false;
        }

        return $tokens->isAnyTokenKindsFound([T_DOC_COMMENT, T_COMMENT]);
    }

    /**
     * @param Tokens<Token> $tokens
     */
    public function fix(SplFileInfo $fileInfo, Tokens $tokens): void
    {
        $reversedTokens = $this->tokenReverser->reverse($tokens);

        foreach ($reversedTokens as $index => $token) {
            if (! $token->isGivenKind([T_DOC_COMMENT, T_COMMENT])) {
                continue;
            }

            // keep docblocks that carry real annotations
            if (str_contains($token->getContent(), '@')) {
                continue;
            }

            $methodName = $this->methodNameResolver->resolve($tokens, $index);
            if ($methodName === null) {
                continue;
            }

            if (! $this->isEventDescription($token->getContent(), $methodName)) {
                continue;
            }

            $tokens->clearTokenAndMergeSurroundingWhitespace($index);
        }
    }

    private function isEventDescription(string $docContent, string $methodName): bool
    {
        $descriptionWords = $this->resolveWords($docContent);

        // must be an "... event" description, otherwise leave it to duplicate-description fixer
        if (! in_array('event', $descriptionWords, true)) {
            return false;
        }

        $descriptionWords = array_filter(
            $descriptionWords,
            static fn (string $word): bool => $word !== 'event'
        );

        if ($descriptionWords === []) {
            return false;
        }

        $methodWords = array_filter(
            $this->resolveWords($methodName),
            // event subscriber handlers are commonly prefixed with "on"
            static fn (string $word): bool => $word !== 'on'
        );

        sort($descriptionWords);
        sort($methodWords);

        return $descriptionWords === $methodWords;
    }

    /**
     * @return string[]
     */
    private function resolveWords(string $value): array
    {
        // split camelCase boundaries, e.g. "onLeadDelete" => "on Lead Delete"
        $spaced = (string) preg_replace('#(?<=[a-z])(?=[A-Z])#', ' ', $value);

        preg_match_all('#[a-zA-Z]+#', strtolower($spaced), $matches);

        return $matches[0];
    }
}
