#!/bin/sh

test_description="test git-po-helper check-commits"

. ./lib/test-lib.sh

HELPER="po-helper --no-special-gettext-versions"
POT_NO="--pot-file=no"

test_expect_success "setup" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	test -f workdir/po/git.pot
'

test_expect_success "new commit with changes outside of po/" '
	(
		cd workdir &&
		echo A >po/A.txt &&
		echo B >po/B.txt &&
		echo C >C.txt &&
		git add -A &&
		cat >.git/commit-message <<-\EOF &&
		l10n: test: commit with changes outside of po/

		Signed-off-by: Author <author@example.com>
		EOF
		test_tick &&
		git commit -F .git/commit-message
	) &&

	test_must_fail git -C workdir $HELPER \
		check-commits $POT_NO HEAD~..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-\EOF &&
	------------------------------------------------------------------------------
	level=error msg="commit <OID>: found changes beyond \"po/\" directory:"
	level=error msg="        C.txt"
	level=error
	------------------------------------------------------------------------------
	level=warning msg="commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different"
	level=info msg="checking commits: 0 passed, 1 failed."
	ERROR: check-commits command failed
	EOF

	test_cmp expect actual
'

test_expect_success "new commit with unsupported hidden meta fields" '
	(
		cd workdir &&
		echo AA >po/A.txt &&
		echo BB >po/B.txt &&
		git add -u &&
		cat >.git/commit-message <<-\EOF &&
		l10n: test: commit with hidden meta fields

		Signed-off-by: Author <author@example.com>
		EOF
		test_tick &&
		git commit -F .git/commit-message &&
		git cat-file commit HEAD >.git/commit-meta &&
		perl -pe "s#^(committer .*)#\1\nnote: i am a hacker#;
		          s#(^committer .*)#\1\nnote: happy coding#" \
		     <.git/commit-meta >.git/commit-hacked-meta &&

		cid=$(git hash-object -w -t commit .git/commit-hacked-meta) &&
		git update-ref refs/heads/master $cid
	) &&

	test_must_fail git -C workdir $HELPER \
		check-commits $POT_NO HEAD~..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-EOF &&
	level=error msg="commit <OID>: unknown commit header: note: happy coding"
	level=error msg="commit <OID>: unknown commit header: note: i am a hacker"
	------------------------------------------------------------------------------
	level=warning msg="commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different"
	level=info msg="checking commits: 0 passed, 1 failed."
	ERROR: check-commits command failed
	EOF

	test_cmp expect actual
'

test_expect_success "new commits with datetime in the future" '
	(
		cd workdir &&
		echo AAA >po/A.txt &&
		git add -u &&
		cat >.git/commit-message <<-\EOF &&
		l10n: test: 1 hour ahead in the future

		Signed-off-by: Author <author@example.com>
		EOF
		test_tick &&
		git commit -F .git/commit-message &&
		git cat-file commit HEAD >.git/commit-meta &&
		future=$(($(date -u +"%s")+3600)) &&
		sed -e "s/^author .*/author Jiang Xin <worldhello.net@gmail.com> $future +0000/" \
		    -e "s/^committer .*/committer Jiang Xin <worldhello.net@gmail.com> $future +0000/" \
			<.git/commit-meta >.git/commit-hacked-meta &&

		cid=$(git hash-object -w -t commit .git/commit-hacked-meta) &&
		git update-ref refs/heads/master $cid
	) &&
	(
		cd workdir &&
		echo BBB >po/B.txt &&
		git add -u &&
		cat >.git/commit-message <<-\EOF &&
		l10n: test: 5 minutes ahead in the future

		Signed-off-by: Author <author@example.com>
		EOF
		test_tick &&
		git commit -F .git/commit-message &&
		git cat-file commit HEAD >.git/commit-meta &&
		future=$(($(date -u +"%s")+300)) &&
		sed -e "s/^author .*/author Jiang Xin <worldhello.net@gmail.com> $future +0000/" \
		    -e "s/^committer .*/committer Jiang Xin <worldhello.net@gmail.com> $future +0000/" \
			<.git/commit-meta >.git/commit-hacked-meta &&

		cid=$(git hash-object -w -t commit .git/commit-hacked-meta) &&
		git update-ref refs/heads/master $cid
	)
'

