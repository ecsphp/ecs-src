package rules

import "testing"

func echoLit(lit string) string { return "<?php echo " + lit + ";" }

// Cases ported from PHP-CS-Fixer's NumericLiteralSeparatorFixerTest.
func TestNumericLiteralSeparatorPhpCsFixerCases(t *testing.T) {
	t.Parallel()
	// default strategy use_separator, override_existing false
	pairs := [][2]string{
		{"1234", "1_234"},
		{"12345", "12_345"},
		{"123456", "123_456"},
		{"0b0101010001101000", "0b01010100_01101000"},
		{"0b01010100011010000110010101101111", "0b01010100_01101000_01100101_01101111"},
		{"0b110001000", "0b1_10001000"},
		{".001", ".001"},
		{".1001", ".100_1"},
		{"0.0001", "0.000_1"},
		{"0.001", "0.001"},
		{"1234.5", "1_234.5"},
		{"1.2345", "1.234_5"},
		{"1234e5", "1_234e5"},
		{"1234E5", "1_234E5"},
		{"1e2345", "1e2_345"},
		{"1234.5678e1234", "1_234.567_8e1_234"},
		{".5678e1234", ".567_8e1_234"},
		{"1.1e-1234", "1.1e-1_234"},
		{"1.1e-12345", "1.1e-12_345"},
		{"01234567e10", "01_234_567e10"},
		{"0123E5", "0_123E5"},
		{"01e-12345", "01e-12_345"},
		{"1.5E+3", "1.5E+3"},
		{".5e+5", ".5e+5"},
		{"1e+12345", "1e+12_345"},
		{"1234.5678e+1234", "1_234.567_8e+1_234"},
		{"0x42726F776E", "0x42_72_6F_77_6E"},
		{"0X42726F776E", "0X42_72_6F_77_6E"},
		{"0x2726F776E", "0x2_72_6F_77_6E"},
		{"0x1234567890abcdef", "0x12_34_56_78_90_ab_cd_ef"},
		{"0X1234567890ABCDEF", "0X12_34_56_78_90_AB_CD_EF"},
		{"0x1234e5", "0x12_34_e5"},
		{"012345", "012_345"},
		{"0123456", "0123_456"},
		{"01234567", "01_234_567"},
		{"10", "10"},
		{"0", "0"},
		{"0o12345", "0o12_345"},
		{"0o123456", "0o123_456"},
	}
	cases := make([]aliasUpstreamCase, 0, len(pairs)+4)
	for _, p := range pairs {
		cases = append(cases, aliasUpstreamCase{p[0], nil, echoLit(p[0]), echoLit(p[1])})
	}
	cases = append(cases,
		aliasUpstreamCase{"negative", nil, "<?php echo -1234;", "<?php echo -1_234;"},
		aliasUpstreamCase{"trailing dot", nil, "<?php $x = 100.;", "<?php $x = 100.;"},
		aliasUpstreamCase{"keep existing", nil, "<?php\necho 0B01010100_01101000;\necho 70_10_00;", "<?php\necho 0B01010100_01101000;\necho 70_10_00;"},
	)
	aliasRunUpstreamCases(t, NumericLiteralSeparator{}, cases)
}

func TestNumericLiteralSeparatorConfiguredCases(t *testing.T) {
	t.Parallel()
	override := map[string]any{"override_existing": true}
	aliasRunUpstreamCases(t, NumericLiteralSeparator{}, []aliasUpstreamCase{
		{"override regroup", override, "<?php\necho 123_4.5;\necho 70_10_00;", "<?php\necho 1_234.5;\necho 701_000;"},
	})
	noSep := map[string]any{"strategy": "no_separator"}
	aliasRunUpstreamCases(t, NumericLiteralSeparator{}, []aliasUpstreamCase{
		{"strip decimal", noSep, "<?php echo 1_234;", "<?php echo 1234;"},
		{"strip hex", noSep, "<?php echo 0x42_72_6F_77_6E;", "<?php echo 0x42726F776E;"},
		{"strip float", noSep, "<?php echo 1_234.567_8e1_234;", "<?php echo 1234.5678e1234;"},
	})
}
