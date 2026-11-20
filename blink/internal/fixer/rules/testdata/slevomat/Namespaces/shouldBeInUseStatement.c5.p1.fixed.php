<?php

namespace Foo\Test\Bla;
use Some\ConstantClass;
use Nette\ObjectPrototype;
use const Boo\FOO;
use function Boo\foo;

class Bar
{

	public function bar()
	{
		new Lorem();
		$constant = ConstantClass::CONSTANT;
		new \Foo\SomeError();
		new \Some\CommonException();
		new \Exception();
		new ObjectPrototype();
		FOO;
		foo();
		min(1, 2);
		PHP_VERSION;
	}

}
