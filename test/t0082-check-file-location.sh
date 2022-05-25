#!/bin/sh

test_description="check file-locations in po file"

. ./lib/sharness.sh

HELPER="po-helper --no-special-gettext-versions"

test_expect_success "setup" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	(
		cd workdir &&
		git checkout master &&
		test -f po/git.pot
	)
'

cat >expect <<-\EOF
level=info msg=---------------------------------------------------------------------------
level=info msg="[po/zh_CN.po]    5282 translated messages."
level=error msg=---------------------------------------------------------------------------
level=error msg="[po/zh_CN.po]    Found file-location comments in po file."
level=error msg="[po/zh_CN.po]    "
level=error msg="[po/zh_CN.po]    Please commit a location-less \"po/XX.po\" file to save repository size."
level=error msg="[po/zh_CN.po]    See: [Updating a \"XX.po\" file] section in \"po/README.md\" for reference."

ERROR: fail to execute "git-po-helper check-po"
EOF

test_expect_success "zh_CN.po: has file-locations" '
	test_must_fail git -C workdir $HELPER check-po \
		--check-file-locations po/zh_CN.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

cat >expect <<-\EOF
level=info msg=---------------------------------------------------------------------------
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
