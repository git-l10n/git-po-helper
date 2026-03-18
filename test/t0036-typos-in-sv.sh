#!/bin/sh

test_description="check typos in sv.po"

. ./lib/test-lib.sh

HELPER="po-helper --no-special-gettext-versions --report-typos=warn --report-file-locations=none"
POT_NO="--pot-file=no"

test_expect_success "checkout po-2.31.1" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	git -C workdir checkout po-2.31.1
'

test_expect_success "check typos in sv.po" '
	test_must_fail git -C workdir $HELPER check-po $POT_NO \
		po/sv.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	cp "$TEST_DIRECTORY/t0036-typos-in-sv.expect" expect &&
	test_cmp expect actual
'

test_expect_success "typos in master branch" '
	git -C workdir checkout master &&
	test_must_fail git -C workdir $HELPER \
		check-po $POT_NO --report-typos=error \
		po/sv.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	cat >expect <<-\EOF &&
	ℹ️ Syntax check with msgfmt
	 INFO [sv.po] 5282 translated messages.
	❌ Obsolete #~ entries
	 ERROR [sv.po] you have 768 obsolete entries, please remove them
	❌ msgid/msgstr pattern check
	 ERROR [sv.po] mismatched patterns: refs/{heads,tags}/-prefix
	 ERROR [sv.po] >> msgid: The destination you provided is not a full refname (i.e.,
	 ERROR [sv.po] starting with "refs/"). We tried to guess what you meant by:
	 ERROR [sv.po]
	 ERROR [sv.po] - Looking for a ref that matches '\''%s'\'' on the remote side.
	 ERROR [sv.po] - Checking if the <src> being pushed ('\''%s'\'')
	 ERROR [sv.po] is a ref in "refs/{heads,tags}/". If so we add a corresponding
	 ERROR [sv.po] refs/{heads,tags}/ prefix on the remote side.
	 ERROR [sv.po]
	 ERROR [sv.po] Neither worked, so we gave up. You must fully qualify the ref.
	 ERROR [sv.po] >> msgstr: Målet du angav är inte ett komplett referensamn (dvs.,
	 ERROR [sv.po] startar med "refs/"). Vi försökte gissa vad du menade genom att:
	 ERROR [sv.po]
	 ERROR [sv.po] - Se efter en referens som motsvarar "%s" på fjärrsidan.
	 ERROR [sv.po] - Se om <källan> som sänds ("%s")
	 ERROR [sv.po] är en referens i "refs/{heads,tags}/". Om så lägger vi till
	 ERROR [sv.po] motsvarande refs/{heads,tags}/-prefix på fjärrsidan.
	 ERROR [sv.po]
	 ERROR [sv.po] Inget av dem fungerade, så vi gav upp. Ange fullständig referens.
	 ERROR [sv.po]
	ERROR: check-po command failed
	EOF
	test_cmp expect actual
'

test_done
