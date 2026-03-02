#!/bin/sh

test_description="check output for --github-action-event"

. ./lib/test-lib.sh

HELPER="po-helper --github-action-event=pull_request_target"
POT_FILE="--pot-file=po/git.pot"
POT_NO="--pot-file=no"

test_expect_success "setup" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	(
		cd workdir &&
		git switch po-2.31.1 &&
		test -f po/git.pot
	)
'

cat >expect <<-\EOF
------------------------------------------------------------------------------
ERROR [po/zh_CN.po]    po/zh_CN.po:25: end-of-line within string
ERROR [po/zh_CN.po]    msgfmt: found 1 fatal error
ERROR [po/zh_CN.po]    fail to check po: exit status 1
------------------------------------------------------------------------------
ERROR [po/zh_CN.po]    Found file-location comments in po file. By submitting a location-less
ERROR [po/zh_CN.po]    "po/XX.po" file, the size of the Git repository can be greatly reduced.
ERROR [po/zh_CN.po]    See the discussion below:
ERROR [po/zh_CN.po]
ERROR [po/zh_CN.po]     https://lore.kernel.org/git/20220504124121.12683-1-worldhello.net@gmail.com/
ERROR [po/zh_CN.po]
ERROR [po/zh_CN.po]    As how to commit a location-less "po/XX.po" file, See:
ERROR [po/zh_CN.po]
ERROR [po/zh_CN.po]     the [Updating a "XX.po" file] section in "po/README.md"
------------------------------------------------------------------------------
ERROR [po/zh_CN.po]    fail to compile po/zh_CN.po: exit status 1
ERROR [po/zh_CN.po]    fail to generate mofile
ERROR: check-po command failed
EOF

test_expect_success "bad syntax of zh_CN.po" '
	cat >workdir/po/zh_CN.po <<-\EOF &&
	msgid ""
	msgstr ""
	"Project-Id-Version: Git\n"
	"Report-Msgid-Bugs-To: Git Mailing List <git@vger.kernel.org>\n"
	"POT-Creation-Date: 2021-03-04 22:41+0800\n"
	"PO-Revision-Date: 2021-03-04 22:41+0800\n"
	"Last-Translator: Automatically generated\n"
	"Language-Team: none\n"
	"Language: zh_CN\n"
	"MIME-Version: 1.0\n"
	"Content-Type: text/plain; charset=UTF-8\n"
	"Content-Transfer-Encoding: 8bit\n"
	"Plural-Forms: nplurals=2; plural=(n != 1);\n"

	#: remote.c:399
	msgid "more than one receivepack given, using the first"
	msgstr "提供了一个以上的 receivepack，使用第一个"

	#: remote.c:407
	msgid "more than one uploadpack given, using the first"
	msgstr "提供了一个以上的 uploadpack，使用第一个"

	msgid "po-helper test: not a real l10n message: xyz"
	msgstr "po-helper 测试：不是一个真正的本地化字符串: xyz""
	EOF

	test_must_fail git -C workdir $HELPER \
		check-po $POT_FILE zh_CN >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	test_cmp expect actual
'

test_expect_success "update zh_CN (--add-location=file)" '
	cat >expect <<-EOF &&
	INFO run msgmerge for "Chinese - China": msgmerge --add-location=file -o - po/zh_CN.po po/git.pot
	------------------------------------------------------------------------------
	INFO [po/zh_CN.po]    2 translated messages, 5102 untranslated messages.
	EOF

	cat >workdir/po/zh_CN.po <<-\EOF &&
	msgid ""
	msgstr ""
	"Project-Id-Version: Git\n"
	"Report-Msgid-Bugs-To: Git Mailing List <git@vger.kernel.org>\n"
	"POT-Creation-Date: 2021-03-04 22:41+0800\n"
	"PO-Revision-Date: 2021-03-04 22:41+0800\n"
	"Last-Translator: Automatically generated\n"
	"Language-Team: none\n"
	"Language: zh_CN\n"
	"MIME-Version: 1.0\n"
	"Content-Type: text/plain; charset=UTF-8\n"
	"Content-Transfer-Encoding: 8bit\n"
	"Plural-Forms: nplurals=2; plural=(n != 1);\n"

	#: remote.c:399
	msgid "more than one receivepack given, using the first"
	msgstr "提供了一个以上的 receivepack，使用第一个"

	#: remote.c:407
	msgid "more than one uploadpack given, using the first"
	msgstr "提供了一个以上的 uploadpack，使用第一个"

	msgid "po-helper test: not a real l10n message: xyz"
	msgstr "po-helper 测试：不是一个真正的本地化字符串: xyz"
	EOF

	git -C workdir $HELPER update $POT_FILE zh_CN 2>&1 |
		make_user_friendly_and_stable_output |
		sed "/^\.\./ d" >actual &&
	test_cmp expect actual
