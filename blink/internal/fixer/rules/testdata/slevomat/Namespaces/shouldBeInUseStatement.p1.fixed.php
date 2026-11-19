<?php

namespace Foo\Test\Bla;
use Some\ConstantClass;
use Foo\SomeError;
use Some\CommonException;
use Exception;
use Nette\ObjectPrototype;
use const Boo\FOO;
use function Boo\foo;

class Bar
{

	public function bar()
	{
		new Lorem();
		$constant = ConstantClass::CONSTANT;
		new SomeError();
		new CommonException();
		new Exception();
		new ObjectPrototype();
		FOO;
		foo();
		min(1, 2);
		PHP_VERSION;
	}

}