test_expect_success "show errors of commit-date drift" '
	test_must_fail git -C workdir $HELPER \
		check-commits $POT_NO HEAD~2..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out |
		sed -e "s/in the future, .* from now/in the future, XX from now/g" >actual &&

	cat >expect <<-EOF &&
	------------------------------------------------------------------------------
	level=error msg="commit <OID>: bad author date: date is in the future, XX from now"
	level=error msg="commit <OID>: bad committer date: date is in the future, XX from now"
	------------------------------------------------------------------------------
	level=error msg="commit <OID>: bad author date: date is in the future, XX from now"
	level=error msg="commit <OID>: bad committer date: date is in the future, XX from now"
	level=info msg="checking commits: 0 passed, 2 failed."
	ERROR: check-commits command failed
	EOF

	test_cmp expect actual
'

test_expect_success "suppress errors of commit-date drift for github actions" '
	test_must_fail git -C workdir $HELPER --github-action-event=pull_request_target \
		check-commits $POT_NO HEAD~2..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out |
		sed -e "s/in the future, .* from now/in the future, XX from now/g" >actual &&

	cat >expect <<-EOF &&
	------------------------------------------------------------------------------
	ERROR commit <OID>: bad author date: date is in the future, XX from now
	ERROR commit <OID>: bad committer date: date is in the future, XX from now
	INFO checking commits: 1 passed, 1 failed.
	ERROR: check-commits command failed
	EOF

	test_cmp expect actual
'

test_expect_success "new commit with bad email address" '
	(
		cd workdir &&
		echo AAAA >po/A.txt &&
		echo BBBB >po/B.txt &&
		git add -u &&
		cat >.git/commit-message <<-\EOF &&
		l10n: test: commit with bad email address

		Signed-off-by: Author <author@example.com>
		EOF
		test_tick &&
		git commit -F .git/commit-message &&
		git cat-file commit HEAD >.git/commit-meta &&
		sed -e "s/^author .*/author Jiang Xin <worldhello.net AT gmail.com> 1112911993 +0800/" \
			-e "s/^committer .*/committer    <worldhello.net@gmail.com> 1112911993 +0800/" \
			<.git/commit-meta >.git/commit-hacked-meta &&

		cid=$(git hash-object -w -t commit .git/commit-hacked-meta) &&
		git update-ref refs/heads/master $cid
	) &&

	test_must_fail git -C workdir $HELPER \
		check-commits $POT_NO HEAD~..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-EOF &&
	------------------------------------------------------------------------------
	level=error msg="commit <OID>: bad format for author field: Jiang Xin <worldhello.net AT gmail.com> 1112911993 +0800"
	level=error msg="commit <OID>: bad format for committer field: <worldhello.net@gmail.com> 1112911993 +0800"
	level=info msg="checking commits: 0 passed, 1 failed."
	ERROR: check-commits command failed
	EOF
	
	test_cmp expect actual
'

test_expect_success "too many commits to check" '
	test_must_fail env MAX_COMMITS=1 git -C workdir $HELPER \
		check-commits $POT_NO >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-\EOF &&
	level=warning msg="too many commits to check (5 > 1), check args or use option --force"
	------------------------------------------------------------------------------
	level=error msg="commit <OID>: bad format for author field: Jiang Xin <worldhello.net AT gmail.com> 1112911993 +0800"
	level=error msg="commit <OID>: bad format for committer field: <worldhello.net@gmail.com> 1112911993 +0800"
	level=info msg="checking commits: 0 passed, 1 failed."
	ERROR: check-commits command failed
	EOF

	test_cmp expect actual
'

test_expect_success "too many commits to check" '
	test_must_fail env MAX_COMMITS=1 git -C workdir $HELPER \
		check-commits $POT_NO --force >out 2>&1 &&
	make_user_friendly_and_stable_output <out |
		sed -e "s/in the future, .* from now/in the future, XX from now/g" >actual &&

	cat >expect <<-\EOF &&
	------------------------------------------------------------------------------
	level=error msg="commit <OID>: bad format for author field: Jiang Xin <worldhello.net AT gmail.com> 1112911993 +0800"
	level=error msg="commit <OID>: bad format for committer field: <worldhello.net@gmail.com> 1112911993 +0800"
	level=info msg="checking commits: 0 passed, 1 failed."
	ERROR: check-commits command failed
	EOF

	test_cmp expect actual
'

