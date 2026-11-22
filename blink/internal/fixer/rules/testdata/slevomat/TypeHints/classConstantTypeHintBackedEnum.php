<?php

enum Status: string
{
	const DEFAULT = 'active';
	public const OTHER = 1;

	case Active = 'active';

	public function other(): int
	{
		return self::OTHER;
	}
}

enum Level:int
{
	const LOW = 1;

	case Low = 1;
}
