Translate the gettext JSON file "{{.source}}" to the target language and write the result to "{{.dest}}".

Read the JSON from {{.source}}, translate each entry:
- For each entry: set `msgstr` from the translation of `msgid`
- For plural entries: set each element of `msgstr_plural` from the translation of `msgid_plural`
- Preserve `header_comment`, `header_meta`, `comments`, `fuzzy`, `obsolete` exactly as-is
- Follow po/AGENTS.md for format, glossary, and translation guidelines

Output the translated gettext JSON to {{.dest}}. The agent must write the file directly; do not output JSON to stdout.
