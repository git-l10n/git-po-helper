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
		check-po $POT_FILE po/zh_CN.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	cp "$TEST_DIRECTORY/t0060-bad-syntax.expect" expect &&
	test_cmp expect actual
'

test_expect_success "update zh_CN (with location)" '
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
	EOF

	git -C workdir $HELPER update $POT_FILE po/zh_CN.po 2>&1 |
		make_user_friendly_and_stable_output |
		sed "/^\.\./ d" >actual &&
	cp "$TEST_DIRECTORY/t0060-update.expect" expect &&
	test_cmp expect actual
'

test_expect_success "check update of zh_CN.po" '
	git -C workdir $HELPER \
		check-po $POT_FILE --report-file-locations=none po/zh_CN.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	cp "$TEST_DIRECTORY/t0060-check-update.expect" expect &&
	test_cmp expect actual
'

test_expect_success "check core update of zh_CN.po" '
	git -C workdir $HELPER \
		check-po $POT_FILE --core --report-file-locations=none po/zh_CN.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	cp "$TEST_DIRECTORY/t0060-check-core.expect" expect &&
	test_cmp expect actual
'

cat >expect <<-\EOF
⚠️ Author and committer
 WARNING commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different
❌ Commit subject
 ERROR commit <OID>: subject ("Add files ...") does not have prefix "l10n:"
❌ Commit message body
 ERROR commit <OID>: empty body of the commit message, no s-o-b signature
INFO: checking commits: 0 passed, 1 failed.
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
❌ Changes outside po/
 ERROR commit <OID>: found changes beyond "po/" directory:
 ERROR         A.txt
 ERROR
 ERROR commit <OID>: break because this commit is not for git-l10n
INFO: checking commits: 0 passed, 1 failed, 1 skipped.
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
⚠️ Changes outside po/
 WARNING commit <OID>: found changes beyond "po/" directory:
 WARNING         A.txt
 WARNING
 WARNING commit <OID>: break because this commit is not for git-l10n
INFO: checking commits: 0 passed, 0 failed, 2 skipped.
EOF

test_expect_success "check-commits --github-action-event=push" '
	git -C workdir po-helper \
		check-commits --pot-file=no \
		--github-action-event push \
		0000000000000000000000000000000000000000..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	test_cmp expect actual
'

test_done
