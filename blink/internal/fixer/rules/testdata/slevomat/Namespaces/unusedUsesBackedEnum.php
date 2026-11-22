<?php

namespace Foo;

use Bar\Labels;
use Bar\Unused;

enum Status: string
{
	use Labels;

	case Active = 'active';
}
