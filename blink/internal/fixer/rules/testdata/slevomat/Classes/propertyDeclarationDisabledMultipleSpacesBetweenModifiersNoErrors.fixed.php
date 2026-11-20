<?php // lint >= 8.1

class Whatever
{

	public   readonly int $one;

	public static string|bool $two = false;

	public function __construct(public   readonly Foo|Bar $three)
	{
	}

}
