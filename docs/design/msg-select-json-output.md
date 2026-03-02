# Design: msg-select --json Output Format

This document describes the `--json` option for `git-po-helper msg-select` and bidirectional conversion between PO and a defined JSON format: the command can accept either a PO file or a JSON file (schema below) as input; with `--json` it writes JSON, without it writes PO.

**Scope**: CLI flag, JSON schema (input and output), and PO entry ↔ JSON conversion. Implementation will reuse `ParsePoEntries` and the existing range logic for PO input; add JSON parsing and PO generation for JSON input and for PO output when input is JSON.

**Naming (implementation)**: The JSON schema is implemented in Go as **GettextJSON** (top-level struct with `header_comment`, `header_meta`, `entries`) and **GettextEntry** (each element of `entries`). The implementation lives in **util/gettext_json.go** (and its tests in util/gettext_json_test.go). The CLI subcommand remains `msg-select`.

---

## GETTEXT JSON File Format (Reference)

The **GETTEXT JSON** format is a JSON representation of PO/POT content. It is used by:

- **msg-select** (`--json`): output JSON; input can be PO or JSON
- **msg-cat** (`--json`): merge and output JSON; input can be PO, POT, or JSON
- **stat**: report statistics for PO or JSON files
- **agent-run translate** (local orchestration): batch JSON files for translation

**Format detection**: A file is treated as GETTEXT JSON if it starts with `{` after leading whitespace. Otherwise it is treated as PO/POT.

### Top-Level Structure

```json
{
  "header_comment": "string",
  "header_meta": "string",
  "entries": [ /* array of entry objects */ ]
}
```

| Field            | Type   | Description |
|------------------|--------|--------------|
| `header_comment` | string | Lines above the first `msgid ""` (comments, glossary). Joined with `\n`. Empty if none. |
| `header_meta`    | string | Decoded `msgstr` of the header entry (Project-Id-Version, Content-Type, Plural-Forms, etc.). Multi-line with embedded newlines. |
| `entries`        | array  | List of PO entries. Order matches source. |

### Entry Object Structure

Each element of `entries`:

| Field            | Type     | Required | Description |
|------------------|----------|----------|-------------|
| `msgid`          | string   | yes      | Singular message ID. PO escapes decoded. |
| `msgstr`         | string   | yes      | Singular message string. Empty for plural entries. |
| `msgid_plural`   | string   | no       | Plural form of msgid. Omit for non-plural. |
| `msgstr_plural`  | []string | no       | Array of msgstr[0], msgstr[1], … Omit for non-plural. |
| `comments`       | []string | no       | Comment lines (`#`, `#.`, `#:`, `#,`, etc.). Each element one line. |
| `fuzzy`          | bool     | yes      | True if entry has fuzzy flag. |
| `obsolete`       | bool     | no       | True for `#~` obsolete entries. Omitted if false. |
| `msgid_previous` | string   | no       | For `#~|` format (gettext 0.19.8+). Omitted if empty. |

**Example (obsolete entry with msgid_previous):**

```json
{
  "msgid": "Old string",
  "msgstr": "旧字符串",
  "msgid_previous": "Previous untranslated",
  "comments": [],
  "fuzzy": false,
  "obsolete": true
}
```

**Implementation**: `GettextJSON` and `GettextEntry` in **util/gettext_json.go**.

---

## 1. Goal

- **Input**: Support both PO files and JSON files as input. The JSON input format is the same as the JSON output format (see §2 and §3 below).
- **Output**: When `--json` is set, write a single JSON object (header_comment, header_meta, entries). When `--json` is not set, write PO text. Thus:
  - PO input, no `--json` → PO output (current behavior).
  - PO input, `--json` → JSON output.
  - JSON input, no `--json` → PO output (convert JSON to PO).
  - JSON input, `--json` → JSON output (e.g. after applying range selection to the parsed JSON).
- **Bidirectional conversion**: Support converting PO entries to JSON and JSON entries back to PO so that round-trip (PO → JSON → PO or JSON → PO → JSON) preserves content. Range selection applies to the logical entry list in both cases.

---

## 2. JSON Schema (Top-Level)

See the **GETTEXT JSON File Format** section above for a quick reference. The root object has three main fields:

| Field            | Type   | Description |
|------------------|--------|-------------|
| `header_comment` | string | All lines that appear above the header entry (before the first `msgid ""`). Typically comment lines (`# ...`), including optional terminology/glossary. Lines are joined with `\n`. Empty if there are no such lines. |
| `header_meta`    | string | The decoded content of the header entry’s `msgstr` (the metadata block: Project-Id-Version, Content-Type, Plural-Forms, etc.). Multi-line values are preserved with embedded newlines. Empty string if the header entry has no msgstr content. |
| `entries`        | array  | List of selected PO entries (see §3). Order matches the order of entries in the PO file (and the range spec). |