test_expect_success "long subject, exceed hard limit" '
	(
		cd workdir &&
		cat >.git/commit-message <<-\EOF &&
		l10n: this subject has too many chracters, which is greater than threshold

		Signed-off-by: Author <author@example.com>
		EOF
		test_tick &&
		git commit --allow-empty -F .git/commit-message
	) &&

	test_must_fail git -C workdir $HELPER \
		check-commits $POT_NO HEAD~..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-EOF &&
	------------------------------------------------------------------------------
	level=warning msg="commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different"
	------------------------------------------------------------------------------
	level=warning msg="commit <OID>: subject length 74 > 72, about 98% commits have a subject less than 72 characters"
	------------------------------------------------------------------------------
	level=error msg="commit <OID>: subject (\"l10n: this ...\") is too long: 74 > 72"
	level=info msg="checking commits: 0 passed, 1 failed."
	ERROR: check-commits command failed
	EOF

	test_cmp expect actual
'

test_expect_success "long subject, exceed soft limit" '
	(
		cd workdir &&
		cat >.git/commit-message <<-\EOF &&
		l10n: the subject of a commit has length between 50 and 62

		Signed-off-by: Author <author@example.com>
		EOF
		test_tick &&
		git commit --allow-empty -F .git/commit-message
	) &&

	git -C workdir $HELPER \
		check-commits $POT_NO HEAD~..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-EOF &&
	------------------------------------------------------------------------------
	level=warning msg="commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different"
	------------------------------------------------------------------------------
	level=warning msg="commit <OID>: subject length 58 > 50, about 63% commits have a subject less than 50 characters"
	level=info msg="checking commits: 1 passed."
	EOF

	test_cmp expect actual
'

test_expect_success "no empty line between subject and body" '
	(
		cd workdir &&
		cat >.git/commit-message <<-\EOF &&
		l10n: test: no blank line between subject and body
		Signed-off-by: Author <author@example.com>
		EOF
		test_tick &&
		git commit --allow-empty -F .git/commit-message
	) &&

	test_must_fail git -C workdir $HELPER \
		check-commits $POT_NO HEAD~..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-EOF &&
	------------------------------------------------------------------------------
	level=warning msg="commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different"
	------------------------------------------------------------------------------
	level=error msg="commit <OID>: no blank line between subject and body of commit message"
	level=info msg="checking commits: 0 passed, 1 failed."
	ERROR: check-commits command failed
	EOF

	test_cmp expect actual
'

test_expect_success "no l10n prefix in subject" '
	(
		cd workdir &&
		cat >.git/commit-message <<-\EOF &&
		test: no l10n prefix in subject

		Signed-off-by: Author <author@example.com>
		EOF
		test_tick &&
		git commit --allow-empty -F .git/commit-message
	) &&

	test_must_fail git -C workdir $HELPER \
		check-commits $POT_NO HEAD~..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-EOF &&
	------------------------------------------------------------------------------
	level=warning msg="commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different"
	------------------------------------------------------------------------------
	level=error msg="commit <OID>: subject (\"test: no ...\") does not have prefix \"l10n:\""
	level=info msg="checking commits: 0 passed, 1 failed."
	ERROR: check-commits command failed
	EOF

	test_cmp expect actual
'

test_expect_success "non-ascii characters in subject" '
	(
		cd workdir &&
		cat >.git/commit-message <<-\EOF &&
		l10n: update translation for zh_CN (简体中文)

		Signed-off-by: Author <author@example.com>
		EOF
		test_tick &&
		git commit --allow-empty -F .git/commit-message
	) &&

	test_must_fail git -C workdir $HELPER \
		check-commits $POT_NO HEAD~..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-EOF &&
	------------------------------------------------------------------------------
	level=warning msg="commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different"
	------------------------------------------------------------------------------
	level=error msg="commit <OID>: subject has non-ascii character \"简\""
	level=info msg="checking commits: 0 passed, 1 failed."
	ERROR: check-commits command failed
	EOF

	test_cmp expect actual
'

test_expect_success "subject end with period" '
	(
		cd workdir &&
		cat >.git/commit-message <<-\EOF &&
		l10n: subject should not end with period.

		Signed-off-by: Author <author@example.com>
		EOF
		test_tick &&
		git commit --allow-empty -F .git/commit-message
	) &&

	test_must_fail git -C workdir $HELPER \
		check-commits $POT_NO HEAD~..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-EOF &&
	------------------------------------------------------------------------------
	level=warning msg="commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different"
	------------------------------------------------------------------------------
	level=error msg="commit <OID>: subject should not end with period"
	level=info msg="checking commits: 0 passed, 1 failed."
	ERROR: check-commits command failed
	EOF

	test_cmp expect actual
'

