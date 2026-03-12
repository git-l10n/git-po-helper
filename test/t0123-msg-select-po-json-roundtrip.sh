#!/bin/sh
#
# Test msg-select: PO and JSON round-trip with \n, \t, \r, \", \\.
# JSON encodes \x07 (bell) as \u0007 per RFC 8259; decodes \u0007 back to \x07.
# Creates input.po and input.json, uses msg-select to read/convert,
# and compares output with expect via test_cmp.
#
# NOTE: Do NOT pipe git-po-helper JSON output through "jq ." for comparison.
# jq 1.6 (Ubuntu/Linux) expands \u0007 to a raw bell byte (0x07) in output,
# while jq 1.7+ (macOS) keeps it as the \u0007 escape sequence.
# This version difference causes cross-platform test failures.
# git-po-helper already outputs indented JSON directly; compare it without jq.
#

test_description="msg-select PO and JSON round-trip with special chars (incl. \\u0007)"

. ./lib/test-lib.sh

HELPER="$TEST_TARGET_DIRECTORY/git-po-helper --no-special-gettext-versions"

test_expect_success "setup: create input.po and input.json with \\n, \\t, \\u0007" '
	bsl=$(printf '\''\\\\'\'') &&
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
	msgstr ""
	"第1行\n"
	"第2行\t带制表符\n"
	"第3行\r带回车\n"
	"第4行\"带引号\n"
	"第5行${bsl}带斜线\n"

	#, c-format
	msgid "Simple %s"
	msgstr "简单 %s"
	ENDPO
	cat >input.json <<-\INPJSON &&
	{
	  "header_comment": "",
	  "header_meta": "Content-Type: text/plain; charset=UTF-8\\n",
	  "entries": [
	    {
	      "msgid": "Line one\\nLine two\\twith tab\\nLine three\\rwith CR\\nLine four\\\"with quote\\nLine five\\\\with slash\\n",
	      "msgstr": [
	        "第1行\\n第2行\\t带制表符\\n第3行\\r带回车\\n第4行\\\"带引号\\n第5行\\\\带斜线\\n"
	      ],
	      "comments": [
	        "#: src/a.c"
	      ],
	      "fuzzy": false
	    },
	    {
	      "msgid": "Simple %s",
	      "msgstr": [
	        "简单 %s"
	      ],
	      "comments": [
	        "#, c-format"
	      ],
	      "fuzzy": false
	    }
	  ]
	}
	INPJSON
	test -s input.po &&
	test -s input.json
'

test_expect_success "msg-select: PO -> JSON -> compare" '
	$HELPER msg-select --range "1-" --json input.po >po2json.json &&
	test_cmp input.json po2json.json
'

test_expect_success "msg-select: JSON -> PO -> compare" '
	$HELPER msg-select --range "1-" -o json2po.po input.json &&
	bsl=$(printf '\''\\\\'\'') &&
	cat >expect <<-ENDPO &&
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
	msgstr ""
	"第1行\n"
	"第2行\t带制表符\n"
	"第3行\r带回车\n"
	"第4行\"带引号\n"
	"第5行${bsl}带斜线\n"

	#, c-format
	msgid "Simple %s"
	msgstr "简单 %s"
	ENDPO
	test_cmp expect json2po.po &&
	test_cmp input.po json2po.po
'

test_done
