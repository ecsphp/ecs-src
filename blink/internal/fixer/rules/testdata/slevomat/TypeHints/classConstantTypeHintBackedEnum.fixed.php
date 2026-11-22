<?php

enum Status: string
{
	const string DEFAULT = 'active';
	public const int OTHER = 1;

	case Active = 'active';

	public function other(): int
	{
		return self::OTHER;
	}
}

enum Level:int
{
	const int LOW = 1;

	case Low = 1;
}
