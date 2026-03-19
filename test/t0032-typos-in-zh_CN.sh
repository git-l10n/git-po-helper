#!/bin/sh

test_description="check typos in zh_CN.po"

. ./lib/test-lib.sh

HELPER="po-helper --no-special-gettext-versions --report-typos=warn --report-file-locations=none"
POT_NO="--pot-file=no"

test_expect_success "checkout po-2.31.1" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	git -C workdir checkout po-2.31.1
'

cat >expect <<-\EOF
ℹ️ Syntax check with msgfmt
 INFO [zh_CN.po] 5104 translated messages.
⚠️ msgid/msgstr pattern check
 WARNING [zh_CN.po] mismatched patterns: crlf_action
 WARNING [zh_CN.po] >> msgid: illegal crlf_action %d
 WARNING [zh_CN.po] >> msgstr: 非法的 crlf 动作 %d
 WARNING [zh_CN.po]
 WARNING [zh_CN.po] mismatched patterns: dimmed_zebra
 WARNING [zh_CN.po] >> msgid: color moved setting must be one of 'no', 'default', 'blocks', 'zebra', 'dimmed-zebra', 'plain'
 WARNING [zh_CN.po] >> msgstr: 移动的颜色设置必须是 'no'、'default'、'blocks'、'zebra'、'dimmed_zebra' 或 'plain'
 WARNING [zh_CN.po]
 WARNING [zh_CN.po] mismatched patterns: lasy_name, lazy_name
 WARNING [zh_CN.po] >> msgid: unable to join lazy_name thread: %s
 WARNING [zh_CN.po] >> msgstr: 不能加入 lasy_name 线程：%s
 WARNING [zh_CN.po]
 WARNING [zh_CN.po] mismatched patterns: --signed
 WARNING [zh_CN.po] >> msgid: the receiving end does not support --signed push
 WARNING [zh_CN.po] >> msgstr: 接收端不支持签名推送
 WARNING [zh_CN.po]
 WARNING [zh_CN.po] mismatched patterns: --signed
 WARNING [zh_CN.po] >> msgid: not sending a push certificate since the receiving end does not support --signed push
 WARNING [zh_CN.po] >> msgstr: 未发送推送证书，因为接收端不支持签名推送
 WARNING [zh_CN.po]
 WARNING [zh_CN.po] mismatched patterns: --atomic
 WARNING [zh_CN.po] >> msgid: the receiving end does not support --atomic push
 WARNING [zh_CN.po] >> msgstr: 接收端不支持原子推送
 WARNING [zh_CN.po]
 WARNING [zh_CN.po] mismatched patterns: --porcelain
 WARNING [zh_CN.po] >> msgid: --progress can't be used with --incremental or porcelain formats
 WARNING [zh_CN.po] >> msgstr: --progress 不能和 --incremental 或 --porcelain 同时使用
 WARNING [zh_CN.po]
 WARNING [zh_CN.po] mismatched patterns: --type=bool
 WARNING [zh_CN.po] >> msgid: option `--default' expects a boolean value with `--type=bool`, not `%s`
 WARNING [zh_CN.po] >> msgstr: 选项 `--default' 和 `type=bool` 期望一个布尔值，不是 `%s`
 WARNING [zh_CN.po]
 WARNING [zh_CN.po] mismatched patterns: --type=ulong
 WARNING [zh_CN.po] >> msgid: option `--default' expects an unsigned long value with `--type=ulong`, not `%s`
 WARNING [zh_CN.po] >> msgstr: 选项 `--default' 和 `type=ulong` 期望一个无符号长整型，不是 `%s`
 WARNING [zh_CN.po]
 WARNING [zh_CN.po] mismatched patterns: extensions.partialClone, extensions.partialclone
 WARNING [zh_CN.po] >> msgid: --filter can only be used with the remote configured in extensions.partialclone
 WARNING [zh_CN.po] >> msgstr: 只可以将 --filter 用于在 extensions.partialClone 中配置的远程仓库
 WARNING [zh_CN.po]
 WARNING [zh_CN.po] mismatched patterns: FSCK_IGNORE
 WARNING [zh_CN.po] >> msgid: %d (FSCK_IGNORE?) should never trigger this callback
 WARNING [zh_CN.po] >> msgstr: %d（忽略 FSCK?）不应该触发这个调用
 WARNING [zh_CN.po]
 WARNING [zh_CN.po] mismatched patterns: git-am
 WARNING [zh_CN.po] >> msgid: It looks like 'git am' is in progress. Cannot rebase.
 WARNING [zh_CN.po] >> msgstr: 看起来 'git-am' 正在执行中。无法变基。
 WARNING [zh_CN.po]
 WARNING [zh_CN.po] mismatched patterns: refs/remotes/
 WARNING [zh_CN.po] >> msgid: Note: A branch outside the refs/remotes/ hierarchy was not removed;
 WARNING [zh_CN.po] to delete it, use:
 WARNING [zh_CN.po] >> msgstr: 注意：ref/remotes 层级之外的一个分支未被移除。要删除它，使用：
 WARNING [zh_CN.po]
 WARNING [zh_CN.po] mismatched patterns: refs/remotes/
 WARNING [zh_CN.po] >> msgid: Note: Some branches outside the refs/remotes/ hierarchy were not removed;
 WARNING [zh_CN.po] to delete them, use:
 WARNING [zh_CN.po] >> msgstr: 注意：ref/remotes 层级之外的一些分支未被移除。要删除它们，使用：
 WARNING [zh_CN.po]
 WARNING [zh_CN.po] mismatched patterns: submodule.alternateLocaion, submodule.alternateLocation
 WARNING [zh_CN.po] >> msgid: Value '%s' for submodule.alternateLocation is not recognized
 WARNING [zh_CN.po] >> msgstr: 不能识别 submodule.alternateLocaion 的取值 '%s'
 WARNING [zh_CN.po]