**Background: PO entry syntax (single-line vs multi-line, special characters)**

- **Single-line entries**: One line each for msgid and msgstr, e.g. `msgid "commit message"` and `msgstr "提交说明"`.
- **Multi-line entries**: The first line of `msgid` and `msgstr` is the empty string; following lines are quoted strings. Line breaks inside the content use the escape sequence `\n` in the PO source. Example:
  ```po
  msgid ""
  "Line 1\n"
  "Line 2"
  msgstr ""
  "行 1\n"
  "行 2"
  ```
  **CRITICAL** for multi-line: first line is `msgid ""` / `msgstr ""`; following lines are quoted strings; use `\n` for line breaks. Preserve quotes and structure exactly.
- **Preserving special characters**: In the PO file, escape sequences are literal backslash followed by a letter: `\n` (newline), `\t` (tab), `\"` (quote), `\\` (backslash). Placeholders like `%s`, `%d` and quotes must be preserved. Correct in PO: `msgstr "行 1\n行 2"` (keep `\n` as the two-character escape). Wrong: `"行 1\\n行 2"` (double backslash) or actual line breaks inside the quoted string.

**JSON output (this design)**: In JSON output, all continuation lines of a multi-line PO value are **merged into a single string**. PO escape sequences (`\n`, `\t`, `\"`, `\\`) are **decoded** into real characters (newline, tab, quote, backslash) so that the JSON holds the logical string value; when merging back into a PO file, the implementation must re-apply PO escaping.

### Example 1: Minimal (single-line entry, with header comment)

**PO source (excerpt):**

```po
# Glossary:
# term1	Translation 1
#
msgid ""
msgstr ""
"Project-Id-Version: git\n"
"Content-Type: text/plain; charset=UTF-8\n"

#. Comment for translator
#: src/file.c:10
msgid "Hello"
msgstr "你好"
```

**JSON output (corresponding):**

```json
{
  "header_comment": "# Glossary:\n# term1\tTranslation 1\n#\n",
  "header_meta": "Project-Id-Version: git\nContent-Type: text/plain; charset=UTF-8\n",
  "entries": [
    {
      "msgid": "Hello",
      "msgstr": "你好",
      "comments": ["#. Comment for translator\n", "#: src/file.c:10\n"],
      "fuzzy": false
    }
  ]
}
```

Note: In PO the glossary line has a literal tab between "term1" and "Translation 1"; in JSON it is stored as the tab character (`\t` in the table above is for readability; the actual JSON string contains one ASCII tab). Header metadata in PO uses `\n` inside quoted strings; in JSON `header_meta` contains real newlines so the string can be merged back into PO by re-escaping.

### Example 2: Multi-line msgid and msgstr (merged, escapes decoded)

**PO source (excerpt):**

```po
msgid ""
msgstr ""
"Project-Id-Version: git\n"
"Content-Type: text/plain; charset=UTF-8\n"

#, c-format
msgid ""
"Line one\n"
"Line two\twith tab, "
"padding for line 2."
msgstr ""
"第一行\n"
"第二行\t带制表符, 第二行的填充。"
```

**JSON output (corresponding):**

```json
{
  "header_comment": "",
  "header_meta": "Project-Id-Version: git\nContent-Type: text/plain; charset=UTF-8\n",
  "entries": [
    {
      "msgid": "Line one\nLine two\twith tab, padding for line 2.",
      "msgstr": "第一行\n第二行\t带制表符, 第二行的填充。",
      "comments": ["#, c-format\n"],
      "fuzzy": false
    }
  ]
}
```

Here the multi-line PO form (first line empty, then continuation lines) is merged into a single JSON string. The PO escape sequences `\n` and `\t` in the source become real newline and tab characters in the JSON string, so that round-trip or merge back to PO can re-escape them correctly. **This case is well-suited for validating PO ↔ JSON round-trip**: it exercises multiple continuation lines, embedded `\n` and `\t`, and comma/period in the middle of a line, and ensures that special characters in msgid and msgstr are not lost when converting between the two formats.

### Example 3: Plural form (msgid_plural, msgstr[0], msgstr[1])

**PO source (excerpt):**

