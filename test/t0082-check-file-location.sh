#!/bin/sh

test_description="check file-locations in po file"

. ./lib/sharness.sh

HELPER="po-helper --no-special-gettext-versions --check-pot-file=no"

test_expect_success "setup" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	(
		cd workdir &&
		git checkout master &&
		test -f po/git.pot
	)
'

cat >expect <<-\EOF
------------------------------------------------------------------------------
level=info msg="[po/zh_CN.po]    5282 translated messages."
------------------------------------------------------------------------------
level=error msg="[po/zh_CN.po]    Found file-location comments in po file. By submitting a location-less"
level=error msg="[po/zh_CN.po]    \"po/XX.po\" file, the size of the Git repository can be greatly reduced."
level=error msg="[po/zh_CN.po]    See the discussion below:"
level=error
level=error msg="[po/zh_CN.po]     https://lore.kernel.org/git/20220504124121.12683-1-worldhello.net@gmail.com/"
level=error
level=error msg="[po/zh_CN.po]    As how to commit a location-less \"po/XX.po\" file, See:"
level=error
level=error msg="[po/zh_CN.po]     the [Updating a \"XX.po\" file] section in \"po/README.md\""

ERROR: fail to execute "git-po-helper check-po"
EOF

test_expect_success "zh_CN.po: has file-locations" '
	test_must_fail git -C workdir $HELPER check-po \
		--check-file-locations po/zh_CN.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

cat >expect <<-\EOF
------------------------------------------------------------------------------
level=info msg="[po/zh_CN.po]    5282 translated messages."
EOF

test_expect_success "zh_CN.po: remove locations" '
	(
		cd workdir &&
		msgcat --add-location=file po/zh_CN.po -o po/zh_CN.poX &&
		mv po/zh_CN.poX po/zh_CN.po
	) &&
	git -C workdir $HELPER check-po \
		--check-file-locations po/zh_CN.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_expect_success "zh_CN.po: remove both files and locations" '
	(
		cd workdir &&
		msgcat --no-location po/zh_CN.po -o po/zh_CN.poX &&
		mv po/zh_CN.poX po/zh_CN.po
	) &&
	git -C workdir $HELPER check-po \
		--check-file-locations po/zh_CN.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_done
