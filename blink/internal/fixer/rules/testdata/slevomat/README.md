# slevomat fixtures

The `*.php` inputs are copied from the [slevomat/coding-standard](https://github.com/slevomat/coding-standard) test fixtures (`tests/Sniffs/*/data`, MIT, see `LICENSE.md`); `cases.json` lists each one with the sniff and the properties its upstream test runs it with.

The `*.fixed.php` files are what ECS produces, not upstream's: ECS ignores `phpcs:disable` comments. Regenerate them after adding a case:

```bash
SLEVOMAT_DIR=/path/to/slevomat-checkout php blink/bin/slevomat-golden.php
```