```po
msgid ""
msgstr ""
"Project-Id-Version: git\n"
"Content-Type: text/plain; charset=UTF-8\n"
"Plural-Forms: nplurals=2; plural=(n != 1);\n"

#, c-format
msgid "One file"
msgid_plural "%d files"
msgstr[0] "一个文件"
msgstr[1] "%d 个文件"
```

**JSON output (corresponding):**

```json
{
  "header_comment": "",
  "header_meta": "Project-Id-Version: git\nContent-Type: text/plain; charset=UTF-8\nPlural-Forms: nplurals=2; plural=(n != 1);\n",
  "entries": [
    {
      "msgid": "One file",
      "msgstr": "",
      "msgid_plural": "%d files",
      "msgstr_plural": ["一个文件", "%d 个文件"],
      "comments": ["#, c-format\n"],
      "fuzzy": false
    }
  ]
}
```

For plural entries, the singular `msgstr` is empty; `msgid_plural` and `msgstr_plural` (array of msgstr[0], msgstr[1], …) carry the plural form.

---

## 3. Entry Object Schema (Detailed)

Each element of `entries` is an object with the following fields. See the **GETTEXT JSON File Format** section above for a quick reference.

| Field           | Type     | Required | Description |
|-----------------|----------|----------|-------------|
| `msgid`         | string   | yes      | Singular message ID: multi-line PO form merged into one string; PO escapes (`\n`, `\t`, `\"`, `\\`) decoded. |
| `msgstr`        | string   | yes      | Singular message string (same merge and decode rules). For plural entries this is often empty. |
| `msgid_plural`  | string   | no       | Present only when the entry is plural. Plural form of msgid (merged and decoded). Omit or empty string for non-plural entries. |
| `msgstr_plural` | []string | no       | Present only when the entry is plural. Array of merged and decoded msgstr[0], msgstr[1], … in order. Omit or empty array for non-plural entries. |
| `comments`      | []string | no       | All comment lines for this entry, in order: `#`, `#.`, `#:`, `#|`, `#,`, etc. Each element is one line including the newline (or without, implementation may normalize). Empty array if no comments. |
| `fuzzy`         | bool     | yes      | True if the entry has the fuzzy flag (from a `#, fuzzy` or `#, ..., fuzzy, ...` line). |
| `obsolete`      | bool     | no       | True for `#~` obsolete entries. Omitted if false. |
| `msgid_previous`| string   | no       | For `#~|` format (gettext 0.19.8+). Previous untranslated string. Omitted if empty. |

**Example (plural entry with fuzzy):**

```json
{
  "msgid": "One file",
  "msgstr": "",
  "msgid_plural": "%d files",
  "msgstr_plural": ["一个文件", "%d 个文件"],
  "comments": ["#, fuzzy\n", "#. Plural form\n"],
  "fuzzy": true
}
```

---

## 4. Derivation from Current Code

- **Header**: `ParsePoEntries` returns `header []string` (all header lines including comments, `msgid ""`, `msgstr "..."`, and continuation lines). The implementation will split these into:
  - **header_comment**: Lines from the start up to (but not including) the line that is `msgid ""` (after trimming). Join with `\n`.
  - **header_meta**: From the first `msgid ""` line, collect through the end of the header block; parse the `msgstr` value (including continuation lines), unescape and decode to a single string.

- **Entries**: The same `entries []*PoEntry` and range logic as today. For each selected entry, map:
  - `msgid` ← `PoEntry.MsgID`
  - `msgstr` ← `PoEntry.MsgStr`
  - `msgid_plural` ← `PoEntry.MsgIDPlural` (omit if empty)
  - `msgstr_plural` ← `PoEntry.MsgStrPlural` (omit if nil or empty)
  - `comments` ← `PoEntry.Comments`
  - `fuzzy` ← `PoEntry.IsFuzzy`
  - `obsolete` ← `PoEntry.IsObsolete` (omit if false)
  - `msgid_previous` ← `PoEntry.MsgIDPrevious` (omit if empty)

No change to `ParsePoEntries` or `ParseEntryRange` is required; only a new code path that builds the above JSON from `header` and the selected `entries`.

---

## 5. CLI and Behavior

- **Flag**: `--json`
- **Type**: boolean (presence = true).
- **Interaction**:
  - When `--json` is set, the command writes a single JSON object to the output (stdout or the file given by `-o`). No PO text is written.
  - When `--json` is not set, behavior is unchanged: PO text is written as today.
