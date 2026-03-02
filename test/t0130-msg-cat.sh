#!/bin/sh
#
# Test msg-cat: merge PO/POT/JSON from fixture and custom example, check entry
# count and fuzzy count. Uses msg-select to extract ranges 1-3 and 10-15, filters
# ",fuzzy" from 1-3 JSON, creates example.po with 2 entries (one translated,
# one fuzzy), then msg-cat merge and verify.
#

test_description="msg-cat: merge PO/JSON, check entry count and fuzzy"

. ./lib/test-lib.sh

HELPER="$TEST_TARGET_DIRECTORY/git-po-helper --no-special-gettext-versions"
FIXTURE="$TEST_DIRECTORY/fixtures/zh_CN_example.po"

test_expect_success "setup: fixture exists" '
	test -f "$FIXTURE" &&
	head -1 "$FIXTURE" | grep -q "Chinese translations"
'

test_expect_success "msg-select range 1-3 to JSON and PO" '
	$HELPER msg-select --range "1-3" --json "$FIXTURE" -o range1-3.json &&
	$HELPER msg-select --range "1-3" "$FIXTURE" -o range1-3.po &&
	test -s range1-3.json &&
	test -s range1-3.po &&
	grep -q "^{" range1-3.json
'

test_expect_success "filter out fuzzy from range1-3.json" '
	grep -q "\"fuzzy\"" range1-3.json &&
	sed -e "s/,\"fuzzy\":true//g" -e "s/,\"fuzzy\":false//g" \
		range1-3.json >range1-3-no-fuzzy.json &&
	test -s range1-3-no-fuzzy.json &&
	! grep -q "\"fuzzy\"" range1-3-no-fuzzy.json
'

test_expect_success "msg-select range 10-15 to JSON and PO" '
	$HELPER msg-select --range "10-15" --json "$FIXTURE" -o range10-15.json &&
	$HELPER msg-select --range "10-15" "$FIXTURE" -o range10-15.po &&
	test -s range10-15.json &&
	test -s range10-15.po
'

test_expect_success "create example.po with 2 entries (one translated, one fuzzy)" '
	cat >example.po <<\EOF &&
# Example custom entries
msgid ""
msgstr ""
"Content-Type: text/plain; charset=UTF-8\n"

msgid "Custom translated"
msgstr "Custom translation"

#, fuzzy
msgid "Custom fuzzy"
msgstr ""
EOF
	test -s example.po
'

test_expect_success "check stat of FIXTURE" '
	cat >expect <<-EOF &&
	fixture.po: 14 translated messages, 6 fuzzy translations.
	EOF
	cp "$FIXTURE" fixture.po &&
	$HELPER stat fixture.po >actual &&
	test_cmp expect actual
'

test_expect_success "check stat of range1-3-no-fuzzy (PO)" '
	cat >expect <<-EOF &&
	range1-3-no-fuzzy.po: 3 translated messages.
	EOF
	$HELPER msg-select --range "1-" range1-3-no-fuzzy.json \
		>range1-3-no-fuzzy.po &&
	$HELPER stat range1-3-no-fuzzy.po >actual &&
	test_cmp expect actual
'

test_expect_success "check stat of range1-3-no-fuzzy (JSON)" '
	cat >expect <<-EOF &&
	range1-3-no-fuzzy.json: 3 translated messages.
	EOF
	$HELPER stat range1-3-no-fuzzy.json >actual &&
	test_cmp expect actual
'

test_expect_success "msg-cat merge: range1-3-no-fuzzy.json, range10-15.po, example.po" '
	cat >expect <<-EOF &&
	merged.po: 17 translated messages, 3 fuzzy translations.
	EOF
	$HELPER msg-cat -o merged.po \
		range1-3-no-fuzzy.json \
		fixture.po &&
	test -s merged.po &&
	$HELPER stat merged.po >actual &&
	test_cmp expect actual
'

test_expect_success "msg-cat --unset-fuzzy: PO output has no fuzzy marker" '
	$HELPER msg-cat --unset-fuzzy -o merged-cleared.po fixture.po &&
	test -s merged-cleared.po &&
	! grep -q "#, fuzzy" merged-cleared.po &&
	$HELPER stat merged-cleared.po | grep -q "translated" &&
	! $HELPER stat merged-cleared.po | grep -q "fuzzy"
'

test_expect_success "msg-cat --unset-fuzzy: JSON output has no fuzzy true" '
	$HELPER msg-cat --unset-fuzzy --json -o merged-cleared.json fixture.po &&
	test -s merged-cleared.json &&
	! grep -q "\"fuzzy\":true" merged-cleared.json
'

test_expect_success "msg-cat --clear-fuzzy: remove fuzzy tag and clear msgstr" '
	cat >expect <<-EOF &&
	merged-clear.po: 14 translated messages, 6 untranslated messages.
	EOF
	$HELPER msg-cat --clear-fuzzy -o merged-clear.po fixture.po &&
	test -s merged-clear.po &&
	! grep -q "#, fuzzy" merged-clear.po &&
	$HELPER stat merged-clear.po >actual &&
	test_cmp expect actual
'

test_expect_success "msg-cat --clear-fuzzy: JSON output, former fuzzy have empty msgstr" '
	$HELPER msg-cat --clear-fuzzy --json -o merged-clear.json fixture.po &&
	test -s merged-clear.json &&
	! grep -q "\"fuzzy\":true" merged-clear.json
'

test_done
