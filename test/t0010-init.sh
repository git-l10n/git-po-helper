#!/bin/sh

test_description="test git-po-helper init"

. ./lib/sharness.sh

HELPER="po-helper --no-gettext-back-compatible"

test_expect_success "setup" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	test -f workdir/po/git.pot &&
	test -f workdir/po/zh_CN.po
'

test_expect_success "fail to init: zh_CN.po already exist" '
	test_must_fail git -C workdir $HELPER init zh_CN >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-\EOF &&
	level=error msg="fail to init, \"po/zh_CN.po\" is already exist"

	ERROR: fail to execute "git-po-helper init"
	EOF

	test_cmp expect actual
'

test_expect_success "init zh_CN" '
	rm workdir/po/zh_CN.po &&
	git -C workdir $HELPER init zh_CN >actual &&

	cat >expect <<-\EOF &&

	========================================================================
	Notes for l10n team leader:

	    Since you created an initial locale file, you are likely to be the
	    leader of the zh_CN l10n team.

	    You should add your team infomation in the "po/TEAMS" file, and
	    make a commit for it.

	    Please read the file "po/README" first to understand the workflow
	    of Git l10n maintenance.
	========================================================================
	EOF
	test_cmp expect actual &&
	test -f workdir/po/zh_CN.po
'

test_expect_success "init with invalid locale" '
	test_must_fail git -C workdir $HELPER init xx >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-\EOF &&
	level=error msg="fail to init: invalid language code for locale \"xx\""

	ERROR: fail to execute "git-po-helper init"
	EOF

	test_cmp expect actual &&
	test ! -f workdir/po/xx.po
'

test_expect_success "init --core en_GB" '
	(
		cd workdir &&
		test ! -f po-core/core.pot &&
		test ! -f po-core/en_GB.po &&
		git $HELPER init --core en_GB &&
		test -f po-core/core.pot &&
		test -f po-core/en_GB.po
	) >actual &&

	cat >expect <<-\EOF &&

	========================================================================
	Notes for core po file:

	    To contribute a new l10n translation for Git, make a full
	    translation is not a piece of cake.  A small part of "po/git.pot"
	    is marked and saved in "po-core/core.pot".

	    The new generated po file for locale "en_GB" is stored in
	    "po-core/en_GB.po" which includes core l10n entries.

	    After translate this core po file, you can merge it to
	    "po/en_GB.po" using the following commands:

	        msgcat po-core/en_GB.po po/en_GB.po -s -o /tmp/en_GB.po
	        mv /tmp/en_GB.po po/en_GB.po
	        msgmerge --add-location --backup=off -U po/en_GB.po po/git.pot
	========================================================================
	EOF

	test_cmp expect actual
'

test_expect_success "init --core with invalid locale" '
	test_must_fail git -C workdir $HELPER init --core xx >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-\EOF &&
	level=error msg="fail to init: invalid language code for locale \"xx\""

	ERROR: fail to execute "git-po-helper init"
	EOF

	test_cmp expect actual &&
	test ! -f workdir/po/xx.po
'

test_done
