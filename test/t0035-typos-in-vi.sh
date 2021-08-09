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
level=warning msg="mismatch variable names: --quiet"
level=warning msg=">> msgid: \nIt took %.2f seconds to enumerate unstaged changes after reset.  You can\nuse '--quiet' to avoid this.  Set the config setting reset.quiet to true\nto make this the default.\n"
level=warning msg=">> msgstr: \nCần %.2f giây để kiểm đếm các thay đổi chưa đưa lên bệ phóng sau khi đặt lại.\nBạn có thể sử dụng để tránh việc này. Đặt reset.quiet thành true trong\ncài đặt config nếu bạn muốn thực hiện nó như là mặc định.\n"
level=warning
level=warning msg="mismatch variable names: $HOME"
level=warning msg=">> msgid: $HOME not set"
level=warning msg=">> msgstr: Chưa đặt biến môi trường HOME"
level=warning
level=warning msg="mismatch variable names: --long"
level=warning msg=">> msgid: --long and -z are incompatible"
level=warning msg=">> msgstr: hai tùy chọn -long và -z không tương thích với nhau"
level=warning
level=warning msg="mismatch variable names: git-am"
level=warning msg=">> msgid: It looks like 'git am' is in progress. Cannot rebase."
level=warning msg=">> msgstr: Hình như đang trong quá trình thực hiện lệnh “git-am”. Không thể rebase."
level=warning
level=warning msg="mismatch variable names: add_cacheinfo, addinfo_cache"
level=warning msg=">> msgid: add_cacheinfo failed for path '%s'; merge aborting."
level=warning msg=">> msgstr: addinfo_cache gặp lỗi đối với đường dẫn “%s”; việc hòa trộn bị bãi bỏ."
level=warning
level=warning msg="mismatch variable names: add_cacheinfo, addinfo_cache"
level=warning msg=">> msgid: add_cacheinfo failed to refresh for path '%s'; merge aborting."
level=warning msg=">> msgstr: addinfo_cache gặp lỗi khi làm mới đối với đường dẫn “%s”; việc hòa trộn bị bãi bỏ."
level=warning
level=warning msg="mismatch variable names: dimmed_zebra"
level=warning msg=">> msgid: color moved setting must be one of 'no', 'default', 'blocks', 'zebra', 'dimmed-zebra', 'plain'"
level=warning msg=">> msgstr: cài đặt màu đã di chuyển phải là một trong “no”, “default”, “blocks”, “zebra”, “dimmed_zebra”, “plain”"
level=warning
level=warning msg="mismatch variable names: --bisect-reset, --bisect-state"
level=warning msg=">> msgid: git bisect--helper --bisect-state (good|old) [<rev>...]"
level=warning msg=">> msgstr: git bisect--helper --bisect-reset (good|old) [<lần_chuyển_giao>…]"
level=warning
level=warning msg="mismatch variable names: --url"
level=warning msg=">> msgid: git submodule--helper clone [--prefix=<path>] [--quiet] [--reference <repository>] [--name <name>] [--depth <depth>] [--single-branch] --url <url> --path <path>"
level=warning msg=">> msgstr: git submodule--helper clone [--prefix=</đường/dẫn>] [--quiet] [--reference <kho>] [--name <tên>] [--depth <sâu>] [--single-branch] [--url <url>] --path </đường/dẫn>"
level=warning
level=warning msg="mismatch variable names: submodule.fetchJobs, submodule.fetchjobs"
level=warning msg=">> msgid: negative values not allowed for submodule.fetchjobs"
level=warning msg=">> msgstr: không cho phép giá trị âm ở submodule.fetchJobs"
level=warning
level=warning msg="mismatch variable names: git-upload-archive, git-upload-pack"
level=warning msg=">> msgid: path to the remote git-upload-archive command"
level=warning msg=">> msgstr: đường dẫn đến lệnh git-upload-pack trên máy chủ"
level=warning
EOF

test_expect_failure "check typos in vi.po" '
	git-po-helper check-po vi >actual 2>&1 &&
	test_cmp expect actual
'

test_done