- **Range and --no-header**: Same as today. Omit `--range` to select all entries. When `--no-header` is set and `--json` is used, the output object should still have `header_comment` and `header_meta` (they may be empty if the header was skipped from selection, but the parser still sees the file header once). Rationale: the JSON is a representation of the *selected* content; if we omit the header from selection, we can either (a) still include header_comment/header_meta from the file for context, or (b) set them to empty. Option (a) is more useful for consumers that want to see file metadata. **Recommendation**: When `--json` is set, always parse and output the file header (header_comment, header_meta) from the source file, regardless of `--no-header`. The `entries` array contains only the entries that match the range; when `--no-header` is true, the range does not include entry 0, so we simply don’t add a “header entry” to `entries` (we never do—entry 0 is the header and is not in `entries`). So `--no-header` only affects the non-JSON text output (whether we write the header block). For JSON, we always include header_comment and header_meta from the file.
- **Empty selection**: If the range results in no content entries (e.g. range 10-20 but file has 5 entries), output a valid JSON object with `header_comment` and `header_meta` as parsed from the file, and `entries: []`.

---

## 6. Implementation Notes

1. **Location**: Add `--json` in `cmd/msg_select.go`; implement gettext JSON support in **util/gettext_json.go**: define **GettextJSON** and **GettextEntry** structs, and functions that take parsed `header` and selected `entries` and build/write the JSON (or convert JSON to PO). Use `encoding/json` with struct tags for the field names above.
2. **Header splitting**: Implement a helper that walks `header` lines to find the first `msgid ""` (after trim), then splits into comment part and the rest; then parse the msgstr value from the rest (lines starting with `msgstr ` or `"` continuation).
3. **Encoding**: JSON strings must escape newlines and quotes; `encoding/json` does this automatically. Ensure PO escape sequences in msgstr (e.g. `\n`, `\"`) are decoded before putting into the struct (they already are in `PoEntry` from `ParsePoEntries`).
4. **Omit empty**: Use `omitempty` for `msgid_plural` and `msgstr_plural` so they are omitted when not plural.

---

## 7. Summary

| Item              | Decision |
|-------------------|----------|
| Top-level keys    | `header_comment`, `header_meta`, `entries` |
| header_comment    | String; lines above first `msgid ""`, joined with `\n` |
| header_meta       | String; decoded msgstr of the header entry |
| entries[]         | Array of objects: msgid, msgstr, optional msgid_plural/msgstr_plural, comments[], fuzzy, optional obsolete, msgid_previous |
| --json            | Boolean flag; output JSON instead of PO text |
| --no-header (JSON)| Still output file header in JSON; only `entries` is range-driven |
| Commands using    | msg-select, msg-cat, stat, agent-run translate |

---

## 8. Implementation Steps

Implement in the following order. Each step should be testable before moving on; prefer small commits.

### 8.1 Step 1: Header splitting and JSON structs (util)

- **Tasks**:
  - In **util/gettext_json.go**, define **GettextJSON** (top-level) and **GettextEntry** (per-entry) structs for the JSON schema (top-level: `header_comment`, `header_meta`, `entries`; entry: `msgid`, `msgstr`, `msgid_plural`, `msgstr_plural`, `comments`, `fuzzy`) with `encoding/json` tags and `omitempty` where specified.
  - Implement a helper that, given `header []string` (from `ParsePoEntries`), returns `(headerComment string, headerMeta string, err error)`: split at the first line that is `msgid ""` (after trim); lines before that joined with `\n` → headerComment; from `msgstr ""` and its continuation lines in the rest, parse and decode the msgstr value → headerMeta.
  - Add a function that builds the top-level JSON object from `headerComment`, `headerMeta`, and selected `[]*PoEntry` (range already applied), and encodes it to an `io.Writer`.
- **Tests**: In **util/gettext_json_test.go**, unit test for header splitting (no comment; comment only; comment + header block; multi-line header_meta). Unit test that builds JSON from a small slice of `PoEntry` and parses it back, comparing key fields.
- **Commit**: e.g. `feat(util): add gettext JSON structs and header split for PO→JSON`

### 8.2 Step 2: PO input + --json → JSON output (cmd + util)

- **Tasks**:
  - In `cmd/msg_select.go`, add `--json` flag (bool). When set, after parsing PO and resolving the range, call the new JSON builder and write to the same output (stdout or `-o` file) instead of writing PO text.
  - Ensure `--no-header` does not strip header from JSON (always include header_comment and header_meta from the parsed file).
  - Handle empty selection: output `{"header_comment":"...","header_meta":"...","entries":[]}`.
- **Tests**: Integration or CLI test: run `msg-select --range "1" --json` on a known PO file, capture stdout, decode JSON and assert structure and one entry’s msgid/msgstr. Test empty range (e.g. `--range "99-100"` on a 5-entry file) yields valid JSON with empty `entries`.
- **Commit**: e.g. `feat(msg-select): add --json to output JSON for PO input`

