#!/bin/sh

test_description="check typos in es.po"

. ./lib/test-lib.sh

HELPER="po-helper --no-special-gettext-versions --report-typos=warn --report-file-locations=none"
POT_NO="--pot-file=no"

test_expect_success "checkout po-2.31.1" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	git -C workdir checkout po-2.31.1
'

test_expect_success "check typos in es.po of git 2.31.1" '
	test_must_fail git -C workdir $HELPER check-po $POT_NO \
		po/es.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	cp "$TEST_DIRECTORY/t0034-typos-in-es.expect" expect &&
	test_cmp expect actual
'

cat >expect <<-\EOF
ℹ️ Syntax check with msgfmt
 INFO [es.po] 5210 translated messages.
❌ Obsolete #~ entries
 ERROR [es.po] you have 125 obsolete entries, please remove them
❌ msgid/msgstr pattern check
 ERROR [es.po] mismatched patterns: refs/preferch/, refs/prefetch/
 ERROR [es.po] >> msgid: modify the refspec to place all refs within refs/prefetch/
 ERROR [es.po] >> msgstr: modificar el refspec para colocar todas las referencias en refs/preferch/
 ERROR [es.po]
 ERROR [es.po] mismatched patterns: refs/remotes/<...>/HEAD, refs/remotos/<...>/HEAD
 ERROR [es.po] >> msgid: delete refs/remotes/<name>/HEAD
 ERROR [es.po] >> msgstr: borrar refs/remotos/<nombre>/HEAD
 ERROR [es.po]
ERROR: check-po command failed
EOF

test_expect_success "typos in master branch" '
	git -C workdir checkout master &&
	test_must_fail git -C workdir $HELPER \
		check-po $POT_NO --report-typos=error \
		po/es.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_done
