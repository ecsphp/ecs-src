<?php

namespace Foo;

use Bar\Labels;

enum Status: string
{
	use Labels;

	case Active = 'active';
}