### 8.3 Step 3: JSON → PO conversion (util)

- **Tasks**:
  - In **util/gettext_json.go**, implement a function that, given the decoded **GettextJSON** (or `header_comment`, `header_meta`, `entries`), produces valid PO content: write header_comment as raw lines (split on `\n`), then `msgid ""` and `msgstr ""` with header_meta encoded (PO escaping: newline→`\n`, tab→`\t`, etc.), then for each entry write comments, msgid (single- or multi-line with PO escaping), msgstr / msgid_plural / msgstr[n] as appropriate, and `#, fuzzy` if needed.
  - Ensure multi-line logic: if string contains newline, output first line `msgid ""`/`msgstr ""` and continuation lines as quoted, PO-escaped strings.
- **Tests**: In **util/gettext_json_test.go**, unit test: take the Example 2 JSON (multi-line, `\n` and `\t`) from this doc, convert to PO, then parse the PO with `ParsePoEntries` and convert back to JSON; assert round-trip equality of msgid and msgstr. Test plural entry round-trip (Example 3). Test that special characters (`\n`, `\t`, `\"`, `\\`) in msgid/msgstr are preserved.
- **Commit**: e.g. `feat(util): add gettext JSON to PO conversion`

### 8.4 Step 4: JSON input detection and JSON → PO output (cmd)

- **Tasks**:
  - Detect input format: read the first non-whitespace bytes of the input file; if it starts with `{`, treat as JSON; otherwise treat as PO.
  - When input is JSON: parse the file into **GettextJSON**; apply range selection to `entries` (indices 1..N); if `--json` is set, write the selected subset as JSON; if `--json` is not set, call the gettext JSON→PO conversion and write PO text.
  - When input is PO, keep current behavior (Step 2).
- **Tests**: CLI test: create a JSON file (e.g. Example 2 or 3 from this doc), run `msg-select --range "1" <json-file>` (no `--json`), assert stdout is valid PO and contains the expected msgid/msgstr. Run with `--json` and assert JSON output. Test range on JSON (e.g. two entries, range "1", then range "2", then "1-2").
- **Commit**: e.g. `feat(msg-select): support gettext JSON input and output PO when --json is omitted`

### 8.5 Step 5: Tests and docs

- **Tests** (add or extend):
  - **Round-trip (Example 2)**: PO file with multi-line msgid/msgstr and `\n`/`\t` → JSON → PO → parse PO → JSON again; compare msgid and msgstr strings to original. This guards against special-character loss.
  - **Round-trip (Example 3)**: Plural entry PO → JSON → PO → parse → JSON; compare.
  - **Edge cases**: Empty entries list; header only (range selects nothing); entry with only comments then msgid/msgstr.
- **Docs**: Update `cmd/msg_select.go` Long description to mention `--json` and that input can be PO or JSON; document gettext JSON schema or refer to this design doc (implemented as GettextJSON/GettextEntry in util/gettext_json.go).
- **Commit**: e.g. `test(msg-select): add PO↔gettext JSON round-trip tests; doc --json and JSON input`

### 8.6 Summary of commits (suggested)

| Order | Commit scope | Main change |
|-------|--------------|-------------|
| 1 | util | JSON structs, header split, PO→JSON build |
| 2 | cmd + util | --json flag, PO input → JSON output |
| 3 | util | JSON→PO conversion (escaping, multi-line) |
| 4 | cmd | JSON input detection, JSON input → PO output (and → JSON with --json) |
| 5 | test + docs | Round-trip tests (Examples 2 & 3), CLI help |

After approval, implementation can proceed in `cmd/msg_select.go` and `util/` as outlined above.

---

## 9. Real-world round-trip example (`zh_CN`)

A sample from `po/zh_CN.po` (e.g. from git-l10n/git-po) is provided in
**test/fixtures/zh_CN_example.po**. The integration test
**test/t0120-msg-select-json-roundtrip.sh** verifies:

1. **PO → JSON**: `msg-select --range "1-" --json` on the fixture → `sample.json`
2. **JSON → PO**: `msg-select --range "1-"` on `sample.json` (no `--json`) → `roundtrip.po`
3. **Normalize**: Run `msgcat` on both the original fixture and `roundtrip.po`
4. **Compare**: The two formatted PO files must be identical (`test_cmp`)

This ensures that round-trip (PO → gettext JSON → PO) preserves content; `msgcat` normalization accounts for minor formatting differences (e.g. line wrapping) so that only semantic differences would cause a diff.
