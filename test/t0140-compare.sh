#!/bin/sh
#
# Test compare command: --stat, --json, PO vs PO, and PO files with obsolete entries.
# Covers: default PO output, --stat diff statistics, --json output format,
# PO file comparison, and obsolete entry handling in comparison.
#

test_description="compare: --stat, --json, PO vs PO, obsolete entries"

. ./lib/test-lib.sh

HELPER="$TEST_TARGET_DIRECTORY/git-po-helper -q --no-special-gettext-versions"

test_expect_success "setup: create old.po and new.po for comparison" '
	cat >old.po <<-\EOF &&
	msgid ""
	msgstr ""
	"Content-Type: text/plain; charset=UTF-8\n"

	msgid "Hello"
	msgstr "你好"

	msgid "World"
	msgstr "世界"
	EOF

	cat >new.po <<-\EOF &&
	msgid ""
	msgstr ""
	"Content-Type: text/plain; charset=UTF-8\n"

	msgid "Hello"
	msgstr "您好"

	msgid "World"
	msgstr "世界"

	msgid "New entry"
	msgstr "新条目"
	EOF

	test -s old.po &&
	test -s new.po
'

test_expect_success "compare --stat: show diff statistics" '
	$HELPER compare --stat old.po new.po >actual 2>&1 &&
	cat >expect <<-\EOF &&
	1 new, 1 changed
	EOF
	test_cmp expect actual
'

test_expect_success "compare PO vs PO: default output has new and changed entries" '
	$HELPER compare -o out.po old.po new.po &&
	test -s out.po &&
	grep -q "msgid \"Hello\"" out.po &&
	grep -q "msgstr \"您好\"" out.po &&
	grep -q "msgid \"New entry\"" out.po &&
	grep -q "msgstr \"新条目\"" out.po
'

test_expect_success "compare --json: output JSON when there are new/changed entries" '
	$HELPER compare --json -o out.json old.po new.po &&
	test -s out.json &&
	grep -q "^{" out.json &&
	grep -q "\"msgid\":\"Hello\"" out.json &&
	grep -q "\"msgstr\":\"您好\"" out.json &&
	grep -q "\"msgid\":\"New entry\"" out.json &&
	grep -q "\"entries\"" out.json
'

test_expect_success "compare --json: empty output when no new/changed entries" '
	$HELPER compare --json -o empty.json new.po new.po &&
	test_path_is_file empty.json &&
	test ! -s empty.json
'

test_expect_success "setup: create PO files with obsolete entries" '
	cat >old-with-obsolete.po <<-\EOF &&
	msgid ""
	msgstr ""
	"Content-Type: text/plain; charset=UTF-8\n"

	msgid "Hello"
	msgstr "你好"

	#~ msgid "Obsolete entry"
	#~ msgstr "已废弃"

	msgid "World"
	msgstr "世界"
	EOF

	cat >new-with-obsolete.po <<-\EOF &&
	msgid ""
	msgstr ""
	"Content-Type: text/plain; charset=UTF-8\n"

	msgid "Hello"
	msgstr "你好"

	msgid "World"
	msgstr "世界"

	msgid "Added entry"
	msgstr "新增"
	EOF

	test -s old-with-obsolete.po &&
	test -s new-with-obsolete.po
'

test_expect_success "compare PO with obsolete: obsolete in old is skipped, new entry reported" '
	$HELPER compare --stat old-with-obsolete.po new-with-obsolete.po >actual 2>&1 &&
	cat >expect <<-\EOF &&
	1 new
	EOF
	test_cmp expect actual
'

test_expect_success "compare PO with obsolete: output contains only new content entry" '
	$HELPER compare -o out-obsolete.po old-with-obsolete.po new-with-obsolete.po &&
	test -s out-obsolete.po &&
	grep -q "msgid \"Added entry\"" out-obsolete.po &&
	grep -q "msgstr \"新增\"" out-obsolete.po &&
	! grep -q "Obsolete" out-obsolete.po
'

test_expect_success "compare PO with obsolete --json: JSON has new entry only" '
	$HELPER compare --json -o out-obsolete.json old-with-obsolete.po new-with-obsolete.po &&
	test -s out-obsolete.json &&
	grep -q "\"msgid\":\"Added entry\"" out-obsolete.json &&
	! grep -q "Obsolete" out-obsolete.json
'

test_expect_success "setup: create JSON files for comparison" '
	$HELPER msg-select --json old.po -o old.json &&
	$HELPER msg-select --json new.po -o new.json &&
	test -s old.json &&
	test -s new.json &&
	grep -q "^{" old.json
'

test_expect_success "compare JSON vs JSON: same result as PO vs PO" '
	$HELPER compare --stat old.json new.json >actual 2>&1 &&
	cat >expect <<-\EOF &&
	1 new, 1 changed
	EOF
	test_cmp expect actual
'

test_expect_success "compare JSON vs JSON --json output" '
	$HELPER compare --json -o out-json.json old.json new.json &&
	test -s out-json.json &&
	grep -q "\"msgid\":\"Hello\"" out-json.json &&
	grep -q "\"msgid\":\"New entry\"" out-json.json
'

test_expect_success "compare PO vs JSON: mixed input works" '
	$HELPER compare --stat old.po new.json >actual 2>&1 &&
	cat >expect <<-\EOF &&
	1 new, 1 changed
	EOF
	test_cmp expect actual
'

test_expect_success "compare JSON vs PO: mixed input works" '
	$HELPER compare --stat old.json new.po >actual 2>&1 &&
	cat >expect <<-\EOF &&
	1 new, 1 changed
	EOF
	test_cmp expect actual
'

test_done
