<?php

namespace Lorem\Ipsum;

use Bar;
use Foo\SomeError as FooError;

function bar()
{
	$bar = new Bar();
	try {
		throw new \Foo\SomeError();
	} catch (\Lorem\Ipsum\Foo\SomeException $e) {

	} catch (\Foo\SomeError $e) {

	} catch (\Lorem\Ipsum\SomeOtherError $e) {
		throw new \Lorem\Ipsum\SomeOtherError;
	}
}
