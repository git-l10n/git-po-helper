#!/bin/sh

test_description="check typos in zh_CN.po"

. ./lib/sharness.sh

test_expect_success "setup" '
	mkdir po &&
	touch po/git.pot &&
	cp ../examples/zh_CN.po po
'

cat >expect <<-\EOF
[po/zh_CN.po] 5204 translated messages.
level=warning msg="mismatch variable names: FSCK_IGNORE"
level=warning msg=">> msgid: %d (FSCK_IGNORE?) should never trigger this callback"
level=warning msg=">> msgstr: %d（忽略 FSCK?）不应该触发这个调用"
level=warning
level=warning msg="mismatch variable names: extensions.partialclone"
level=warning msg=">> msgid: --filter can only be used with the remote configured in extensions.partialclone"
level=warning msg=">> msgstr: 只可以将 --filter 用于在 extensions.partialClone 中配置的远程仓库"
level=warning
level=warning msg="mismatch variable names: submodule.alternateLocation"
level=warning msg=">> msgid: Value '%s' for submodule.alternateLocation is not recognized"
level=warning msg=">> msgstr: 不能识别 submodule.alternateLocaion 的取值 '%s'"
level=warning
level=warning msg="mismatch variable names: crlf_action"
level=warning msg=">> msgid: illegal crlf_action %d"
level=warning msg=">> msgstr: 非法的 crlf 动作 %d"
level=warning
level=warning msg="mismatch variable names: lazy_name"
level=warning msg=">> msgid: unable to join lazy_name thread: %s"
level=warning msg=">> msgstr: 不能加入 lasy_name 线程：%s"
level=warning
EOF

test_expect_success "check typos in zh_CN.po" '
	git-po-helper check-po zh_CN >actual 2>&1 &&
	test_cmp expect actual
'

test_done
