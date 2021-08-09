#!/bin/sh

test_description="check typos in vi.po"

. ./lib/sharness.sh

test_expect_success "setup" '
	mkdir po &&
	touch po/git.pot &&
	cp ../examples/vi.po po
'

cat >expect <<-\EOF
[po/vi.po]    5204 translated messages.
level=warning msg="mismatch variable names: add_cacheinfo"
level=warning msg=">> msgid: add_cacheinfo failed for path '%s'; merge aborting."
level=warning msg=">> msgstr: addinfo_cache gặp lỗi đối với đường dẫn “%s”; việc hòa trộn bị bãi bỏ."
level=warning
level=warning msg="mismatch variable names: add_cacheinfo"
level=warning msg=">> msgid: add_cacheinfo failed to refresh for path '%s'; merge aborting."
level=warning msg=">> msgstr: addinfo_cache gặp lỗi khi làm mới đối với đường dẫn “%s”; việc hòa trộn bị bãi bỏ."
level=warning
level=warning msg="mismatch variable names: submodule.fetchjobs"
level=warning msg=">> msgid: negative values not allowed for submodule.fetchjobs"
level=warning msg=">> msgstr: không cho phép giá trị âm ở submodule.fetchJobs"
level=warning
EOF

test_expect_success "check typos in vi.po" '
	git-po-helper check-po vi >actual 2>&1 &&
	test_cmp expect actual
'

test_done
