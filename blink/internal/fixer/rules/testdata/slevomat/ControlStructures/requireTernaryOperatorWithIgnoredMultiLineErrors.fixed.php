<?php

return true ? $a . $b : $a;

if (true) {
	return $a;
} else {
	return $a
		. $b;
}

$c = true ? $a . $b : $a;

if (true) {
	$c = $a;
} else {
	$c = $a
		. $b;
}