test_expect_success "empty commit log" '
	(
		cd workdir &&
		test_tick &&
		git commit --allow-empty -m "remove this line" &&
		git cat-file commit HEAD >.git/commit-meta &&
		sed -e "/^remove this line/ d" <.git/commit-meta \
			>.git/commit-hacked-meta &&

		cid=$(git hash-object -w -t commit .git/commit-hacked-meta) &&
		git update-ref refs/heads/master $cid
	) &&

	test_must_fail git -C workdir $HELPER \
		check-commits $POT_NO HEAD~..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-EOF &&
	------------------------------------------------------------------------------
	level=warning msg="commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different"
	------------------------------------------------------------------------------
	level=error msg="commit <OID>: do not have any commit message"
	level=info msg="checking commits: 0 passed, 1 failed."
	ERROR: check-commits command failed
	EOF

	test_cmp expect actual
'

test_expect_success "oneline commit message" '
	(
		cd workdir &&
		cat >.git/commit-message <<-\EOF &&
		l10n: one line commit message (test)
		EOF
		test_tick &&
		git commit --allow-empty -F .git/commit-message
	) &&

	test_must_fail git -C workdir $HELPER \
		check-commits $POT_NO HEAD~..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-EOF &&
	------------------------------------------------------------------------------
	level=warning msg="commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different"
	------------------------------------------------------------------------------
	level=error msg="commit <OID>: empty body of the commit message, no s-o-b signature"
	level=info msg="checking commits: 0 passed, 1 failed."
	ERROR: check-commits command failed
	EOF

	test_cmp expect actual
'

test_expect_success "no s-o-b signature (has body message, but no s-o-b)" '
	(
		cd workdir &&
		cat >.git/commit-message <<-\EOF &&
		l10n: test: no s-o-b signature

		This is body of commit log.
		more commit log message...
		EOF
		test_tick &&
		git commit --allow-empty -F .git/commit-message
	) &&

	test_must_fail git -C workdir $HELPER \
		check-commits $POT_NO HEAD~..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-EOF &&
	------------------------------------------------------------------------------
	level=warning msg="commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different"
	------------------------------------------------------------------------------
	level=error msg="commit <OID>: cannot find \"Signed-off-by:\" signature"
	level=info msg="checking commits: 0 passed, 1 failed."
	ERROR: check-commits command failed
	EOF

	test_cmp expect actual
'

test_expect_success "no s-o-b signature (has body message, no s-o-b, but has other signature)" '
	(
		cd workdir &&
		cat >.git/commit-message <<-\EOF &&
		l10n: test: no s-o-b signature

		This is body of commit log.
		more commit log message...

		Reported-by: reporter <reporter@example.com>
		EOF
		test_tick &&
		git commit --allow-empty -F .git/commit-message
	) &&

	test_must_fail git -C workdir $HELPER \
		check-commits $POT_NO HEAD~..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-EOF &&
	------------------------------------------------------------------------------
	level=warning msg="commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different"
	------------------------------------------------------------------------------
	level=error msg="commit <OID>: cannot find \"Signed-off-by:\" signature"
	level=info msg="checking commits: 0 passed, 1 failed."
	ERROR: check-commits command failed
	EOF

	test_cmp expect actual
'

test_expect_success "has s-o-b signature (only s-o-b, no body message)" '
	(
		cd workdir &&
		cat >.git/commit-message <<-\EOF &&
		l10n: test: has s-o-b signature

		Signed-off-by: author <author@example.com>
		EOF
		test_tick &&
		git commit --allow-empty -F .git/commit-message
	) &&

	git -C workdir $HELPER check-commits $POT_NO HEAD~..HEAD
'

test_expect_success "has s-o-b signature (only s-o-b and other signature, no body message)" '
	(
		cd workdir &&
		cat >.git/commit-message <<-\EOF &&
		l10n: test: has s-o-b signature

		Signed-off-by: author <author@example.com>
		Reported-by: Reporter <reporter@example.com>
		EOF
		test_tick &&
		git commit --allow-empty -F .git/commit-message
	) &&

	git -C workdir $HELPER check-commits $POT_NO HEAD~..HEAD
'

test_expect_success "has s-o-b signature (have s-o-b and other signature)" '
	(
		cd workdir &&
		cat >.git/commit-message <<-\EOF &&
		l10n: test: has s-o-b signature

		This is body of commit log.
		more commit log message...

		Signed-off-by: author <author@example.com>
		Reported-by: Reporter <reporter@example.com>
		EOF
		test_tick &&
		git commit --allow-empty -F .git/commit-message
	) &&

	git -C workdir $HELPER check-commits $POT_NO HEAD~..HEAD
