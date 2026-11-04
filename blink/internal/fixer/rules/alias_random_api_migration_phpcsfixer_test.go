package rules

import "testing"

func randomReplacements(pairs map[string]string) map[string]any {
	m := make(map[string]any, len(pairs))
	for k, v := range pairs {
		m[k] = v
	}
	return map[string]any{"replacements": m}
}

// Cases ported from PHP-CS-Fixer's RandomApiMigrationFixerTest. The default
// config is the v3 mapping (rand->mt_rand, srand->mt_srand, getrandmax->mt_getrandmax).
func TestRandomApiMigrationPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	aliasRunUpstreamCases(t, RandomApiMigration{}, []aliasUpstreamCase{
		{"method call", nil, "<?php $smth->srand($a);", "<?php $smth->srand($a);"},
		{"longer name", nil, "<?php srandSmth($a);", "<?php srandSmth($a);"},
		{"suffix name", nil, "<?php smth_srand($a);", "<?php smth_srand($a);"},
		{"new", nil, "<?php new srand($a);", "<?php new srand($a);"},
		{"new namespaced", nil, "<?php new Smth\\srand($a);", "<?php new Smth\\srand($a);"},
		{"namespaced", nil, "<?php Smth\\srand($a);", "<?php Smth\\srand($a);"},
		{"static call", nil, "<?php Smth::srand($a);", "<?php Smth::srand($a);"},
		{"name before ::", nil, "<?php srand::smth($a);", "<?php srand::smth($a);"},
		{"name before backslash", nil, "<?php srand\\smth($a);", "<?php srand\\smth($a);"},
		{"srand", nil, "<?php srand($a);", "<?php mt_srand($a);"},
		{"srand leading backslash", nil, "<?php \\srand($a);", "<?php \\mt_srand($a);"},
		{"reference", nil, "<?php $a = &srand($a);", "<?php $a = &mt_srand($a);"},
		{"reference namespaced", nil, "<?php $a = &\\srand($a);", "<?php $a = &\\mt_srand($a);"},
		{"comments", nil, "<?php /* foo */ srand /** bar */ ($a);", "<?php /* foo */ mt_srand /** bar */ ($a);"},
		{"getrandmax zero args", nil, "<?php a(getrandmax ());", "<?php a(mt_getrandmax ());"},
		{"rand zero args", nil, "<?php a(rand());", "<?php a(mt_rand());"},
		{"srand zero args", nil, "<?php a(srand());", "<?php a(mt_srand());"},
		{"srand zero args backslash", nil, "<?php a(\\srand());", "<?php a(\\mt_srand());"},
		{"srand trailing comma", nil, "<?php srand($a,);", "<?php mt_srand($a,);"},
		{"first class callable", nil, "<?php $f = srand(...);", "<?php $f = srand(...);"},
	})
}

func TestRandomApiMigrationRandomIntCases(t *testing.T) {
	t.Parallel()
	randInt := randomReplacements(map[string]string{"rand": "random_int"})
	aliasRunUpstreamCases(t, RandomApiMigration{}, []aliasUpstreamCase{
		{"rand to random_int zero args", randInt, "<?php rand();", "<?php random_int(0, getrandmax());"},
		{"nested", randInt, "<?php rand($d, rand($a,$b));", "<?php random_int($d, random_int($a,$b));"},
		{"keep mt_rand namespaced", randInt, "<?php rand($a, \\Other\\Scope\\mt_rand($a));", "<?php random_int($a, \\Other\\Scope\\mt_rand($a));"},
	})
	bothInt := randomReplacements(map[string]string{"rand": "random_int", "mt_rand": "random_int"})
	aliasRunUpstreamCases(t, RandomApiMigration{}, []aliasUpstreamCase{
		{"rand and mt_rand", bothInt, "<?php $a = rand(1,2) + mt_rand(3,4);", "<?php $a = random_int(1,2) + random_int(3,4);"},
		{"rand and mt_rand trailing comma", bothInt, "<?php $a = rand(1,2,) + mt_rand(3,4,);", "<?php $a = random_int(1,2,) + random_int(3,4,);"},
	})
	v4 := randomReplacements(map[string]string{
		"mt_getrandmax": "getrandmax", "mt_rand": "random_int", "mt_srand": "srand", "rand": "random_int",
	})
	aliasRunUpstreamCases(t, RandomApiMigration{}, []aliasUpstreamCase{
		{"v4 mapping", v4,
			"<?php mt_srand(123); return mt_rand(0, mt_getrandmax()) + rand(456, 789);",
			"<?php srand(123); return random_int(0, getrandmax()) + random_int(456, 789);"},
	})
}
