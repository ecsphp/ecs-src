package rules

import "testing"

// Cases ported from PHP-CS-Fixer's PhpdocAnnotationWithoutDotFixerTest. "\xae"
// is the invalid byte chr(174) the upstream test uses to check byte-mode fallback.
func TestPhpdocAnnotationWithoutDotUpstreamCases(t *testing.T) {
	t.Parallel()
	runUpstreamCases(t, PhpdocAnnotationWithoutDot{}, []upstreamCase{
		{"many params and sentences", nil,
			"<?php\n    /**\n     * Summary.\n     *\n     * Description.\n     *\n     * @param string|null $str   some string\n     * @param string $ip         IPv4 is not lowercased\n     * @param string $a          A\n     * @param string $a_string   a string\n     * @param string $ab         ab\n     * @param string $t34        T34\n     * @param string $s          S§\n     * @param string $genrb      Optional. The path to the \"genrb\" executable\n     * @param string $ellipsis1  Ellipsis is this: ...\n     * @param string $ellipsis2  Ellipsis is this: 。。。\n     * @param string $ellipsis3  Ellipsis is this: …\n     * @param bool   $isStr      Is it a string?\n     * @param int    $int        Some single-line description. With many dots.\n     * @param int    $int        Some multiline\n     *                           description. With many dots.\n     *\n     * @return array result array\n     *\n     * @SomeCustomAnnotation This is important sentence that must not be modified.\n     */",
			"<?php\n    /**\n     * Summary.\n     *\n     * Description.\n     *\n     * @param string|null $str   Some string.\n     * @param string $ip         IPv4 is not lowercased.\n     * @param string $a          A.\n     * @param string $a_string   A string.\n     * @param string $ab         Ab.\n     * @param string $t34        T34.\n     * @param string $s          S§.\n     * @param string $genrb      Optional. The path to the \"genrb\" executable\n     * @param string $ellipsis1  Ellipsis is this: ...\n     * @param string $ellipsis2  Ellipsis is this: 。。。\n     * @param string $ellipsis3  Ellipsis is this: …\n     * @param bool   $isStr      Is it a string?\n     * @param int    $int        Some single-line description. With many dots.\n     * @param int    $int        Some multiline\n     *                           description. With many dots.\n     *\n     * @return array Result array。\n     *\n     * @SomeCustomAnnotation This is important sentence that must not be modified.\n     */"},
		{"invalid utf8 byte", nil,
			"<?php\n    /**\n     * @var string this: \xae is an odd character\n     * @var string This: \xae is an odd character 2nd time。\n     */",
			"<?php\n    /**\n     * @var string This: \xae is an odd character.\n     * @var string This: \xae is an odd character 2nd time。\n     */"},
		{"deprecated with mid dot", nil,
			"<?php\n    /**\n     * @deprecated since version 2. Use emergency() which is PSR-3 compatible.\n     */", ""},
		{"internal continuation", nil,
			"<?php\n    /**\n     * @internal This method is public to be usable as callback. It should not\n     *           be used in user code.\n     */", ""},
		{"deprecated lowercased", nil,
			"<?php\n    /**\n     * @deprecated this is\n     *             deprecated\n     */",
			"<?php\n    /**\n     * @deprecated This is\n     *             deprecated.\n     */"},
		{"return multiline", nil,
			"<?php\n    /**\n     * @return bool|null returns `true` if the class has a single-column ID\n     *                   and Returns `false` otherwise\n     */",
			"<?php\n    /**\n     * @return bool|null Returns `true` if the class has a single-column ID\n     *                   and Returns `false` otherwise.\n     */"},
		{"throws trailing spaces", nil,
			"<?php\n    /**\n     * @throws \\Exception having whitespaces after dot, yet I am fixed\n     */",
			"<?php\n    /**\n     * @throws \\Exception having whitespaces after dot, yet I am fixed.   \n     */"},
		{"throws trailing tabs", nil,
			"<?php\n    /**\n     * @throws \\Exception having tabs after dot, yet I am fixed\n     */",
			"<?php\n    /**\n     * @throws \\Exception having tabs after dot, yet I am fixed.\t\t\n     */"},
		{"no change multiline params", nil,
			"<?php\n    /**\n     * Dispatches an event to all registered listeners.\n     *\n     * @param string    $eventName The name of the event to dispatch. The name of the event is\n     *                             the name of the method that is invoked on listeners.\n     * @param EventArgs $eventArgs The event arguments to pass to the event handlers/listeners.\n     *                             If not supplied, the single empty EventArgs instance is used.\n     *\n     * @return bool\n     */\n    function dispatchEvent($eventName, EventArgs $eventArgs = null) {}\n\n    /**\n     * Extract the `object_to_populate` field from the context if it exists\n     * and is an instance of the provided $class.\n     *\n     * @param string $class   The class the object should be\n     * @param array  $context The denormalization context\n     * @param string $key     Key in which to look for the object to populate.\n     *                        Keeps backwards compatibility with `AbstractNormalizer`.\n     *\n     * @return null|object an object if things check out, null otherwise\n     */\n    function extractObjectToPopulate($class, array $context, $key = null) {}\n                ", ""},
		{"broken phpdoc missing asterisk", nil,
			"<?php\n    /**\n     * This is a broken phpdoc - missing asterisk\n     * @param string $str As it is broken, let us not apply the rule to description of parameter.\n\n     */\n    function foo($str) {}", ""},
		{"multilined comment no change", nil,
			"<?php\n    /**\n     * @return bool|null Returns `true` if the class has a single-column ID.\n                         Returns `false` otherwise.\n                         That was multilined comment. With plenty of sentenced.\n     */\n    function nothingToDo() {}", ""},
		{"greek", nil,
			"<?php\n/**\n * @param string $bar τάχιστη\n */\nfunction foo ($bar) {}\n",
			"<?php\n/**\n * @param string $bar Τάχιστη.\n */\nfunction foo ($bar) {}\n"},
		{"url with var", nil,
			"<?php\n    /**\n     * @param Url $url URL is not lowercased\n     */",
			"<?php\n    /**\n     * @param Url $url URL is not lowercased.\n     */"},
		{"url without var", nil,
			"<?php\n    /**\n     * @param Url URL is not lowercased and note that name of param is ommited\n     */",
			"<?php\n    /**\n     * @param Url URL is not lowercased and note that name of param is ommited.\n     */"},
	})
}