'

test_expect_success "no s-o-b signature (tailing trash message)" '
	(
		cd workdir &&
		cat >.git/commit-message <<-\EOF &&
		l10n: test: has s-o-b signature

		This is body of commit log.
		more commit log message...

		Signed-off-by: author <author@example.com>
		Reported-by: Reporter <reporter@example.com>

		l10n: message should no be here
		EOF
		test_tick &&
		git commit --allow-empty -F .git/commit-message
	) &&

	test_must_fail git -C workdir $HELPER \
		check-commits $POT_NO HEAD~..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-EOF &&
	------------------------------------------------------------------------------
	level=warning msg="commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different"
	------------------------------------------------------------------------------
	level=error msg="commit <OID>: cannot find \"Signed-off-by:\" signature"
	level=info msg="checking commits: 0 passed, 1 failed."
	ERROR: check-commits command failed
	EOF

	test_cmp expect actual
'

test_expect_success "too long message in commit log body" '
	(
		cd workdir &&
		cat >.git/commit-message <<-\EOF &&
		l10n: test: too long commit log message body

		Start body of commit log. This is is a very long commit log message, which exceed 72
		characters.

		EOF
		test_tick &&
		git commit --allow-empty -F .git/commit-message
	) &&

	test_must_fail git -C workdir $HELPER \
		check-commits $POT_NO HEAD~..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-EOF &&
	------------------------------------------------------------------------------
	level=warning msg="commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different"
	------------------------------------------------------------------------------
	level=error msg="commit <OID>: cannot find \"Signed-off-by:\" signature"
	level=error msg="commit <OID>: line #3 (\"Start body ...\") is too long: 84 > 72"
	level=info msg="checking commits: 0 passed, 1 failed."
	ERROR: check-commits command failed
	EOF

	test_cmp expect actual
'

test_expect_success "merge commit with details" '
	(
		cd workdir &&
		git checkout -b topic/1 master &&
		cat >.git/commit-message <<-\EOF &&
		l10n: topic/1

		New commit for topic/1.

		Signed-off-by: Author <author@example.com>
		EOF
		test_tick &&
		git commit --allow-empty -F .git/commit-message &&

		git checkout master &&
		git -c merge.log=true merge --no-ff topic/1
	) &&

	git -C workdir $HELPER \
		check-commits $POT_NO HEAD~..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-EOF &&
	------------------------------------------------------------------------------
	level=warning msg="commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different"
	------------------------------------------------------------------------------
	level=warning msg="commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different"
	level=info msg="checking commits: 2 passed."
	EOF

	test_cmp expect actual
'

test_expect_success "merge commit subject not start with Merge and no details" '
	(
		cd workdir &&
		git checkout -b topic/2 master &&
		cat >.git/commit-message <<-\EOF &&
		l10n: topic/2

		New commit for topic/2.

		Signed-off-by: Author <author@example.com>
		EOF
		test_tick &&
		git commit --allow-empty -F .git/commit-message &&

		git checkout master &&
		git -c merge.log=false merge --no-ff -m "l10n: a merge commit" topic/2
	) &&

	test_must_fail git -C workdir $HELPER \
		check-commits $POT_NO HEAD~..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-EOF &&
	------------------------------------------------------------------------------
	level=warning msg="commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different"
	------------------------------------------------------------------------------
	level=error msg="commit <OID>: merge commit does not have prefix \"Merge\" in subject"
	------------------------------------------------------------------------------
	level=error msg="commit <OID>: empty body of the commit message, set merge.log=true"
	------------------------------------------------------------------------------
	level=warning msg="commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different"
	level=info msg="checking commits: 1 passed, 1 failed."
	ERROR: check-commits command failed
	EOF

	test_cmp expect actual
'

test_expect_success "utf-8 characters in commit log" '
	(
		cd workdir &&

		cat >.git/commit-message <<-\EOF &&
		l10n: test: utf-8 commit message

		使用 utf-8 编码的提交说明。

		Signed-off-by: Author <author@example.com>
		EOF
		test_tick &&
		git commit --allow-empty -F .git/commit-message &&
		git cat-file commit HEAD >.git/commit-meta
	) &&

	git -C workdir $HELPER \
		check-commits $POT_NO HEAD~..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-EOF &&
	------------------------------------------------------------------------------
	level=warning msg="commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different"
	level=info msg="checking commits: 1 passed."
	EOF

	test_cmp expect actual
