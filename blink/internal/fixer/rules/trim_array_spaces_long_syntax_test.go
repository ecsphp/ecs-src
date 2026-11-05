package rules

import "testing"

// The expected output of every case comes from the PHP engine.
func TestTrimArraySpacesLongSyntax(t *testing.T) {
	t.Parallel()
	cases := []struct{ src, want string }{
		{"$x = array( 1, 2 );", "$x = array(1, 2);"},
		{"$x = array( );", "$x = array();"},
		{"$x = array( 1, );", "$x = array(1, );"},
		{"$x = ARRAY( 1 );", "$x = ARRAY(1);"},
		{"$x = array ( 1 );", "$x = array (1);"},
		{"$x = array( // c\n    1\n);", "$x = array( // c\n    1\n);"},
		{"$x = array( /* c */ 1 /* d */ );", "$x = array(/* c */ 1 /* d */);"},
		{"$x = [ 1 ];", "$x = [1];"},
		{"foo( array( foo( 1 ) ) );", "foo( array(foo( 1 )) );"},
		{"$f = fn(array $a ) => array( $a );", "$f = fn(array $a ) => array($a);"},
		{"$x = array(\n    1 );", "$x = array(\n    1);"},
		{"$a[ 1 ];", "$a[ 1 ];"},
		{"$x = array(1, 2);", "$x = array(1, 2);"},
	}
	for _, c := range cases {
		got, changed := apply(t, TrimArraySpaces{}, "<?php\n"+c.src+"\n")
		if want := "<?php\n" + c.want + "\n"; got != want || changed != (c.src != c.want) {
			t.Errorf("%s\n got=%q changed=%v\nwant=%q", c.src, got, changed, want)
		}
	}
}
