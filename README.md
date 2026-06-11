# Easy Coding Standard – Development Repository

Development repository of [Symplify EasyCodingStandard (ECS)](https://github.com/easy-coding-standard/ecs). Code here is scoped, downgraded and deployed to the [easy-coding-standard/ecs](https://github.com/easy-coding-standard/ecs) repository that `composer require symplify/easy-coding-standard` installs.

User documentation lives in [`build/target-repository/README.md`](build/target-repository/README.md).

PHP `>=8.4` required.

```bash
composer install

composer test         # PHPUnit tests
composer lint         # phpstan + rector dry-run + check-cs
composer lint.fix     # auto-fix rector + coding standard
```

Please make sure `composer lint` and `composer test` pass before sending a pull request.
