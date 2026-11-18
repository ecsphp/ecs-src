<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Tests\Blink;

use Iterator;
use Nette\Utils\FileSystem;
use Nette\Utils\Json;
use PhpCsFixer\Fixer\ConfigurableFixerInterface;
use PhpCsFixer\Fixer\FixerInterface;
use PhpCsFixer\FixerFactory;
use PhpCsFixer\Tokenizer\Tokens;
use PHPUnit\Framework\Attributes\DataProvider;
use PHPUnit\Framework\TestCase;
use SplFileInfo;

/**
 * Blink copies these PHP-CS-Fixer bugs on purpose. When this fails, upstream changed:
 * update expected.php from the new output, and port the change to blink.
 */
final class UpstreamBugFixtureTest extends TestCase
{
    private const string CASES_DIRECTORY = __DIR__ . '/../../blink/internal/upstreambug/testdata/cases';

    #[DataProvider('provideData')]
    public function test(string $caseDirectory): void
    {
        $tokens = Tokens::fromCode(FileSystem::read($caseDirectory . '/input.php'));

        $checkers = Json::decode(FileSystem::read($caseDirectory . '/checkers.json'), forceArrays: true);
        foreach ($checkers['rules'] as $rule) {
            $fixer = $this->getFixer($rule['class']);
            if ($fixer instanceof ConfigurableFixerInterface) {
                $fixer->configure($rule['config']);
            }

            $fixer->fix(new SplFileInfo($caseDirectory . '/input.php'), $tokens);
            $tokens->clearEmptyTokens();
        }

        $this->assertSame(
            FileSystem::read($caseDirectory . '/expected.php'),
            $tokens->generateCode(),
            'PHP-CS-Fixer changed its output, see README.md in ' . basename($caseDirectory),
        );
    }

    private function getFixer(string $class): FixerInterface
    {
        $fixerFactory = new FixerFactory();
        $fixerFactory->registerBuiltInFixers();

        foreach ($fixerFactory->getFixers() as $fixer) {
            if ($fixer::class === $class) {
                return $fixer;
            }
        }

        $this->fail(sprintf('Fixer "%s" is not a built-in PHP-CS-Fixer fixer', $class));
    }

    /**
     * @return Iterator<string, array{string}>
     */
    public static function provideData(): Iterator
    {
        foreach ((array) glob(self::CASES_DIRECTORY . '/*', GLOB_ONLYDIR) as $caseDirectory) {
            yield basename((string) $caseDirectory) => [(string) $caseDirectory];
        }
    }
}
