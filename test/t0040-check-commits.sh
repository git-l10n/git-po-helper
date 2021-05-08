#!/bin/sh

test_description="test git-po-helper check-commits"

. ./lib/sharness.sh

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
		git commit -F .git/commit-message &&

		cat >expect <<-EOF &&
		level=error msg="commit <OID>: found changes beyond \"po/\" directory"
		level=error msg="    C.txt"
		level=warning msg="commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different"
		EOF
		test_must_fail git-po-helper check-commits HEAD~..HEAD >out 2>&1 &&
		make_user_friendly_and_stable_output <out >actual &&
		test_cmp expect actual
	)
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
		sed -e "/^parent /a note: i am a hacker" \
			-e "/^committer /a note: happy coding" <.git/commit-meta \
			>.git/commit-hacked-meta &&

		cid=$(git hash-object -w -t commit .git/commit-hacked-meta) &&
		git update-ref refs/heads/master $cid &&

		cat >expect <<-EOF &&
		level=error msg="commit <OID>: unknown commit header: note: i am a hacker"
		level=error msg="commit <OID>: unknown commit header: note: happy coding"
		level=warning msg="commit <OID>: author (A U Thor <author@example.com>) and committer (C O Mitter <committer@example.com>) are different"
		EOF
		test_must_fail git-po-helper check-commits HEAD~..HEAD >out 2>&1 &&
		make_user_friendly_and_stable_output <out >actual &&
		test_cmp expect actual
	)
'

test_expect_success "new commit with datetime in the future" '
	(
		cd workdir &&
		echo AAA >po/A.txt &&
		echo BBB >po/B.txt &&
		git add -u &&
		cat >.git/commit-message <<-\EOF &&
		l10n: test: commit with datetime in the future

		Signed-off-by: Author <author@example.com>
		EOF
		test_tick &&
		git commit -F .git/commit-message &&
		git cat-file commit HEAD >.git/commit-meta &&
		future=$(($(date -u +"%s")+100)) &&
		sed -e "s/^author .*/author Jiang Xin <worldhello.net@gmail.com> $future +0000/" \
			-e "s/^committer .*/committer Jiang Xin <worldhello.net@gmail.com> $future +0000/" \
			<.git/commit-meta >.git/commit-hacked-meta &&

		cid=$(git hash-object -w -t commit .git/commit-hacked-meta) &&
		git update-ref refs/heads/master $cid &&

		cat >expect <<-EOF &&
		level=error msg="commit <OID>: bad author date: date is in the future, XX seconds from now"
		level=error msg="commit <OID>: bad committer date: date is in the future, XX seconds from now"
		EOF
		test_must_fail git-po-helper check-commits HEAD~..HEAD >out 2>&1 &&
		make_user_friendly_and_stable_output <out |
			sed -e "s/[0-9]* seconds/XX seconds/g" >actual &&
		test_cmp expect actual
	)
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
			-e "s/^committer .*/committer <worldhello.net@gmail.com> 1112911993 +0800/" \
			<.git/commit-meta >.git/commit-hacked-meta &&

		cid=$(git hash-object -w -t commit .git/commit-hacked-meta) &&
		git update-ref refs/heads/master $cid &&

		cat >expect <<-EOF &&
		level=error msg="commit <OID>: bad format for author field: Jiang Xin <worldhello.net AT gmail.com> 1112911993 +0800"
		level=error msg="commit <OID>: bad format for committer field: <worldhello.net@gmail.com> 1112911993 +0800"
		EOF
		test_must_fail git-po-helper check-commits HEAD~..HEAD >out 2>&1 &&
		make_user_friendly_and_stable_output <out >actual &&
		test_cmp expect actual
	)
'

test_done
