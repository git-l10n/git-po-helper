#!/bin/sh

test_description="test git-po-helper init"

. ./lib/sharness.sh

test_expect_success "setup" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	test -f workdir/po/git.pot
'

test_expect_success "fail to init: zh_CN.po already exist" '
	(
		cd workdir &&
		touch po/zh_CN.po &&
		test_must_fail git-po-helper init zh_CN >actual 2>&1 &&
		cat >expect <<-\EOF &&
		level=error msg="fail to init, \"po/zh_CN.po\" is already exist"

		ERROR: fail to execute "git-po-helper init"
		EOF
		test_cmp expect actual &&
		test -f po/zh_CN.po
	)
'

test_expect_success "init zh_CN" '
	(
		cd workdir &&
		rm po/zh_CN.po &&

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
		git-po-helper init zh_CN >actual &&
		test_cmp expect actual &&
		test -f po/zh_CN.po
	)
'

test_expect_success "init with invalid locale" '
	(
		cd workdir &&
		test_must_fail git-po-helper init xx >actual 2>&1 &&
		cat >expect <<-\EOF &&
		level=error msg="fail to init: invalid language code for locale \"xx\""

		ERROR: fail to execute "git-po-helper init"
		EOF
		test_cmp expect actual &&
		test ! -f po/xx.po
	)
'

test_expect_success "init --core en_GB" '
	(
		cd workdir &&
		test ! -f po-core/core.pot &&
		test ! -f po-core/en_GB.po &&
		git-po-helper init --core en_GB >actual &&
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
		test_cmp expect actual &&
		test -f po-core/core.pot &&
		test -f po-core/en_GB.po
	)
'

test_expect_success "init --core with invalid locale" '
	(
		cd workdir &&
		test_must_fail git-po-helper init --core xx >actual 2>&1 &&
		cat >expect <<-\EOF &&
		level=error msg="fail to init: invalid language code for locale \"xx\""

		ERROR: fail to execute "git-po-helper init"
		EOF
		test_cmp expect actual &&
		test ! -f po/xx.po
	)
'

test_done
