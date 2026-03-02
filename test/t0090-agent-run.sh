#!/bin/sh

test_description="test git-po-helper agent-run update-pot and update-po"

. ./lib/test-lib.sh

HELPER="po-helper --no-special-gettext-versions"

# Create a mock agent script that simulates agent behavior
create_mock_agent() {
	cat >"$1" <<\EOF
#!/bin/sh
# Mock agent that updates po/git.pot or a specific po/XX.po
# Usage: mock-agent --prompt "<prompt>" [<source>]

# Parse arguments
SOURCE=""
while [ $# -gt 0 ]; do
	case "$1" in
		--prompt|-p)
			shift
			PROMPT="$1"
			;;
		*)
			# Treat the first non-flag argument as source path if not set
			if [ -z "$SOURCE" ]; then
				SOURCE="$1"
			fi
			;;
	esac
	shift
done

# If a source path is provided, update that PO file (used by update-po)
if [ -n "$SOURCE" ]; then
	if [ -f "$SOURCE" ]; then
		echo "# Updated by mock agent: $PROMPT" >>"$SOURCE"
		exit 0
	else
		echo "Error: $SOURCE not found" >&2
		exit 1
	fi
fi

# Fallback: simulate agent work on po/git.pot (used by update-pot)
if [ -f "po/git.pot" ]; then
	# Add a comment to indicate the file was updated
	echo "# Updated by mock agent: $PROMPT" >> po/git.pot
	exit 0
else
	echo "Error: po/git.pot not found" >&2
	exit 1
fi
EOF
	chmod +x "$1"
}

test_expect_success "setup" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir &&
	test -f workdir/po/git.pot &&
	# Create mock agent
	create_mock_agent "$PWD/mock-agent" &&
	# Create config file
	cat >workdir/.git-po-helper.yaml <<-\EOF &&
prompt:
  update_pot: "update po/git.pot according to po/AGENTS.md"
agents:
  mock:
    cmd: ["$PWD/mock-agent", "--prompt", "{{.prompt}}"]
    kind: echo
EOF
	# Replace $PWD with actual path in config
	sed -i.bak "s|\$PWD|$PWD|g" workdir/.git-po-helper.yaml &&
	rm -f workdir/.git-po-helper.yaml.bak
'

test_expect_success "agent-run update-pot: no config file" '
	rm -f workdir/.git-po-helper.yaml &&
	# Without config file, default config is used (with default test agent)
	test_must_fail git -C workdir $HELPER agent-run update-pot >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	# Should complete successfully with default agent
	grep "multiple agents configured .*, --agent flag required" actual
'

test_expect_success "agent-run update-pot: multiple agents without --agent" '
	cat >workdir/.git-po-helper.yaml <<-\EOF &&
prompt:
  update_pot: "update po/git.pot"
agents:
  agent1:
    cmd: ["$PWD/mock-agent", "--prompt", "{{.prompt}}"]
    kind: echo
  agent2:
    cmd: ["$PWD/mock-agent", "--prompt", "{{.prompt}}"]
    kind: echo
EOF
	sed -i.bak "s|\$PWD|$PWD|g" workdir/.git-po-helper.yaml &&
	rm -f workdir/.git-po-helper.yaml.bak &&

	test_must_fail git -C workdir $HELPER agent-run update-pot >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	# Should mention multiple agents
	grep "multiple agents configured" actual
'

test_expect_success "agent-run update-pot: agent not found" '
	cat >workdir/.git-po-helper.yaml <<-\EOF &&
prompt:
  update_pot: "update po/git.pot"
agents:
  mock:
    cmd: ["$PWD/mock-agent", "--prompt", "{{.prompt}}"]
    kind: echo
EOF
	sed -i.bak "s|\$PWD|$PWD|g" workdir/.git-po-helper.yaml &&
	rm -f workdir/.git-po-helper.yaml.bak &&

	test_must_fail git -C workdir $HELPER agent-run update-pot --agent nonexistent >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	# Should mention agent not found
	grep "agent.*not found" actual
'

test_expect_success "agent-run update-pot: success with single agent" '
	cat >workdir/.git-po-helper.yaml <<\EOF &&
prompt:
  update_pot: "update po/git.pot according to po/AGENTS.md"
agents:
  mock:
    cmd: ["$PWD/mock-agent", "--prompt", "{{.prompt}}"]
    kind: echo
EOF
	sed -i.bak "s|\$PWD|$PWD|g" workdir/.git-po-helper.yaml &&
	rm -f workdir/.git-po-helper.yaml.bak &&

	# Save original pot file size
	ORIG_SIZE=$(wc -l < workdir/po/git.pot) &&

	git -C workdir $HELPER agent-run update-pot --agent mock >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	# Should complete successfully
	grep "completed successfully" actual &&

	# Verify pot file was updated (should have mock agent comment)
	grep "Updated by mock agent" workdir/po/git.pot
'

test_expect_success "agent-run update-pot: success with --agent flag" '
	cat >workdir/.git-po-helper.yaml <<-\EOF &&
