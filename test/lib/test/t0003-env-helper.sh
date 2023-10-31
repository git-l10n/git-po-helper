#!/bin/sh
#
# Copyright (c) 2023 Jiang Xin
#

test_description='Test on test-tool env-helper'

. ../test-lib.sh

test_expect_success 'boolean env not set as false' '
	test_must_fail test-tool env-helper --type bool \
		TEST_ENV_HELPER_VAR1 >actual &&
	cat >expect <<-EOF &&
		false
	EOF
	test_cmp expect actual
'

test_expect_success 'env set as boolean (true)' '
	env TEST_ENV_HELPER_VAR1=true \
		test-tool env-helper --type bool TEST_ENV_HELPER_VAR1 >actual &&
	cat >expect <<-EOF &&
		true
	EOF
	test_cmp expect actual
'

test_expect_success 'env set as boolean (yes)' '
	env TEST_ENV_HELPER_VAR1=yes \
		test-tool env-helper --type bool TEST_ENV_HELPER_VAR1 >actual &&
	cat >expect <<-EOF &&
		true
	EOF
	test_cmp expect actual
'

test_expect_success 'env default to boolean (on)' '
	test-tool env-helper --type bool --default on \
		TEST_ENV_HELPER_VAR1 >actual &&
	cat >expect <<-EOF &&
		true
	EOF
	test_cmp expect actual
'

test_expect_success 'env default to boolean (1)' '
	test-tool env-helper --type bool --default 1 \
		TEST_ENV_HELPER_VAR1 >actual &&
	cat >expect <<-EOF &&
		true
	EOF
	test_cmp expect actual
'

test_expect_success 'env set as boolean (false)' '
	test_must_fail env TEST_ENV_HELPER_VAR1=false \
		test-tool env-helper --type bool TEST_ENV_HELPER_VAR1 >actual &&
	cat >expect <<-EOF &&
		false
	EOF
	test_cmp expect actual
'

test_expect_success 'env set as boolean (no)' '
	test_must_fail env TEST_ENV_HELPER_VAR1=no \
		test-tool env-helper --type bool TEST_ENV_HELPER_VAR1 >actual &&
	cat >expect <<-EOF &&
		false
	EOF
	test_cmp expect actual
'

test_expect_success 'env default to boolean (off)' '
	test_must_fail test-tool env-helper --type bool --default off \
		TEST_ENV_HELPER_VAR1 >actual &&
	cat >expect <<-EOF &&
		false
	EOF
	test_cmp expect actual
'

test_expect_success 'env default to boolean (0)' '
	test_must_fail test-tool env-helper --type bool --default 0 \
		TEST_ENV_HELPER_VAR1 >actual &&
	cat >expect <<-EOF &&
		false
	EOF
	test_cmp expect actual
'

test_expect_success 'env with wrong boolean default' '
	test_must_fail test-tool env-helper --type bool \
		--default 100 \
		TEST_ENV_HELPER_VAR1 >actual 2>&1 &&
	cat >expect <<-EOF &&
		ERROR: bad boolean environment value ${SQ}100${SQ} for ${SQ}TEST_ENV_HELPER_VAR1${SQ}
	EOF
	test_cmp expect actual
'

test_expect_success 'ulong env not set as 0' '
	test_must_fail test-tool env-helper --type ulong \
		TEST_ENV_HELPER_VAR1 >actual &&
	cat >expect <<-EOF &&
		0
	EOF
	test_cmp expect actual
'

test_expect_success 'ulong env: 255' '
	env TEST_ENV_HELPER_VAR1=255 \
		test-tool env-helper --type ulong \
		TEST_ENV_HELPER_VAR1 >actual &&
	cat >expect <<-EOF &&
		255
	EOF
	test_cmp expect actual
'

test_expect_success 'ulong env: 1k' '
	env TEST_ENV_HELPER_VAR1=1k \
		test-tool env-helper --type ulong \
		TEST_ENV_HELPER_VAR1 >actual &&
	cat >expect <<-EOF &&
		1024
	EOF
	test_cmp expect actual
'

test_expect_success 'ulong env: 1m' '
	test-tool env-helper --type ulong --default=1m \
		TEST_ENV_HELPER_VAR1 >actual &&
	cat >expect <<-EOF &&
		1048576
	EOF
	test_cmp expect actual
'

test_expect_success 'ulong env: 1g' '
	env TEST_ENV_HELPER_VAR1=1g test-tool env-helper \
		--type ulong --default=1k \
		TEST_ENV_HELPER_VAR1 >actual &&
	cat >expect <<-EOF &&
		1073741824
	EOF
	test_cmp expect actual
'

test_expect_success 'ulong env: bad-number' '
	test_must_fail env TEST_ENV_HELPER_VAR1=100-bad-number \
		test-tool env-helper --type ulong --default=1k \
		TEST_ENV_HELPER_VAR1 >actual 2>&1 &&
	cat >expect <<-EOF &&
		ERROR: failed to parse ulong number ${SQ}100-bad-number${SQ}
	EOF
	test_cmp expect actual
'

test_done