'

test_expect_success "check update of zh_CN.po" '
	cat >expect <<-\EOF &&
	------------------------------------------------------------------------------
	INFO [po/zh_CN.po]    2 translated messages, 5102 untranslated messages.
	EOF

	git -C workdir $HELPER \
		check-po $POT_FILE zh_CN >out 2>&1 &&
	make_user_friendly_and_stable_output <out |
		head -2 >actual &&
	test_cmp expect actual
'

cat >expect <<-\EOF
------------------------------------------------------------------------------
INFO [po/zh_CN.po]    2 translated messages, 5102 untranslated messages.
------------------------------------------------------------------------------
INFO [zh_CN.po (core)]    2 translated messages, 479 untranslated messages.
------------------------------------------------------------------------------
WARNING [po/zh_CN.po]    5102 untranslated string(s) in your 'po/XX.po'
WARNING [po/zh_CN.po]
WARNING [po/zh_CN.po]     > po/XX.po:18: this message is untranslated
WARNING [po/zh_CN.po]     > po/XX.po:22: this message is untranslated
WARNING [po/zh_CN.po]     > po/XX.po:26: this message is untranslated
WARNING [po/zh_CN.po]     > ...
WARNING [po/zh_CN.po]
EOF

test_expect_success "check core update of zh_CN.po" '
	git -C workdir $HELPER \
		check-po $POT_FILE --core zh_CN >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

cat >expect <<-\EOF
------------------------------------------------------------------------------
WARNING commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different
------------------------------------------------------------------------------
ERROR commit <OID>: subject ("Add files ...") does not have prefix "l10n:"
------------------------------------------------------------------------------
ERROR commit <OID>: empty body of the commit message, no s-o-b signature
INFO checking commits: 0 passed, 1 failed.
ERROR: check-commits command failed
EOF

test_expect_success "check-commits (old-oid is zero)" '
	test_must_fail git -C workdir $HELPER \
		check-commits $POT_NO 0000000000000000000000000000000000000000..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_expect_success "create new non-l10n commit" '
	(
		cd workdir &&
		echo A >A.txt &&
		git add A.txt &&
		test_tick &&
		git commit -m "A"
	)
'

cat >expect <<-\EOF
------------------------------------------------------------------------------
ERROR commit <OID>: found changes beyond "po/" directory:
ERROR         A.txt
ERROR
ERROR commit <OID>: break because this commit is not for git-l10n
INFO checking commits: 0 passed, 1 failed, 1 skipped.
ERROR: check-commits command failed
EOF

test_expect_success "check-commits (non-l10n commit)" '
	test_must_fail git -C workdir $HELPER \
		check-commits $POT_NO 0000000000000000000000000000000000000000..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_expect_success "check-commits --github-action-event=pull_request" '
	test_must_fail git -C workdir po-helper \
		check-commits --pot-file=no \
		--github-action-event=pull_request \
		0000000000000000000000000000000000000000..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_expect_success "check-commits --github-action-event=pull_request_target" '
	test_must_fail git -C workdir po-helper \
		check-commits --pot-file=no \
		--github-action-event=pull_request_target \
		0000000000000000000000000000000000000000..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

cat >expect <<-\EOF
------------------------------------------------------------------------------
WARNING commit <OID>: found changes beyond "po/" directory:
WARNING         A.txt
WARNING
WARNING commit <OID>: break because this commit is not for git-l10n
INFO checking commits: 0 passed, 0 failed, 2 skipped.
EOF

test_expect_success "check-commits --github-action-event=push" '
	git -C workdir po-helper \
		check-commits --pot-file=no \
		--github-action-event push \
		0000000000000000000000000000000000000000..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_expect_success "check-commits --github-action-event=push" '
	git -C workdir po-helper \
		check-commits --pot-file=no \
		--github-action-event push \
		0000000000000000000000000000000000000000..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_done
