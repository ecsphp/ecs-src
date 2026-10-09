package rules

import "testing"

func TestRemoveUselessDefaultCommentKeepsHeredocBlankLine(t *testing.T) {
	t.Parallel()
	src := "<?php\n#[AsCommand(\n    name: 'mautic:broadcasts:send',\n    description: 'Process contacts pending to receive a channel broadcast.',\n    help: <<<'TXT'\n            The <info>%command.name%</info> command is send a channel broadcast to pending contacts.\n\n<info>php %command.full_name% --channel=email --id=3</info>\nTXT\n)]\nclass A {}\n"
	got, changed := apply(t, RemoveUselessDefaultComment{}, src)
	if changed || got != src {
		t.Fatalf("heredoc blank line must be preserved: changed=%v\ngot=%q", changed, got)
	}
}

func TestRemoveUselessDefaultCommentStillRemovesTodo(t *testing.T) {
	t.Parallel()
	src := "<?php\nclass A {\n    public function x()\n    {\n        // TODO: Implement x() method.\n    }\n}\n"
	got, changed := apply(t, RemoveUselessDefaultComment{}, src)
	if !changed {
		t.Fatalf("useless TODO comment should be removed, got=%q", got)
	}
}
