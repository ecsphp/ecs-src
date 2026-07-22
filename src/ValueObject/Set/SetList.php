<?php

declare(strict_types=1);

namespace Symplify\EasyCodingStandard\ValueObject\Set;

/**
 * @api
 */
final class SetList
{
    /**
     * @api
     */
    public const string PSR_12 = __DIR__ . '/../../../config/set/psr12.php';

    /**
     * @api
     */
    public const string CLEAN_CODE = __DIR__ . '/../../../config/set/clean-code.php';

    /**
     * @api
     * @deprecated rules moved to the "common" sets (array, docblock, spaces). Use SetList::COMMON instead.
     */
    public const string SYMPLIFY = __DIR__ . '/../../../config/set/symplify.php';

    /**
     * @api
     */
    public const string ARRAY = __DIR__ . '/../../../config/set/common/array.php';

    /**
     * @api
     */
    public const string COMMON = __DIR__ . '/../../../config/set/common.php';

    /**
     * @api
     */
    public const string COMMENTS = __DIR__ . '/../../../config/set/common/comments.php';

    /**
     * @api
     */
    public const string CONTROL_STRUCTURES = __DIR__ . '/../../../config/set/common/control-structures.php';

    /**
     * @api
     */
    public const string DOCBLOCK = __DIR__ . '/../../../config/set/common/docblock.php';

    /**
     * @api
     */
    public const string NAMESPACES = __DIR__ . '/../../../config/set/common/namespaces.php';

    /**
     * @api
     */
    public const string SPACES = __DIR__ . '/../../../config/set/common/spaces.php';

    /**
     * @api
     */
    public const string CASING = __DIR__ . '/../../../config/set/common/casing.php';

    /**
     * @api
     */
    public const string CLEANUP = __DIR__ . '/../../../config/set/common/cleanup.php';

    /**
     * @api
     */
    public const string DOCTRINE_ANNOTATIONS = __DIR__ . '/../../../config/set/doctrine-annotations.php';

    /**
     * @api
     */
    public const string LARAVEL = __DIR__ . '/../../../config/set/laravel.php';

    /**
     * @api
     */
    public const string STANDALONE_LINE = __DIR__ . '/../../../config/set/standalone-line.php';
}
