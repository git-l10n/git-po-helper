#!/bin/sh
#
# Test msg-select filter options with PO file input.
# Covers: default (all), --no-obsolete, --only-obsolete, --translated,
# --untranslated, --fuzzy, --only-same, and combinations.
#

test_description="msg-select filter options with PO file input"

. ./lib/test-lib.sh

HELPER="$TEST_TARGET_DIRECTORY/git-po-helper --no-special-gettext-versions"

# PO with: 1 translated, 2 same, 3 untranslated, 4 fuzzy, 5 obsolete
test_expect_success "setup: create filter-test.po with all entry states" '
	cat >filter-test.po <<-\EOF &&
	msgid ""
	msgstr ""
	"Content-Type: text/plain; charset=UTF-8\n"

	msgid "Translated entry"
	msgstr "已翻译"

	msgid "Same entry"
	msgstr "Same entry"

	msgid "Untranslated entry"
	msgstr ""

	#, fuzzy
	msgid "Fuzzy entry"
	msgstr "模糊"

	#~ msgid "Obsolete entry"
	#~ msgstr "已废弃"
	EOF
	test -s filter-test.po
'

test_expect_success "default: all entries including obsolete" '
	$HELPER msg-select filter-test.po -o default.po &&
	cat >expect <<-\EOF &&
	msgid ""
	msgstr ""
	"Content-Type: text/plain; charset=UTF-8\n"

	msgid "Translated entry"
	msgstr "已翻译"

	msgid "Same entry"
	msgstr "Same entry"

	msgid "Untranslated entry"
	msgstr ""

	#, fuzzy
	msgid "Fuzzy entry"
	msgstr "模糊"

	#~ msgid "Obsolete entry"
	#~ msgstr "已废弃"

	EOF
	test_cmp expect default.po
'

test_expect_success "--no-obsolete: exclude obsolete" '
	$HELPER msg-select --no-obsolete filter-test.po -o no-obsolete.po &&
	cat >expect <<-\EOF &&
	msgid ""
	msgstr ""
	"Content-Type: text/plain; charset=UTF-8\n"

	msgid "Translated entry"
	msgstr "已翻译"

	msgid "Same entry"
	msgstr "Same entry"

	msgid "Untranslated entry"
	msgstr ""

	#, fuzzy
	msgid "Fuzzy entry"
	msgstr "模糊"

	EOF
	test_cmp expect no-obsolete.po
'

test_expect_success "--only-obsolete: only obsolete entries" '
	$HELPER msg-select --only-obsolete filter-test.po -o only-obsolete.po &&
	cat >expect <<-\EOF &&
	msgid ""
	msgstr ""
	"Content-Type: text/plain; charset=UTF-8\n"

	#~ msgid "Obsolete entry"
	#~ msgstr "已废弃"

	EOF
	test_cmp expect only-obsolete.po
'

test_expect_success "--translated: translated and same (obsolete included by default)" '
	$HELPER msg-select --translated filter-test.po -o translated.po &&
	cat >expect <<-\EOF &&
	msgid ""
	msgstr ""
	"Content-Type: text/plain; charset=UTF-8\n"

	msgid "Translated entry"
	msgstr "已翻译"

	msgid "Same entry"
	msgstr "Same entry"

	#~ msgid "Obsolete entry"
	#~ msgstr "已废弃"

	EOF
	test_cmp expect translated.po
'

test_expect_success "--untranslated: untranslated only" '
	$HELPER msg-select --untranslated filter-test.po -o untranslated.po &&
	cat >expect <<-\EOF &&
	msgid ""
	msgstr ""
	"Content-Type: text/plain; charset=UTF-8\n"

	msgid "Untranslated entry"
	msgstr ""

	#~ msgid "Obsolete entry"
	#~ msgstr "已废弃"

	EOF
	test_cmp expect untranslated.po
'

test_expect_success "--fuzzy: fuzzy only" '
	$HELPER msg-select --fuzzy filter-test.po -o fuzzy.po &&
	cat >expect <<-\EOF &&
	msgid ""
	msgstr ""
	"Content-Type: text/plain; charset=UTF-8\n"

	#, fuzzy
	msgid "Fuzzy entry"
	msgstr "模糊"

	#~ msgid "Obsolete entry"
	#~ msgstr "已废弃"

	EOF
	test_cmp expect fuzzy.po
'

test_expect_success "--only-same: same only" '
	$HELPER msg-select --only-same filter-test.po -o only-same.po &&
	cat >expect <<-\EOF &&
	msgid ""
	msgstr ""
	"Content-Type: text/plain; charset=UTF-8\n"

	msgid "Same entry"
	msgstr "Same entry"

	EOF
	test_cmp expect only-same.po
'

test_expect_success "--translated --untranslated: OR combination" '
	$HELPER msg-select --translated --untranslated filter-test.po -o trans-untrans.po &&
	cat >expect <<-\EOF &&
	msgid ""
	msgstr ""
	"Content-Type: text/plain; charset=UTF-8\n"

	msgid "Translated entry"
	msgstr "已翻译"

	msgid "Same entry"
	msgstr "Same entry"

	msgid "Untranslated entry"
	msgstr ""

	#~ msgid "Obsolete entry"
	#~ msgstr "已废弃"

	EOF
	test_cmp expect trans-untrans.po
'

test_expect_success "--translated --no-obsolete: no obsolete in translated" '
	$HELPER msg-select --translated --no-obsolete filter-test.po -o trans-no-obs.po &&
	cat >expect <<-\EOF &&
	msgid ""
	msgstr ""
	"Content-Type: text/plain; charset=UTF-8\n"

	msgid "Translated entry"
	msgstr "已翻译"

	msgid "Same entry"
	msgstr "Same entry"

	EOF
	test_cmp expect trans-no-obs.po
'

test_expect_success "--unset-fuzzy: remove fuzzy marker, keep translations" '
	$HELPER msg-select --unset-fuzzy filter-test.po -o unset-fuzzy.po &&
	! grep -q "#, fuzzy" unset-fuzzy.po &&
	grep -q "msgstr \"模糊\"" unset-fuzzy.po
'

test_expect_success "--clear-fuzzy: remove fuzzy marker and clear msgstr" '
	$HELPER msg-select --clear-fuzzy filter-test.po -o clear-fuzzy.po &&
	! grep -q "#, fuzzy" clear-fuzzy.po &&
	grep -A1 "msgid \"Fuzzy entry\"" clear-fuzzy.po | grep -q "msgstr \"\""
'

test_expect_success "--unset-fuzzy and --clear-fuzzy are mutually exclusive" '
	test_must_fail $HELPER msg-select --unset-fuzzy --clear-fuzzy filter-test.po 2>err &&
	grep -q "mutually exclusive" err
'

test_expect_success "--only-same and --only-obsolete are mutually exclusive" '
	test_must_fail $HELPER msg-select --only-same --only-obsolete filter-test.po 2>err &&
	grep -q "mutually exclusive" err
'

test_expect_success "--only-same and --translated are mutually exclusive" '
	test_must_fail $HELPER msg-select --only-same --translated filter-test.po 2>err &&
	grep -q "mutually exclusive" err
'

test_done