prompt:
  update_pot: "update po/git.pot according to po/AGENTS.md"
agents:
  mock:
    cmd: ["$PWD/mock-agent", "--prompt", "{{.prompt}}"]
    kind: echo
  mock2:
    cmd: ["$PWD/mock-agent", "--prompt", "{{.prompt}}"]
    kind: echo
EOF
	sed -i.bak "s|\$PWD|$PWD|g" workdir/.git-po-helper.yaml &&
	rm -f workdir/.git-po-helper.yaml.bak &&

	# Remove previous mock agent comment
	sed -i.bak "/Updated by mock agent/d" workdir/po/git.pot &&
	rm -f workdir/po/git.pot.bak &&

	git -C workdir $HELPER agent-run update-pot --agent mock >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	# Should complete successfully
	grep "completed successfully" actual &&

	# Verify pot file was updated
	grep "Updated by mock agent" workdir/po/git.pot
'

test_expect_success "agent-run update-pot: with pre-validation" '
	# Count entries in pot file
	ENTRY_COUNT=$(grep -c "^msgid " workdir/po/git.pot | head -1) &&
	ENTRY_COUNT=$((ENTRY_COUNT - 1)) &&

	cat >workdir/.git-po-helper.yaml <<-EOF &&
prompt:
  update_pot: "update po/git.pot according to po/AGENTS.md"
agent-test:
  pot_entries_before_update: $ENTRY_COUNT
agents:
  mock:
    cmd: ["$PWD/mock-agent", "--prompt", "{{.prompt}}"]
    kind: echo
EOF

	# Remove previous mock agent comment
	sed -i.bak "/Updated by mock agent/d" workdir/po/git.pot &&
	rm -f workdir/po/git.pot.bak &&

	git -C workdir $HELPER agent-run update-pot >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	# Should complete successfully with pre-validation
	grep "completed successfully" actual
'

test_expect_success "agent-run update-pot: agent command failure" '
	# Create a failing mock agent
	cat >"$PWD/failing-agent" <<-EOF &&
#!/bin/sh
echo "Agent failed" >&2
exit 1
EOF
	chmod +x "$PWD/failing-agent" &&

	cat >workdir/.git-po-helper.yaml <<-EOF &&
prompt:
  update_pot: "update po/git.pot according to po/AGENTS.md"
agents:
  failing:
    cmd: ["$PWD/failing-agent"]
    kind: echo
EOF

	test_must_fail git -C workdir $HELPER agent-run update-pot >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	# Should mention agent command failure
	grep "agent command failed" actual
'

test_expect_success "agent-run update-po: success using default_lang_code" '
	test -f workdir/po/zh_CN.po &&

	cat >workdir/.git-po-helper.yaml <<-EOF &&
default_lang_code: "zh_CN"
prompt:
  update_po: "update {{.source}} according to po/AGENTS.md"
agents:
  mock:
    cmd: ["$PWD/mock-agent", "--prompt", "{{.prompt}}", "{{.source}}"]
    kind: echo
EOF

	# Remove previous mock agent comments from zh_CN.po
	sed -i.bak "/Updated by mock agent/d" workdir/po/zh_CN.po &&
	rm -f workdir/po/zh_CN.po.bak &&

	git -C workdir $HELPER agent-run update-po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	# Should complete successfully
	grep "completed successfully" actual &&

	# Verify PO file was updated
	grep "Updated by mock agent" workdir/po/zh_CN.po
'

test_expect_success "agent-run update-pot: with --prompt override" '
	cat >workdir/.git-po-helper.yaml <<-EOF &&
prompt:
  update_pot: "config prompt for update pot"
agents:
  mock:
    cmd: ["$PWD/mock-agent", "--prompt", "{{.prompt}}"]
    kind: echo
EOF

	# Remove previous mock agent comments
	sed -i.bak "/Updated by mock agent/d" workdir/po/git.pot &&
	rm -f workdir/po/git.pot.bak &&

	git -C workdir $HELPER agent-run update-pot --prompt "override prompt from command line" >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	# Should complete successfully
	grep "completed successfully" actual &&

	# Verify the override prompt was used (check in pot file)
	grep "override prompt from command line" workdir/po/git.pot
'

test_expect_success "agent-run update-po: with --prompt override" '
	test -f workdir/po/zh_CN.po &&

	cat >workdir/.git-po-helper.yaml <<-EOF &&
default_lang_code: "zh_CN"
prompt:
  update_po: "config prompt for update po"
agents:
  mock:
    cmd: ["$PWD/mock-agent", "--prompt", "{{.prompt}}", "{{.source}}"]
    kind: echo
EOF

	# Remove previous mock agent comments from zh_CN.po
	sed -i.bak "/Updated by mock agent/d" workdir/po/zh_CN.po &&
	rm -f workdir/po/zh_CN.po.bak &&

	git -C workdir $HELPER agent-run update-po --prompt "override prompt for update-po" >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	# Should complete successfully
	grep "completed successfully" actual &&

	# Verify the override prompt was used
	grep "override prompt for update-po" workdir/po/zh_CN.po
'

test_done
