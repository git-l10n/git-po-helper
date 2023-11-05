#!/bin/sh

test_description="check typos in vi.po"

. ./lib/test-lib.sh

HELPER="po-helper --no-special-gettext-versions --pot-file=no --report-typos=warn --report-file-locations=none"

test_expect_success "checkout po-2.31.1" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	git -C workdir checkout po-2.31.1
'

cat >expect <<-\EOF
------------------------------------------------------------------------------
level=error msg="[po/vi.po]    5104 translated messages."
level=error msg="[po/vi.po]    too many obsolete entries (466) in comments, please remove them"
------------------------------------------------------------------------------
level=warning msg="[po/vi.po]    mismatch variable names: --quiet"
level=warning msg="[po/vi.po]    >> msgid: "
level=warning msg="[po/vi.po]    It took %.2f seconds to enumerate unstaged changes after reset. You can"
level=warning msg="[po/vi.po]    use '--quiet' to avoid this. Set the config setting reset.quiet to true"
level=warning msg="[po/vi.po]    to make this the default."
level=warning msg="[po/vi.po]"
level=warning msg="[po/vi.po]    >> msgstr: "
level=warning msg="[po/vi.po]    Cần %.2f giây để kiểm đếm các thay đổi chưa đưa lên bệ phóng sau khi đặt lại."
level=warning msg="[po/vi.po]    Bạn có thể sử dụng để tránh việc này. Đặt reset.quiet thành true trong"
level=warning msg="[po/vi.po]    cài đặt config nếu bạn muốn thực hiện nó như là mặc định."
level=warning msg="[po/vi.po]"
level=warning msg="[po/vi.po]"
level=warning msg="[po/vi.po]    mismatch variable names: $HOME"
level=warning msg="[po/vi.po]    >> msgid: $HOME not set"
level=warning msg="[po/vi.po]    >> msgstr: Chưa đặt biến môi trường HOME"
level=warning msg="[po/vi.po]"
level=warning msg="[po/vi.po]    mismatch variable names: --contents"
level=warning msg="[po/vi.po]    >> msgid: --contents and --reverse do not blend well."
level=warning msg="[po/vi.po]    >> msgstr: tùy chọn--contents và --reverse không được trộn vào nhau."
level=warning msg="[po/vi.po]"
level=warning msg="[po/vi.po]    mismatch variable names: --deepen"
level=warning msg="[po/vi.po]    >> msgid: --deepen and --depth are mutually exclusive"
level=warning msg="[po/vi.po]    >> msgstr: Các tùy chọn--deepen và --depth loại từ lẫn nhau"
level=warning msg="[po/vi.po]"
level=warning msg="[po/vi.po]    mismatch variable names: --long"
level=warning msg="[po/vi.po]    >> msgid: --long and -z are incompatible"
level=warning msg="[po/vi.po]    >> msgstr: hai tùy chọn -long và -z không tương thích với nhau"
level=warning msg="[po/vi.po]"
level=warning msg="[po/vi.po]    mismatch variable names: --stdout"
level=warning msg="[po/vi.po]    >> msgid: --stdout, --output, and --output-directory are mutually exclusive"
level=warning msg="[po/vi.po]    >> msgstr: Các tùy chọn--stdout, --output, và --output-directory loại từ lẫn nhau"
level=warning msg="[po/vi.po]"
level=warning msg="[po/vi.po]    mismatch variable names: git-am"
level=warning msg="[po/vi.po]    >> msgid: It looks like 'git am' is in progress. Cannot rebase."
level=warning msg="[po/vi.po]    >> msgstr: Hình như đang trong quá trình thực hiện lệnh “git-am”. Không thể rebase."
level=warning msg="[po/vi.po]"
level=warning msg="[po/vi.po]    mismatch variable names: add_cacheinfo, addinfo_cache"
level=warning msg="[po/vi.po]    >> msgid: add_cacheinfo failed for path '%s'; merge aborting."
level=warning msg="[po/vi.po]    >> msgstr: addinfo_cache gặp lỗi đối với đường dẫn “%s”; việc hòa trộn bị bãi bỏ."
level=warning msg="[po/vi.po]"
level=warning msg="[po/vi.po]    mismatch variable names: add_cacheinfo, addinfo_cache"
level=warning msg="[po/vi.po]    >> msgid: add_cacheinfo failed to refresh for path '%s'; merge aborting."
level=warning msg="[po/vi.po]    >> msgstr: addinfo_cache gặp lỗi khi làm mới đối với đường dẫn “%s”; việc hòa trộn bị bãi bỏ."
level=warning msg="[po/vi.po]"
level=warning msg="[po/vi.po]    mismatch variable names: dimmed_zebra"
level=warning msg="[po/vi.po]    >> msgid: color moved setting must be one of 'no', 'default', 'blocks', 'zebra', 'dimmed-zebra', 'plain'"
level=warning msg="[po/vi.po]    >> msgstr: cài đặt màu đã di chuyển phải là một trong “no”, “default”, “blocks”, “zebra”, “dimmed_zebra”, “plain”"
level=warning msg="[po/vi.po]"
level=warning msg="[po/vi.po]    mismatch variable names: --bisect-reset, --bisect-state"
level=warning msg="[po/vi.po]    >> msgid: git bisect--helper --bisect-state (good|old) [<rev>...]"
level=warning msg="[po/vi.po]    >> msgstr: git bisect--helper --bisect-reset (good|old) [<lần_chuyển_giao>…]"
level=warning msg="[po/vi.po]"
level=warning msg="[po/vi.po]    mismatch variable names: --url"
level=warning msg="[po/vi.po]    >> msgid: git submodule--helper clone [--prefix=<path>] [--quiet] [--reference <repository>] [--name <name>] [--depth <depth>] [--single-branch] --url <url> --path <path>"
level=warning msg="[po/vi.po]    >> msgstr: git submodule--helper clone [--prefix=</đường/dẫn>] [--quiet] [--reference <kho>] [--name <tên>] [--depth <sâu>] [--single-branch] [--url <url>] --path </đường/dẫn>"
level=warning msg="[po/vi.po]"
level=warning msg="[po/vi.po]    mismatch variable names: --decorate"
level=warning msg="[po/vi.po]    >> msgid: invalid --decorate option: %s"
level=warning msg="[po/vi.po]    >> msgstr: tùy chọn--decorate không hợp lệ: %s"
level=warning msg="[po/vi.po]"
level=warning msg="[po/vi.po]    mismatch variable names: submodule.fetchJobs, submodule.fetchjobs"
level=warning msg="[po/vi.po]    >> msgid: negative values not allowed for submodule.fetchjobs"
level=warning msg="[po/vi.po]    >> msgstr: không cho phép giá trị âm ở submodule.fetchJobs"
level=warning msg="[po/vi.po]"
level=warning msg="[po/vi.po]    mismatch variable names: git-upload-archive, git-upload-pack"
level=warning msg="[po/vi.po]    >> msgid: path to the remote git-upload-archive command"
level=warning msg="[po/vi.po]    >> msgstr: đường dẫn đến lệnh git-upload-pack trên máy chủ"
level=warning msg="[po/vi.po]"
level=warning msg="[po/vi.po]    mismatch variable names: %%(objectname), %%(objectsize)"
level=warning msg="[po/vi.po]    >> msgid: unrecognized %%(objectsize) argument: %s"
level=warning msg="[po/vi.po]    >> msgstr: tham số không được thừa nhận %%(objectname): %s"
level=warning msg="[po/vi.po]"
level=warning msg="[po/vi.po]    mismatch variable names: %%(color:%s), %%(màu:%s)"
level=warning msg="[po/vi.po]    >> msgid: unrecognized color: %%(color:%s)"
level=warning msg="[po/vi.po]    >> msgstr: không nhận ra màu: %%(màu:%s)"
level=warning msg="[po/vi.po]"

ERROR: fail to execute "git-po-helper check-po"
EOF

test_expect_success "check typos in vi.po" '
	test_must_fail git -C workdir $HELPER check-po vi >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_expect_success "no typos in master branch" '
	git -C workdir checkout master &&
	git -C workdir $HELPER \
		check-po --report-typos=error vi
'

test_done
