#!/bin/sh

test_description="check typos in vi.po"

. ./lib/test-lib.sh

HELPER="po-helper --no-special-gettext-versions --report-typos=warn --report-file-locations=none"
POT_NO="--pot-file=no"

test_expect_success "checkout po-2.31.1" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	git -C workdir checkout po-2.31.1
'

cat >expect <<-\EOF
ℹ️ Syntax check with msgfmt
 INFO [vi.po] 5104 translated messages.
❌ Obsolete #~ entries
 ERROR [vi.po] you have 466 obsolete entries, please remove them
⚠️ msgid/msgstr pattern check
 WARNING [vi.po] mismatched patterns: --quiet
 WARNING [vi.po] >> msgid:
 WARNING [vi.po] It took %.2f seconds to enumerate unstaged changes after reset. You can
 WARNING [vi.po] use '--quiet' to avoid this. Set the config setting reset.quiet to true
 WARNING [vi.po] to make this the default.
 WARNING [vi.po]
 WARNING [vi.po] >> msgstr:
 WARNING [vi.po] Cần %.2f giây để kiểm đếm các thay đổi chưa đưa lên bệ phóng sau khi đặt lại.
 WARNING [vi.po] Bạn có thể sử dụng để tránh việc này. Đặt reset.quiet thành true trong
 WARNING [vi.po] cài đặt config nếu bạn muốn thực hiện nó như là mặc định.
 WARNING [vi.po]
 WARNING [vi.po]
 WARNING [vi.po] mismatched patterns: $HOME
 WARNING [vi.po] >> msgid: $HOME not set
 WARNING [vi.po] >> msgstr: Chưa đặt biến môi trường HOME
 WARNING [vi.po]
 WARNING [vi.po] mismatched patterns: --contents
 WARNING [vi.po] >> msgid: --contents and --reverse do not blend well.
 WARNING [vi.po] >> msgstr: tùy chọn--contents và --reverse không được trộn vào nhau.
 WARNING [vi.po]
 WARNING [vi.po] mismatched patterns: --deepen
 WARNING [vi.po] >> msgid: --deepen and --depth are mutually exclusive
 WARNING [vi.po] >> msgstr: Các tùy chọn--deepen và --depth loại từ lẫn nhau
 WARNING [vi.po]
 WARNING [vi.po] mismatched patterns: --long
 WARNING [vi.po] >> msgid: --long and -z are incompatible
 WARNING [vi.po] >> msgstr: hai tùy chọn -long và -z không tương thích với nhau
 WARNING [vi.po]
 WARNING [vi.po] mismatched patterns: --stdout
 WARNING [vi.po] >> msgid: --stdout, --output, and --output-directory are mutually exclusive
 WARNING [vi.po] >> msgstr: Các tùy chọn--stdout, --output, và --output-directory loại từ lẫn nhau
 WARNING [vi.po]
 WARNING [vi.po] mismatched patterns: git-am
 WARNING [vi.po] >> msgid: It looks like 'git am' is in progress. Cannot rebase.
 WARNING [vi.po] >> msgstr: Hình như đang trong quá trình thực hiện lệnh “git-am”. Không thể rebase.
 WARNING [vi.po]
 WARNING [vi.po] mismatched patterns: add_cacheinfo, addinfo_cache
 WARNING [vi.po] >> msgid: add_cacheinfo failed for path '%s'; merge aborting.
 WARNING [vi.po] >> msgstr: addinfo_cache gặp lỗi đối với đường dẫn “%s”; việc hòa trộn bị bãi bỏ.
 WARNING [vi.po]
 WARNING [vi.po] mismatched patterns: add_cacheinfo, addinfo_cache
 WARNING [vi.po] >> msgid: add_cacheinfo failed to refresh for path '%s'; merge aborting.
 WARNING [vi.po] >> msgstr: addinfo_cache gặp lỗi khi làm mới đối với đường dẫn “%s”; việc hòa trộn bị bãi bỏ.
 WARNING [vi.po]
 WARNING [vi.po] mismatched patterns: dimmed_zebra
 WARNING [vi.po] >> msgid: color moved setting must be one of 'no', 'default', 'blocks', 'zebra', 'dimmed-zebra', 'plain'
 WARNING [vi.po] >> msgstr: cài đặt màu đã di chuyển phải là một trong “no”, “default”, “blocks”, “zebra”, “dimmed_zebra”, “plain”
 WARNING [vi.po]
 WARNING [vi.po] mismatched patterns: --bisect-reset, --bisect-state
 WARNING [vi.po] >> msgid: git bisect--helper --bisect-state (good|old) [<rev>...]
 WARNING [vi.po] >> msgstr: git bisect--helper --bisect-reset (good|old) [<lần_chuyển_giao>…]
 WARNING [vi.po]
 WARNING [vi.po] mismatched patterns: --url
 WARNING [vi.po] >> msgid: git submodule--helper clone [--prefix=<path>] [--quiet] [--reference <repository>] [--name <name>] [--depth <depth>] [--single-branch] --url <url> --path <path>
 WARNING [vi.po] >> msgstr: git submodule--helper clone [--prefix=</đường/dẫn>] [--quiet] [--reference <kho>] [--name <tên>] [--depth <sâu>] [--single-branch] [--url <url>] --path </đường/dẫn>
 WARNING [vi.po]
 WARNING [vi.po] mismatched patterns: --decorate
 WARNING [vi.po] >> msgid: invalid --decorate option: %s
 WARNING [vi.po] >> msgstr: tùy chọn--decorate không hợp lệ: %s
 WARNING [vi.po]
 WARNING [vi.po] mismatched patterns: submodule.fetchJobs, submodule.fetchjobs
 WARNING [vi.po] >> msgid: negative values not allowed for submodule.fetchjobs
 WARNING [vi.po] >> msgstr: không cho phép giá trị âm ở submodule.fetchJobs
 WARNING [vi.po]
 WARNING [vi.po] mismatched patterns: git-upload-archive, git-upload-pack
 WARNING [vi.po] >> msgid: path to the remote git-upload-archive command
 WARNING [vi.po] >> msgstr: đường dẫn đến lệnh git-upload-pack trên máy chủ
 WARNING [vi.po]
 WARNING [vi.po] mismatched patterns: %%(objectname), %%(objectsize)
 WARNING [vi.po] >> msgid: unrecognized %%(objectsize) argument: %s
 WARNING [vi.po] >> msgstr: tham số không được thừa nhận %%(objectname): %s
 WARNING [vi.po]
 WARNING [vi.po] mismatched patterns: %%(color:%s), %%(màu:%s)
 WARNING [vi.po] >> msgid: unrecognized color: %%(color:%s)
 WARNING [vi.po] >> msgstr: không nhận ra màu: %%(màu:%s)
 WARNING [vi.po]
ERROR: check-po command failed
EOF

test_expect_success "check typos in vi.po" '
	test_must_fail git -C workdir $HELPER check-po $POT_NO \
		po/vi.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_expect_success "no typos in master branch" '
	git -C workdir checkout master &&
	test_must_fail git -C workdir $HELPER \
		check-po $POT_NO --report-typos=error \
		po/vi.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	cat >expect <<-\EOF &&
	ℹ️ Syntax check with msgfmt
	 INFO [vi.po] 5282 translated messages.
	❌ Obsolete #~ entries
	 ERROR [vi.po] you have 73 obsolete entries, please remove them
	ERROR: check-po command failed
	EOF
	test_cmp expect actual
'

test_done
