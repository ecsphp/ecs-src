<?php

declare(strict_types=1);

namespace Symplify\CodingStandard\Fixer\Commenting;

use PhpCsFixer\FixerDefinition\FixerDefinition;
use PhpCsFixer\FixerDefinition\FixerDefinitionInterface;
use PhpCsFixer\Tokenizer\Token;
use PhpCsFixer\Tokenizer\Tokens;
use SplFileInfo;
use Symplify\CodingStandard\Fixer\AbstractSymplifyFixer;
use Symplify\CodingStandard\TokenRunner\Traverser\TokenReverser;
use Symplify\CodingStandard\Utils\Regex;

/**
 * Turns a "type" doc tag into the expected "var" tag, also upgrading a single-asterisk inline comment to a doc block.
 *
 * @see \Symplify\CodingStandard\Tests\Fixer\Commenting\TypeToVarTagFixer\TypeToVarTagFixerTest
 */
final class TypeToVarTagFixer extends AbstractSymplifyFixer
{
    /**
     * @see https://regex101.com/r/8tFqJp/1
     */
    private const string TYPE_TAG_REGEX = '#@type\b#';

    /**
     * @see https://regex101.com/r/cj95e6/1
     */
    private const string SINGLE_ASTERISK_START_REGEX = '#^/\*(\n?\s+@var)#';

    private const string ERROR_MESSAGE = 'Use "@var" doc tag instead of "@type"';

    public function __construct(
        private readonly TokenReverser $tokenReverser
    ) {
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

            $docContent = $token->getContent();
            if (! Regex::match($docContent, self::TYPE_TAG_REGEX)) {
                continue;
            }

            $newDocContent = Regex::replace($docContent, self::TYPE_TAG_REGEX, '@var');
            $newDocContent = Regex::replace($newDocContent, self::SINGLE_ASTERISK_START_REGEX, '/**$1');

            $tokens[$index] = new Token([T_DOC_COMMENT, $newDocContent]);
        }
    }
}
