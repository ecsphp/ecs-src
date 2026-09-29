<?php

/**
 * @psalm-type ConfigType = array{
 *     mercure?: bool|array{
 *         enabled?: bool, // Default: false
 *         include_type?: bool, // Always include @type in updates (including delete ones). // Default: false
 *     },
 * }
 */
final class SomeConfig
{
}
