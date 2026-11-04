# Easy Coding Standard - Development Repository

Source for [Symplify EasyCodingStandard (ECS)](https://github.com/easy-coding-standard/ecs). Code here is scoped, downgraded and deployed to [easy-coding-standard/ecs](https://github.com/easy-coding-standard/ecs), the package `composer require symplify/easy-coding-standard` installs.

- **Using ECS?** See the [user documentation](build/target-repository/README.md).
- **Contributing?** Read on.

Requires PHP `>=8.4`.

```bash
composer install

composer test         # PHPUnit tests
composer lint         # phpstan + rector dry-run + check-cs
composer lint.fix     # auto-fix rector + coding standard
```

Please make sure `composer lint` and `composer test` pass before sending a pull request.

Experimental [`--blink` mode](docs/blink.md) runs the reco Go binary instead of the PHP engine.
