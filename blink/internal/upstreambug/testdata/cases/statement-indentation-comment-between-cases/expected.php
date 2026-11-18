<?php

switch ($type) {
    case 'a':
        // handles a and b
    case 'b':
        doSomething();
}

switch ($a) {
    case 1:
        return 'one';

        // Two and three share a branch
    case 2:
    case 3:
        // Three too
    case 4:
        return 'more';
}
