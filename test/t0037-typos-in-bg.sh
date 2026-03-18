#!/bin/sh

test_description="check typos in bg.po"

. ./lib/test-lib.sh

HELPER="po-helper --no-special-gettext-versions --report-typos=warn --report-file-locations=none"
POT_NO="--pot-file=no"

test_expect_success "checkout po-2.31.1" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	git -C workdir checkout po-2.31.1
'


test_expect_success "check typos in bg.po" '
	git -C workdir $HELPER check-po $POT_NO po/bg.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	cp "$TEST_DIRECTORY/t0037-typos-in-bg.expect" expect &&
	test_cmp expect actual
'


test_expect_success "still has typos in master branch" '
	git -C workdir checkout master &&
	test_must_fail git -C workdir $HELPER \
		check-po $POT_NO --report-typos=error po/bg.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	cat >expect <<-\EOF &&
	ℹ️ Syntax check with msgfmt
	 INFO [bg.po] 5195 translated messages.
	❌ msgid/msgstr pattern check
	 ERROR [bg.po] mismatched patterns: --dirstat=<...>,, --dirstat=files,param1,param2...
	 ERROR [bg.po] >> msgid: synonym for --dirstat=files,param1,param2...
	 ERROR [bg.po] >> msgstr: псевдоним на „--dirstat=ФАЙЛ…,ПАРАМЕТЪР_1,ПАРАМЕТЪР_2,…“
	 ERROR [bg.po]
	 ERROR [bg.po] mismatched patterns: --force
	 ERROR [bg.po] >> msgid: helper %s does not support '\''force'\''
	 ERROR [bg.po] >> msgstr: насрещната помощна програма „%s“ не поддържа опцията „--force“
	 ERROR [bg.po]
	 ERROR [bg.po] mismatched patterns: refs/heads, refs/heads/
	 ERROR [bg.po] >> msgid: HEAD (%s) points outside of refs/heads/
	 ERROR [bg.po] >> msgstr: „HEAD“ (%s) сочи извън директорията „refs/heads“
	 ERROR [bg.po]
	 ERROR [bg.po] mismatched patterns: _git_rev
	 ERROR [bg.po] >> msgid: git bundle create [<options>] <file> <git-rev-list args>
	 ERROR [bg.po] >> msgstr: git bundle create [ОПЦИЯ…] ФАЙЛ АРГУМЕНТ_ЗА_git_rev-list…
	 ERROR [bg.po]
	 ERROR [bg.po] mismatched patterns: --bare
	 ERROR [bg.po] >> msgid: create a mirror repository (implies bare)
	 ERROR [bg.po] >> msgstr: създаване на хранилище-огледало (включва опцията „--bare“ за голо хранилище)
	 ERROR [bg.po]
	 ERROR [bg.po] mismatched patterns: --mirror
	 ERROR [bg.po] >> msgid: unknown mirror argument: %s
	 ERROR [bg.po] >> msgstr: неправилна стойност за „--mirror“: %s
	 ERROR [bg.po]
	ERROR: check-po command failed
	EOF
	test_cmp expect actual
'

test_done
