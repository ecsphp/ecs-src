<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\Console;

final class ExitCode
{
    public const int SUCCESS = 0;

    public const int FAILURE = 1;

    public const int CHANGED_CODE_OR_FOUND_ERRORS = 2;
}
