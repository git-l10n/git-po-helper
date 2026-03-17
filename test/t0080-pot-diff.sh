#!/bin/sh

test_description="show diff of git.pot"

. ./lib/test-lib.sh

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

	git -C workdir po-helper compare --stat -- po/git.pot >out 2>&1 &&
	grep -v "^#" <out |
		sed -e "s#from .* for git vN.N.N#from **** for git vN.N.N#" >actual &&

	# Compare uses msgctxt+msgid as key; if POT differs by msgctxt in some
	# entries between versions, "changed" can become "new" + "removed".
	cat >expect <<-\EOF &&
		397 new, 575 removed
	EOF
	test_cmp expect actual
'

test_expect_success "diff new version of po/git.pot" '
	(
		cd workdir &&
		git reset --hard HEAD~ &&
		git checkout remotes/origin/master -- po/git.pot
	) &&

	git -C workdir po-helper compare --stat -- po/git.pot >out 2>&1 &&
	grep -v "^#" <out |
		sed -e "s#from .* for git vN.N.N#from **** for git vN.N.N#" >actual &&

	# Compare uses msgctxt+msgid as key; if POT differs by msgctxt in some
	# entries between versions, "changed" can become "new" + "removed".
	cat >expect <<-\EOF &&
		575 new, 397 removed
	EOF
	test_cmp expect actual
'

test_done
