<?php

use SomeOther\Exception as ConcreteException;

class Foo extends Exception
{

	public function __construct(
		\Throwable $e,
		\Throwable $generalException,
		Throwable $previous = null
	)
	{
		new Exception('foo');
		if ($e instanceof Exception) {

		}
		try {
			foo();
		} catch (\Exception $e) { // ok - catching \Throwable later

		} catch (Exception $e) { // ok - catching \Throwable later

		} catch (\Throwable $e) {

		}

		try {

		} catch (\Throwable $e) {

		} catch (\Throwable $e) {

		}
	}

}