EOF

test_expect_success "check typos in zh_CN.po" '
	git -C workdir $HELPER check-po $POT_NO \
		--report-file-locations=none po/zh_CN.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

cat >expect <<-\EOF
ℹ️ Syntax check with msgfmt
 INFO [zh_CN.po] 5282 translated messages.
ℹ️ Location comments (#:)
 INFO [zh_CN.po] entry 1@L160 (msgid "Huh (%s)?"): location comment contains line number (use file-only or remove): "add-interactive.c:382"
ℹ️ PO filter (.gitattributes)
 INFO [zh_CN.po] No filter attribute set for XX.po. This will introduce location newlines into the
 INFO [zh_CN.po] repository and cause repository bloat.
 INFO [zh_CN.po]
 INFO [zh_CN.po] Please configure the filter attribute for XX.po, for example:
 INFO [zh_CN.po]
 INFO [zh_CN.po] .gitattributes: *.po filter=gettext-no-location
 INFO [zh_CN.po]
 INFO [zh_CN.po] See:
 INFO [zh_CN.po]
 INFO [zh_CN.po] https://lore.kernel.org/git/20220504124121.12683-1-worldhello.net@gmail.com/
⚠️ msgid/msgstr pattern check
 WARNING [zh_CN.po] mismatched patterns: refs/remotes/
 WARNING [zh_CN.po] >> msgid: Note: A branch outside the refs/remotes/ hierarchy was not removed;
 WARNING [zh_CN.po] to delete it, use:
 WARNING [zh_CN.po] >> msgstr: 注意：ref/remotes 层级之外的一个分支未被移除。要删除它，使用：
 WARNING [zh_CN.po]
 WARNING [zh_CN.po] mismatched patterns: refs/remotes/
 WARNING [zh_CN.po] >> msgid: Note: Some branches outside the refs/remotes/ hierarchy were not removed;
 WARNING [zh_CN.po] to delete them, use:
 WARNING [zh_CN.po] >> msgstr: 注意：ref/remotes 层级之外的一些分支未被移除。要删除它们，使用：
 WARNING [zh_CN.po]
EOF

test_expect_success "check typos in master branch" '
	git -C workdir checkout master &&
	git -C workdir $HELPER \
		check-po $POT_NO --report-typos=warn \
		--report-file-locations=warn po/zh_CN.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_done
