#!/bin/sh
#
# Test stat -c/--count: print content entry count (excluding header).
#

test_description="stat -c/--count: entry count excluding header"

. ./lib/test-lib.sh

HELPER="$TEST_TARGET_DIRECTORY/git-po-helper --no-special-gettext-versions"

test_expect_success "setup: PO with header and 3 content entries" '
	cat >sample.po <<-\EOF &&
	msgid ""
	msgstr ""
	"Content-Type: text/plain; charset=UTF-8\n"

	msgid "Hello"
	msgstr "你好"

	msgid "World"
	msgstr ""

	#, fuzzy
	msgid "Fuzzy"
	msgstr "模糊"
	EOF
	test -s sample.po
'

test_expect_success "stat -c counts content entries (excludes header)" '
	echo 3 >expect &&
	$HELPER stat -c sample.po >actual &&
	test_cmp expect actual
'

test_expect_success "stat --count is the same as -c" '
	echo 3 >expect &&
	$HELPER stat --count sample.po >actual &&
	test_cmp expect actual
'

test_expect_success "stat -c on header-only PO prints 0" '
	cat >header-only.po <<-\EOF &&
	msgid ""
	msgstr ""
	"Content-Type: text/plain; charset=UTF-8\n"
	EOF
	echo 0 >expect &&
	$HELPER stat -c header-only.po >actual &&
	test_cmp expect actual
'

test_expect_success "stat -c includes obsolete entries" '
	cat >with-obsolete.po <<-\EOF &&
	msgid ""
	msgstr ""
	"Content-Type: text/plain; charset=UTF-8\n"

	msgid "Live"
	msgstr "活着"

	#~ msgid "Gone"
	#~ msgstr "已删除"
	EOF
	echo 2 >expect &&
	$HELPER stat -c with-obsolete.po >actual &&
	test_cmp expect actual
'

test_expect_success "stat -c on JSON counts entries" '
	$HELPER msg-select --json --no-header sample.po -o sample.json &&
	echo 3 >expect &&
	$HELPER stat -c sample.json >actual &&
	test_cmp expect actual
'

test_expect_success "stat -c detects PO content in .tmp by content not extension" '
	cp sample.po sample.tmp &&
	echo 3 >expect &&
	$HELPER stat -c sample.tmp >actual &&
	test_cmp expect actual
'

test_expect_success "stat -c detects JSON content in .tmp by content not extension" '
	cp sample.json sample-json.tmp &&
	echo 3 >expect &&
	$HELPER stat -c sample-json.tmp >actual &&
	test_cmp expect actual
'

test_expect_success "stat -c missing file: stdout 0, stderr error, non-zero exit" '
	echo 0 >expect &&
	test_must_fail $HELPER stat -c missing.po >actual 2>err &&
	test_cmp expect actual &&
	grep -q "file does not exist: missing.po" err
'

test_expect_success "stat -c multiple files: one count per line" '
	cat >expect <<-EOF &&
	3
	2
	EOF
	$HELPER stat -c sample.po with-obsolete.po >actual &&
	test_cmp expect actual
'

test_expect_success "stat -c mixed existing and missing" '
	cat >expect <<-EOF &&
	3
	0
	2
	EOF
	test_must_fail $HELPER stat -c sample.po missing.po with-obsolete.po >actual 2>err &&
	test_cmp expect actual &&
	grep -q "file does not exist: missing.po" err
'

test_done
