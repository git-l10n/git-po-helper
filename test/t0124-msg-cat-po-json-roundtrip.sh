#!/bin/sh
#
# Test msg-cat: PO and JSON round-trip with \n, \t in strings.
# Creates input.po and input.json, uses msg-cat to read/convert,
# formats JSON with jq before writing, compares output with expect via test_cmp.
#

test_description="msg-cat PO and JSON round-trip with special chars"

. ./lib/test-lib.sh

HELPER="$TEST_TARGET_DIRECTORY/git-po-helper --no-special-gettext-versions"

if ! command -v jq >/dev/null 2>&1; then
	skip_all="jq not found, skip msg-cat PO/JSON round-trip test"
	test_done
fi

test_expect_success "setup: create input.po and input.json with \\n, \\t, \\u0007" '
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
	cat >input.json <<-\INPJSON &&
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
	INPJSON
	test -s input.po &&
	test -s input.json
'

test_expect_success "msg-cat: PO -> JSON (jq format) -> compare" '
	$HELPER msg-cat --json input.po | jq . >po2json.json &&
	cat >expect <<-\EXPJSON &&
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
	EXPJSON
	test_cmp expect po2json.json &&
	test_cmp expect input.json
'

test_expect_success "msg-cat: JSON -> PO -> compare" '
	$HELPER msg-cat -o json2po.po input.json &&
	cat >expect <<-EOF &&
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
	EOF
	test_cmp expect json2po.po &&
	test_cmp input.po json2po.po
'

test_done