'

test_expect_success "utf-8 characters in commit log with wrong encoding" '
	(
		cd workdir &&

		cat >.git/commit-message <<-\EOF &&
		l10n: test: utf-8 commit message

		使用 utf-8 编码的提交说明。

		Signed-off-by: Author <author@example.com>
		EOF
		test_tick &&
		git -c i18n.commitencoding=iso-8859-6 commit --allow-empty -F .git/commit-message &&
		git cat-file commit HEAD >.git/commit-meta
	) &&

	test_must_fail git -C workdir $HELPER \
		check-commits $POT_NO HEAD~..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-EOF &&
	------------------------------------------------------------------------------
	level=warning msg="commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different"
	------------------------------------------------------------------------------
	level=error msg="commit <OID>: bad iso-8859-6 characters in: \"使用 utf-8 编码的提交说明。\""
	level=error msg="    <iconv failure message>..."
	level=info msg="checking commits: 0 passed, 1 failed."
	ERROR: check-commits command failed
	EOF

	test_cmp expect actual
'

test_expect_success "gbk characters in commit log with proper encoding" '
	(
		cd workdir &&

		cat <<-\EOF |
		l10n: test: gbk commit message

		使用 gbk 编码的提交说明。

		Signed-off-by: Author <author@example.com>
		EOF
		iconv -f UTF-8 -t GBK >.git/commit-message &&
		test_tick &&
		git -c i18n.commitencoding=GBK commit --allow-empty -F .git/commit-message &&
		git cat-file commit HEAD >.git/commit-meta
	) &&

	git -C workdir $HELPER check-commits $POT_NO HEAD~..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-EOF &&
	------------------------------------------------------------------------------
	level=warning msg="commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different"
	level=info msg="checking commits: 1 passed."
	EOF

	test_cmp expect actual
'

test_expect_success "gbk characters in commit log with wrong encoding" '
	(
		cd workdir &&

		cat <<-\EOF |
		l10n: test: gbk commit message

		使用 gbk 编码的提交说明。

		Signed-off-by: Author <author@example.com>
		EOF
		iconv -f UTF-8 -t GBK >.git/commit-message &&
		test_tick &&
		git -c i18n.commitencoding=iso-8859-6 commit --allow-empty -F .git/commit-message &&
		git cat-file commit HEAD >.git/commit-meta
	) &&

	test_must_fail git -C workdir $HELPER \
		check-commits $POT_NO HEAD~..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-EOF &&
	------------------------------------------------------------------------------
	level=warning msg="commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different"
	------------------------------------------------------------------------------
	level=error msg="commit <OID>: bad iso-8859-6 characters in: \"ʹ\xd3\xc3 gbk \xb1\xe0\xc2\xeb\xb5\xc4\xccύ˵\xc3\xf7\xa1\xa3\""
	level=error msg="    <iconv failure message>..."
	level=info msg="checking commits: 0 passed, 1 failed."
	ERROR: check-commits command failed
	EOF

	test_cmp expect actual
'

test_expect_success "bad utf-8 characters in commit log" '
	(
		cd workdir &&

		git cat-file commit HEAD >.git/commit-meta &&
		sed -e "/^encoding /d" <.git/commit-meta \
			>.git/commit-hacked-meta &&

		cid=$(git hash-object -w -t commit .git/commit-hacked-meta) &&
		git update-ref refs/heads/master $cid
	) &&

	test_must_fail git -C workdir $HELPER \
		check-commits $POT_NO HEAD~..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-EOF &&
	------------------------------------------------------------------------------
	level=warning msg="commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different"
	------------------------------------------------------------------------------
	level=error msg="commit <OID>: bad UTF-8 characters in: \"ʹ\xd3\xc3 gbk \xb1\xe0\xc2\xeb\xb5\xc4\xccύ˵\xc3\xf7\xa1\xa3\""
	level=info msg="checking commits: 0 passed, 1 failed."
	ERROR: check-commits command failed
	EOF

	test_cmp expect actual
'

test_expect_success "bad commit range" '
	test_must_fail git -C workdir $HELPER \
		check-commits $POT_NO -qq non_exist_commit..HEAD >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-EOF &&
	level=error msg="fail to run git-rev-list: exit status 128"
	ERROR: check-commits command failed
	EOF

	test_cmp expect actual
'

test_done
