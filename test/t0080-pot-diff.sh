#!/bin/sh

test_description="show diff of git.pot"

. ./lib/sharness.sh

HELPER="po-helper --no-special-gettext-versions"

test_expect_success "setup" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	test -f workdir/po/git.pot
'

test_expect_success "diff old version of po/git.pot" '
	(
		cd workdir &&
		git checkout HEAD~ -- po/git.pot
	) &&

	git -C workdir po-helper diff >out 2>&1 &&
	grep -v "^#" <out |
		sed -e "s#from .* for git vN.N.N#from **** for git vN.N.N#" >actual &&

	cat >expect <<-\EOF &&
	l10n: git.pot: vN.N.N round N (395 new, 573 removed)

	Generate po/git.pot from **** for git vN.N.N l10n round N.
	EOF
	test_cmp expect actual
'

test_expect_success "diff new version of po/git.pot" '
	(
		cd workdir &&
		git reset --hard HEAD~ &&
		git checkout remotes/origin/master -- po/git.pot
	) &&

	git -C workdir po-helper diff >out 2>&1 &&
	grep -v "^#" <out |
		sed -e "s#from .* for git vN.N.N#from **** for git vN.N.N#" >actual &&

	cat >expect <<-\EOF &&
	l10n: git.pot: vN.N.N round N (573 new, 395 removed)

	Generate po/git.pot from **** for git vN.N.N l10n round N.
	EOF
	test_cmp expect actual
'

test_done
