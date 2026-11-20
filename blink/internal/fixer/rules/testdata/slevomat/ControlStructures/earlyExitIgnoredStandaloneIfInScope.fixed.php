<?php

function () {
	if (false) {
		return;
	}

	doSomething();
};

function ($values) {
	foreach ($values as $value) {
		if (!$value) {
			continue;
		}

		doSomething();
	}
};
