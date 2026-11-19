<?php

use Some\Exception;
use Some\Other\DifferentException;
use Foo\BarError;
function foo(FooException $e)
{
	try {
		throw new FooException();
	} catch (BarException $ex) {

	} catch (Throwable $ex) {

	} catch (TypeError $ex) {

	}
}
?><?php
function bar(Exception $e)
{
	try {
		throw new \Some\Other\Exception();
	} catch (DifferentException $ex) {

	} catch (Throwable $ex) {

	} catch (Exception $ex) {

	} catch (TypeError $ex) {

	} catch (BarError $ex) {

	}
}
