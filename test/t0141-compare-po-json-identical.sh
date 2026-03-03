#!/bin/sh
#
# Test compare: PO vs JSON when both have identical content.
# Based on t0124-msg-cat-po-json-roundtrip.sh setup.
# Verifies compare and compare --msgid report no changes for identical PO/JSON.
#

test_description="compare: identical PO and JSON, include --msgid"

. ./lib/test-lib.sh

HELPER="$TEST_TARGET_DIRECTORY/git-po-helper -q --no-special-gettext-versions"

if ! command -v jq >/dev/null 2>&1; then
	skip_all="jq not found, skip compare PO/JSON identical test"
	test_done
fi

test_expect_success "setup: create identical input.po and input.json" '
	bell=$(printf "\x07") && bsl=$(printf '\''\\\\'\'') &&
	cat >input.po <<-ENDPO &&
	msgid ""
	msgstr ""
	"Content-Type: text/plain; charset=UTF-8\n"

	#: src/a.c
	msgid ""
	"Line one\n"
	"Line two\twith tab\n"
	"Line three\rwith CR\n"
	"Line four\"with quote\n"
	"Line five${bsl}with slash\n"
	"Line six${bell}with bell\n"
	msgstr ""
	"第1行\n"
	"第2行\t带制表符\n"
	"第3行\r带回车\n"
	"第4行\"带引号\n"
	"第5行${bsl}带斜线\n"
	"第6行${bell}带铃\n"

	#, c-format
	msgid "Simple %s"
	msgstr "简单 %s"
	ENDPO
	cat >>input.json <<-\ENDJSON &&
	{
	  "header_comment": "",
	  "header_meta": "Content-Type: text/plain; charset=UTF-8\\n",
	  "entries": [
	    {
	      "msgid": "Line one\\nLine two\\twith tab\\nLine three\\rwith CR\\nLine four\\\"with quote\\nLine five\\\\with slash\\nLine six\u0007with bell\\n",
	      "msgstr": "第1行\\n第2行\\t带制表符\\n第3行\\r带回车\\n第4行\\\"带引号\\n第5行\\\\带斜线\\n第6行\u0007带铃\\n",
	      "comments": [
	        "#: src/a.c"
	      ],
	      "fuzzy": false
	    },
	    {
	      "msgid": "Simple %s",
	      "msgstr": "简单 %s",
	      "comments": [
	        "#, c-format"
	      ],
	      "fuzzy": false
	    }
	  ]
	}
	ENDJSON
	$HELPER msg-cat --json input.po | jq . >po2json.json &&
	test -s input.po &&
	test -s input.json &&
	test_cmp input.json po2json.json
'

test_expect_success "compare --stat: PO vs PO (same file) reports no changes" '
	$HELPER compare --stat input.po input.po >actual 2>&1 &&
	cat >expect <<-\EOF &&
	Nothing changed.
	EOF
	test_cmp expect actual
'

test_expect_success "compare --stat: JSON vs JSON (same file) reports no changes" '
	$HELPER compare --stat input.json input.json >actual 2>&1 &&
	cat >expect <<-\EOF &&
	Nothing changed.
	EOF
	test_cmp expect actual
'

test_expect_success "compare --stat: PO vs JSON (identical content) reports no changes" '
	$HELPER compare --stat input.po input.json >actual 2>&1 &&
	cat >expect <<-\EOF &&
	Nothing changed.
	EOF
	test_cmp expect actual
'

test_expect_success "compare --stat: JSON vs PO (identical content) reports no changes" '
	$HELPER compare --stat input.json input.po >actual 2>&1 &&
	cat >expect <<-\EOF &&
	Nothing changed.
	EOF
	test_cmp expect actual
'

test_expect_success "compare --stat --msgid: PO vs JSON reports no changes" '
	$HELPER compare --stat --msgid input.po input.json >actual 2>&1 &&
	cat >expect <<-\EOF &&
	Nothing changed.
	EOF
	test_cmp expect actual
'

test_expect_success "compare --stat --msgid: JSON vs PO reports no changes" '
	$HELPER compare --stat --msgid input.json input.po >actual 2>&1 &&
	cat >expect <<-\EOF &&
	Nothing changed.
	EOF
	test_cmp expect actual
'

test_expect_success "compare --assert-no-changes: PO vs JSON passes" '
	$HELPER compare --assert-no-changes input.po input.json
'

test_expect_success "compare --assert-no-changes: JSON vs PO passes" '
	$HELPER compare --assert-no-changes input.json input.po
'

test_expect_success "compare --assert-changes: PO vs JSON fails (identical)" '
	test_must_fail $HELPER compare --assert-changes input.po input.json 2>stderr &&
	grep -q "assert-changes failed" stderr &&
	grep -q "no new or changed entries" stderr
'

test_expect_success "compare -o: PO vs JSON produces empty output" '
	$HELPER compare -o out.po input.po input.json &&
	test_path_is_file out.po &&
	test ! -s out.po
'

test_expect_success "compare --json -o: PO vs JSON produces empty JSON" '
	$HELPER compare --json -o out.json input.po input.json &&
	test_path_is_file out.json &&
	test ! -s out.json
'

test_done
