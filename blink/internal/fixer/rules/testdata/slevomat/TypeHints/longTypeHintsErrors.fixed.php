<?php

/**
 * @param int $a
 * @return bool
 */
function doSomething($a)
{
	return true;
}

class Foo
{

	/** @var int|null */
	private $integer = 0;

	/**
	 * Boolean
	 * With
	 * Long
	 * Destription
	 *
	 * @var bool
	 */
	private $boolean = true;

	/**
	 * Array
	 * @var int[]|bool[]|null
	 */
	public $array;

	/**
	 * @param bool|null $a
	 * @param null|boolean
	 * @param int|bool $c
	 * @return integer|null
	 */
	public function doSomething($a, $b, $c)
	{
		return 0;
	}

	public function inlineDocComment($values)
	{
		/** @var bool|int|string $value */
		foreach ($values as $value) {

		}
	}

}

/**
 * @property bool $property
 * @property-read int $propertyRead
 * @property-write integer[] $propertyWrite
 * @method boolean method(integer $m, boolean ...$m2)
 * @method method2(?integer $m = null, ?boolean $m2, $m3)
 * @method boolean[]|array method3(?Boolean<integer> $m)
 */
class Boo
{

}

/**
 * @property int&bool $property
 */
class IntersectionAndGeneric
{

	/** @var (bool|null|int)[] */
	public $a;

	/** @var int&bool<boolean, integer> */
	public $b;

	/** @var string&(int|float) */
	public $c;

	/** @var string|(float&int) */
	public $d;

	/** @var bool[][][] */
	public $e;

	/** @var (int|bool)[][][] */
	public $f;

	/** @var int|(string<foo>&bool)[] */
	public $g;

	/** @var \Foo<\Boo<int, bool>> */
	public $h;

}

class CallableType
{

	/**
	 * @return callable(bool $bool): (int|float)
	 */
	public function returnsCallable()
	{

	}

}

/**
 * @return bool - true - if some cond
 *                   false - if some other cond
 */
function multilineDescription() {

}

/** @var array{int, bool} $arrayShape1 */
$arrayShape1 = [];

/** @var array{foo: int, bar: bool} $arrayShape2 */
$arrayShape2 = [];
