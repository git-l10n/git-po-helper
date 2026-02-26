#!/bin/sh

test_description="test git-po-helper agent-test update-pot and update-po"

. ./lib/test-lib.sh

HELPER="po-helper --no-special-gettext-versions"

# Create a mock agent script that simulates agent behavior
create_mock_agent() {
	cat >"$1" <<-\EOF
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
	create_mock_agent "$PWD/mock-agent"
'

test_expect_success "agent-test update-pot: basic test with default runs" '
	cat >workdir/git-po-helper.yaml <<-EOF &&
prompt:
  update_pot: "update po/git.pot according to po/AGENTS.md"
agents:
  mock:
    cmd: ["$PWD/mock-agent", "--prompt", "{{.prompt}}"]
    kind: echo
EOF

	# Remove any previous mock agent comments
	sed -i.bak "/Updated by mock agent/d" workdir/po/git.pot &&
	rm -f workdir/po/git.pot.bak &&

	git -C workdir $HELPER agent-test --dangerously-remove-po-directory update-pot >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	# Should show test results
	grep "Agent Test Results" actual &&
	grep "Summary" actual &&
	grep "Average score" actual &&

	# Should complete successfully
	grep "completed successfully" actual
'

test_expect_success "agent-test update-pot: with --runs flag" '
	cat >workdir/git-po-helper.yaml <<-EOF &&
prompt:
  update_pot: "update po/git.pot according to po/AGENTS.md"
agents:
  mock:
    cmd: ["$PWD/mock-agent", "--prompt", "{{.prompt}}"]
    kind: echo
EOF

	# Remove any previous mock agent comments
	sed -i.bak "/Updated by mock agent/d" workdir/po/git.pot &&
	rm -f workdir/po/git.pot.bak &&

	git -C workdir $HELPER agent-test --dangerously-remove-po-directory update-pot --runs 3 >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	# Should show 3 runs
	grep "Run 3:" actual &&
	! grep "Run 4:" actual &&

	# Should show summary with 3 total runs
	grep "Total runs:.*3" actual
'

test_expect_success "agent-test update-pot: with validation" '
	# Count entries in pot file
	ENTRY_COUNT=$(grep -c "^msgid " workdir/po/git.pot | head -1) &&
	ENTRY_COUNT=$((ENTRY_COUNT - 1)) &&

	cat >workdir/git-po-helper.yaml <<-EOF &&
prompt:
  update_pot: "update po/git.pot according to po/AGENTS.md"
agent-test:
  runs: 2
  pot_entries_before_update: $ENTRY_COUNT
  pot_entries_after_update: $ENTRY_COUNT
agents:
  mock:
    cmd: ["$PWD/mock-agent", "--prompt", "{{.prompt}}"]
    kind: echo
EOF

	# Remove any previous mock agent comments
	sed -i.bak "/Updated by mock agent/d" workdir/po/git.pot &&
	rm -f workdir/po/git.pot.bak &&

	git -C workdir $HELPER agent-test --dangerously-remove-po-directory update-pot >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	# Should show validation status
	grep "Pre-validation" actual &&
	grep "Post-validation" actual &&

	# Should show all runs passed
	grep "Run 1: PASS" actual &&
	grep "Run 2: PASS" actual &&

	# Should show average score
	grep "Average score" actual
'

test_expect_success "agent-test update-pot: pre-validation failure" '
	# Count entries in pot file
	ENTRY_COUNT=$(grep -c "^msgid " workdir/po/git.pot | head -1) &&
	ENTRY_COUNT=$((ENTRY_COUNT - 1)) &&
	WRONG_COUNT=$((ENTRY_COUNT + 100)) &&

	cat >workdir/git-po-helper.yaml <<-EOF &&
prompt:
  update_pot: "update po/git.pot according to po/AGENTS.md"
agent-test:
  runs: 2
  pot_entries_before_update: $WRONG_COUNT
agents:
  mock:
    cmd: ["$PWD/mock-agent", "--prompt", "{{.prompt}}"]
    kind: echo
EOF

	# Remove any previous mock agent comments
	sed -i.bak "/Updated by mock agent/d" workdir/po/git.pot &&
	rm -f workdir/po/git.pot.bak &&

	git -C workdir $HELPER agent-test --dangerously-remove-po-directory update-pot >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	# Should show pre-validation failures
	grep "Pre-validation.*FAIL" actual &&

	# Should show agent execution was skipped
	grep "Agent execution: SKIPPED" actual &&

	# Should show failed runs
	grep "Run.*FAIL" actual
'

test_expect_success "agent-test update-pot: post-validation failure" '
	# Count entries in pot file
	ENTRY_COUNT=$(grep -c "^msgid " workdir/po/git.pot | head -1) &&
	ENTRY_COUNT=$((ENTRY_COUNT - 1)) &&
	WRONG_COUNT=$((ENTRY_COUNT + 100)) &&

	cat >workdir/git-po-helper.yaml <<-EOF &&
prompt:
  update_pot: "update po/git.pot according to po/AGENTS.md"
agent-test:
  runs: 2
  pot_entries_after_update: $WRONG_COUNT
agents:
  mock:
    cmd: ["$PWD/mock-agent", "--prompt", "{{.prompt}}"]
    kind: echo
EOF

	# Remove any previous mock agent comments
	sed -i.bak "/Updated by mock agent/d" workdir/po/git.pot &&
	rm -f workdir/po/git.pot.bak &&

	git -C workdir $HELPER agent-test --dangerously-remove-po-directory update-pot >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	# Should show post-validation failures
	grep "Post-validation.*FAIL" actual &&

	# Should show failed runs
	grep "Run.*FAIL" actual
'

test_expect_success "agent-test update-pot: with failing agent" '
	# Create a failing mock agent
	cat >"$PWD/failing-agent" <<-EOF &&
#!/bin/sh
echo "Agent failed" >&2
exit 1
EOF
	chmod +x "$PWD/failing-agent" &&

	cat >workdir/git-po-helper.yaml <<-EOF &&
prompt:
  update_pot: "update po/git.pot according to po/AGENTS.md"
agent-test:
  runs: 2
agents:
  failing:
    cmd: ["$PWD/failing-agent"]
    kind: echo
EOF

	git -C workdir $HELPER agent-test --dangerously-remove-po-directory update-pot >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	# Should show failed runs
	grep "Run.*FAIL" actual &&

	# Should show agent execution failures
	grep "Agent execution: FAIL" actual &&

	# Should show average score (likely 0 if all failed)
	grep "Average score" actual
'

test_expect_success "agent-test update-po: basic test with default runs" '
	test -f workdir/po/zh_CN.po &&

	cat >workdir/git-po-helper.yaml <<-EOF &&
default_lang_code: "zh_CN"
prompt:
  update_po: "update {{.source}} according to po/AGENTS.md"
agent-test:
  runs: 2
agents:
  mock:
    cmd: ["$PWD/mock-agent", "--prompt", "{{.prompt}}", "{{.source}}"]
    kind: echo
EOF

	# Remove any previous mock agent comments from zh_CN.po
	sed -i.bak "/Updated by mock agent/d" workdir/po/zh_CN.po &&
	rm -f workdir/po/zh_CN.po.bak &&

	git -C workdir $HELPER agent-test --dangerously-remove-po-directory update-po >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	# Should show test results
	grep "Agent Test Results" actual &&
	grep "Summary" actual &&
	grep "Average score" actual &&

	# Should complete successfully
	grep "completed successfully" actual
'

test_expect_success "agent-test update-pot: with --prompt override" '
	cat >workdir/git-po-helper.yaml <<-EOF &&
prompt:
  update_pot: "config prompt for update pot"
agent-test:
  runs: 2
agents:
  mock:
    cmd: ["$PWD/mock-agent", "--prompt", "{{.prompt}}"]
    kind: echo
EOF

	# Remove any previous mock agent comments
	sed -i.bak "/Updated by mock agent/d" workdir/po/git.pot &&
	rm -f workdir/po/git.pot.bak &&

	git -C workdir $HELPER agent-test --dangerously-remove-po-directory update-pot --prompt "override prompt from command line" >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	# Should show test results
	grep "Agent Test Results" actual &&
	grep "Summary" actual &&

	# Should complete successfully
	grep "completed successfully" actual &&

	# Verify the override prompt was used in all runs
	grep "override prompt from command line" workdir/po/git.pot
'

test_expect_success "agent-test update-po: with --prompt override" '
	test -f workdir/po/zh_CN.po &&

	cat >workdir/git-po-helper.yaml <<-EOF &&
default_lang_code: "zh_CN"
prompt:
  update_po: "config prompt for update po"
agent-test:
  runs: 2
agents:
  mock:
    cmd: ["$PWD/mock-agent", "--prompt", "{{.prompt}}", "{{.source}}"]
    kind: echo
EOF

	# Remove any previous mock agent comments from zh_CN.po
	sed -i.bak "/Updated by mock agent/d" workdir/po/zh_CN.po &&
	rm -f workdir/po/zh_CN.po.bak &&

	git -C workdir $HELPER agent-test --dangerously-remove-po-directory update-po --prompt "override prompt for update-po test" >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	# Should show test results
	grep "Agent Test Results" actual &&
	grep "Summary" actual &&

	# Should complete successfully
	grep "completed successfully" actual &&

	# Verify the override prompt was used
	grep "override prompt for update-po test" workdir/po/zh_CN.po
'

test_done
