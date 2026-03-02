#!/bin/sh
#
# Test msg-select filter options with JSON file input.
# Same filter combinations as t0121, but input is JSON (from PO via msg-select --json).
#

test_description="msg-select filter options with JSON file input"

. ./lib/test-lib.sh

HELPER="$TEST_TARGET_DIRECTORY/git-po-helper --no-special-gettext-versions"

# Create PO, convert to JSON for JSON input tests
test_expect_success "setup: create filter-test.po and convert to JSON" '
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
	$HELPER msg-select --json filter-test.po -o filter-test.json &&
	test -s filter-test.json &&
	grep -q "^{" filter-test.json
'

test_expect_success "default: all entries including obsolete (JSON input)" '
	$HELPER msg-select filter-test.json -o default.po &&
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

test_expect_success "--no-obsolete: exclude obsolete (JSON input)" '
	$HELPER msg-select --no-obsolete filter-test.json -o no-obsolete.po &&
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

test_expect_success "--only-obsolete: only obsolete entries (JSON input)" '
	$HELPER msg-select --only-obsolete filter-test.json -o only-obsolete.po &&
	cat >expect <<-\EOF &&
	msgid ""
	msgstr ""
	"Content-Type: text/plain; charset=UTF-8\n"

	#~ msgid "Obsolete entry"
	#~ msgstr "已废弃"
	EOF
	test_cmp expect only-obsolete.po
'

test_expect_success "--translated: translated and same (JSON input)" '
	$HELPER msg-select --translated filter-test.json -o translated.po &&
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

test_expect_success "--untranslated: untranslated only (JSON input)" '
	$HELPER msg-select --untranslated filter-test.json -o untranslated.po &&
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

test_expect_success "--fuzzy: fuzzy only (JSON input)" '
	$HELPER msg-select --fuzzy filter-test.json -o fuzzy.po &&
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

test_expect_success "--only-same: same only (JSON input)" '
	$HELPER msg-select --only-same filter-test.json -o only-same.po &&
	cat >expect <<-\EOF &&
	msgid ""
	msgstr ""
	"Content-Type: text/plain; charset=UTF-8\n"

	msgid "Same entry"
	msgstr "Same entry"
	EOF
	test_cmp expect only-same.po
'

test_expect_success "--translated --untranslated: OR combination (JSON input)" '
	$HELPER msg-select --translated --untranslated filter-test.json -o trans-untrans.po &&
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

test_expect_success "--translated --no-obsolete: no obsolete (JSON input)" '
	$HELPER msg-select --translated --no-obsolete filter-test.json -o trans-no-obs.po &&
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

test_expect_success "--unset-fuzzy: JSON input to PO, no fuzzy marker" '
	$HELPER msg-select --unset-fuzzy filter-test.json -o unset-fuzzy.po &&
	! grep -q "#, fuzzy" unset-fuzzy.po &&
	grep -q "msgstr \"模糊\"" unset-fuzzy.po
'

test_expect_success "--unset-fuzzy: JSON input to JSON, no fuzzy true" '
	$HELPER msg-select --unset-fuzzy --json filter-test.json -o unset-fuzzy.json &&
	! grep -q "\"fuzzy\":true" unset-fuzzy.json
'

test_expect_success "--clear-fuzzy: JSON input, remove fuzzy and clear msgstr" '
	$HELPER msg-select --clear-fuzzy filter-test.json -o clear-fuzzy.po &&
	! grep -q "#, fuzzy" clear-fuzzy.po &&
	grep -A1 "msgid \"Fuzzy entry\"" clear-fuzzy.po | grep -q "msgstr \"\""
'

test_expect_success "--unset-fuzzy and --clear-fuzzy are mutually exclusive (JSON input)" '
	test_must_fail $HELPER msg-select --unset-fuzzy --clear-fuzzy filter-test.json 2>err &&
	grep -q "mutually exclusive" err
'

test_expect_success "--only-same and --only-obsolete are mutually exclusive (JSON input)" '
	test_must_fail $HELPER msg-select --only-same --only-obsolete filter-test.json 2>err &&
	grep -q "mutually exclusive" err
'

test_expect_success "--only-same and --translated are mutually exclusive (JSON input)" '
	test_must_fail $HELPER msg-select --only-same --translated filter-test.json 2>err &&
	grep -q "mutually exclusive" err
'

test_done
