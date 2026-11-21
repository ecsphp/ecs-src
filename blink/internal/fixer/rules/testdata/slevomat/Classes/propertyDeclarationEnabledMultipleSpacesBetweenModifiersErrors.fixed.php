<?php // lint >= 8.1

class Whatever
{

	public readonly int $one;

	public static string|bool $two = false;

	public function __construct(public readonly ?Foo $three, public readonly Foo|Bar|null $four)
	{
	}

}

class Anything
{

	public function __construct(
		public readonly ?Foo $five,
		protected readonly Foo|Bar|null $six,
	)
	{
	}

}
