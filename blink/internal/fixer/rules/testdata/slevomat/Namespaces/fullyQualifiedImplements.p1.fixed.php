<?php

use Some\OtherClass;
class Foo implements SomeClass, OtherClass
{
	private $version = PHP_VERSION;
}

class Ipsum implements /*\Omega, */ Dolor
{

}

class Lorem implements Dolor, Amet
{

}

class LoremConsecteur implements Dolor, Amet
{

}

class Bar implements Dolor, Amet, Omega
{

}
