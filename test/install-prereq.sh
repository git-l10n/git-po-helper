#!/bin/sh

set -e

run_as_root () {
	if test "$(id -u)" = 0
	then
		"$@"
	else
		sudo "$@"
	fi
}

if ! command -v xgettext >/dev/null
then
	if command -v apt-get >/dev/null
	then
		run_as_root apt-get update
		run_as_root apt-get install --assume-yes gettext
	elif command -v yum >/dev/null
	then
		run_as_root yum makecache
		run_as_root yum install -y gettext
	elif command -v brew >/dev/null
	then
		brew install gettext
		if test -n "${GITHUB_PATH:-}"
		then
			gettext_prefix=$(brew --prefix gettext)
			printf '%s\n' "$gettext_prefix/bin" >>"$GITHUB_PATH"
		fi
	else
		echo "Cannot install gettext: no supported package manager found" >&2
		exit 1
	fi
fi
