#!/bin/sh

test_description="test git-po-helper agent-run translate --use-local-orchestration"

. ./lib/test-lib.sh

HELPER="po-helper --no-special-gettext-versions"

# Need msgcat and msgfmt (gettext) for local orchestration
if ! command -v msgcat >/dev/null 2>&1 || ! command -v msgfmt >/dev/null 2>&1; then
	skip_all="msgcat or msgfmt (gettext) not found, skip translate local orchestration test"
	test_done
fi

# Mock agent: copy JSON and set msgstr=msgid for untranslated entries (identity "translation")
create_translate_copy_agent() {
	cat >"$1" <<'PYEOF'
#!/usr/bin/env python3
import json, sys, os
# Usage: translate-copy-agent <source> <dest>
# Reads JSON from source, sets msgstr=msgid for empty msgstr, writes to dest
if len(sys.argv) < 3:
    sys.stderr.write("Usage: translate-copy-agent <source> <dest>\n")
    sys.exit(1)
src, dst = sys.argv[1], sys.argv[2]
with open(src) as f:
    data = json.load(f)
for e in data.get("entries", []):
    if not e.get("msgstr") and e.get("msgid"):
        e["msgstr"] = e["msgid"]
    if e.get("msgid_plural") and e.get("msgstr_plural"):
        for i in range(len(e["msgstr_plural"])):
            if not e["msgstr_plural"][i]:
                e["msgstr_plural"][i] = e["msgid_plural"]
with open(dst, "w") as f:
    json.dump(data, f, ensure_ascii=False, indent=2)
PYEOF
	chmod +x "$1"
}

test_expect_success "setup" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	test -f workdir/po/zh_CN.po &&
	create_translate_copy_agent "$PWD/translate-copy-agent" &&
	# Config with copy agent that uses {{.source}} {{.dest}}
	cat >workdir/.git-po-helper.yaml <<-EOF &&
default_lang_code: "zh_CN"
prompt:
  translate: "Translate file {{.source}}"
  local_orchestration_translation: "Translate {{.source}} to {{.dest}}"
agents:
  copy:
    cmd: ["$PWD/translate-copy-agent", "{{.source}}", "{{.dest}}"]
    kind: echo
EOF
	sed -i.bak "s|\$PWD|$PWD|g" workdir/.git-po-helper.yaml &&
	rm -f workdir/.git-po-helper.yaml.bak
'

test_expect_success "agent-run translate: mutual exclusivity of mode flags" '
	test_must_fail git -C workdir $HELPER agent-run translate \
		--use-agent-md --use-local-orchestration po/zh_CN.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	grep "mutually exclusive" actual
'

test_expect_success "agent-run translate --use-local-orchestration: success" '
	git -C workdir $HELPER agent-run translate \
		--use-local-orchestration --agent copy po/zh_CN.po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&
	grep "completed successfully" actual &&
	grep "local orchestration" actual
'

test_expect_success "agent-run translate --use-local-orchestration: PO valid after run" '
	# Verify PO file syntax is valid
	msgfmt -o /dev/null -c workdir/po/zh_CN.po
'

test_done
