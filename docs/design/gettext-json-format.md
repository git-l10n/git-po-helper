# Gettext JSON Format

This document describes the **gettext JSON** format used by `git-po-helper` for PO entry selection (`msg-select --json`), merging (`msg-cat`), comparison, and agent workflows. The format is designed to have a straightforward correspondence with the [gettext PO file format](gettext-format.md) so that PO↔JSON conversion and round-trips are well-defined.

**Implementation:** `util/gettext_json.go`, `util/gettext.go` (GettextEntry).

---

## 1. Purpose and usage

- **Input/output for msg-select and msg-cat:** JSON can be read from or written to files; PO is parsed and can be emitted from JSON.
- **Comparison and merge:** `compare` and merge logic operate on GettextJSON; PO and JSON files are loaded into the same in-memory structure.
- **Agent workflows:** Translation and review prompts use gettext JSON as the batch format (entries, header_comment, header_meta).

**Round-trip:** PO → parse → GettextJSON → write PO should preserve content. JSON → parse → GettextJSON → write JSON should preserve content. When writing PO from JSON, string values are converted from JSON escaping to PO escaping (see String format below).

---

## 2. Top-level structure

The root object has three fields:

| Field             | Type   | Description |
|-------------------|--------|-------------|
| `header_comment`  | string | All lines before the first `msgid ""` (comments, glossary, etc.), concatenated with `\n`. Empty or omitted when there is no header comment block. |
| `header_meta`     | string | The decoded `msgstr` of the header entry (first entry with `msgid ""`). Typically contains Project-Id-Version, Content-Type, Plural-Forms, etc., with newlines as `\n`. |
| `entries`        | array  | Array of entry objects; order matches the PO file. Entry 0 in the PO is the header and is **not** included in `entries`; `entries` are content entries only. |

Example (in the JSON file, special characters are encoded: e.g. newline in the string value is stored as the two characters `\` and `n`, so it appears as `\\n` in JSON):

```json
{
  "header_comment": "# Git glossary\\n#   term | translation\\n",
  "header_meta": "Project-Id-Version: git\\nContent-Type: text/plain; charset=UTF-8\\n",
  "entries": [
    {
      "msgid": "commit",
      "msgstr": ["提交"],
      "comments": ["#: builtin/commit.c\\n"],
      "fuzzy": false
    }
  ]
}
```

**Correspondence with PO (gettext-format.md):**

- **Header entry (Section 7):** In PO, the first entry is `msgid ""` / `msgstr "..."`. In gettext JSON, that entry is not represented as an element of `entries`; its comment lines (above `msgid ""`) go into `header_comment`, and its `msgstr` value goes into `header_meta`.
- **Trailing lines (Section 7):** Lines after the last entry are not represented in this format.

---

## 3. Entry object (current implementation)

Each element of `entries` is an object with the following fields. All string values that represent PO content (msgid, msgstr, header_meta, comment lines) use **PO string format** when stored in memory after parsing: escape sequences such as newline and tab are stored as literal backslash + character (e.g. `\n`, `\t`). When the JSON file is **read**, the parser converts JSON escape sequences (e.g. real newline in JSON) into this PO format so that writing back to PO is correct.

| Field             | Type    | Required | Description |
|-------------------|---------|----------|-------------|
| `msgid`           | string  | yes      | Singular message id. PO format (e.g. `\n` for newline in string). |
| `msgstr`          | array   | no       | Translation form(s). **Always an array** in JSON: one element for singular, multiple for plural forms in order (`msgstr[0]`, `msgstr[1]`, …). Omit or empty when untranslated. Parsers accept a single string and normalize to one-element array. |
| `msgid_plural`    | string  | no       | Plural form of msgid. Omit for non-plural entries. |
| `comments`        | array   | no       | Comment lines in PO order. Each element is one line including the prefix (`# `, `#.`, `#:`, `#,`). Newlines: use PO format `\n` in content. Omit or empty array when no comments. |
| `fuzzy`           | boolean | no       | True if the entry has the `fuzzy` workflow flag (see gettext-format.md Section 3). Default false. When writing PO, `#, fuzzy` is emitted if true (and merged into `#,` line if comments contain flags). |
| `obsolete`        | boolean | no       | True for obsolete entries (`#~` / `#~|` in PO). Default false. When writing PO, keyword/string lines are prefixed with `#~ ` or `#~| ` (see gettext-format.md Section 9). |
| `msgid_previous`  | string  | no       | Previous untranslated string (PO form `#| msgid "..."` or obsolete `#~| msgid "..."`). Used for fuzzy/obsolete context. |

**Correspondence with PO:**

| PO (gettext-format.md)        | Gettext JSON field   |
|------------------------------|------------------------|
| Section 1: `msgid`           | `msgid`               |
| Section 1: `msgstr`          | `msgstr[0]`           |
| Section 2: `# ` translator   | `comments[]` line     |
| Section 2: `#.` extracted    | `comments[]` line     |
| Section 2: `#:` reference    | `comments[]` line     |
| Section 2: `#,` flags        | `comments[]` line; `fuzzy` true if `fuzzy` in flags |
| Section 2: `#| msgid` previous | `msgid_previous`    |
| Section 6: `msgid_plural`     | `msgid_plural`        |
| Section 6: `msgstr[n]`       | `msgstr[n]`           |
| Section 9: obsolete `#~` / `#~|` | `obsolete` true; `msgid_previous` for `#~| msgid` |

**Notes:**

- When building JSON from PO, the implementation strips the `fuzzy` flag from the `#,` line stored in `comments` and keeps it only in the `fuzzy` field, so that tools can clear fuzzy without editing comment strings.
- When writing PO from JSON for an **obsolete** entry, the implementation prefixes only the **keyword and string lines** (`msgid`, `msgid_plural`, `msgstr`, `msgstr[n]`, and `msgid_previous` as `#~| msgid`) with `#~ ` or `#~| `. Comment lines in `comments` are written as-is (without adding `#~ `). So for obsolete entries that originally had comment lines prefixed with `#~ #:`, `#~ #,`, etc., the current round-trip stores those lines in `comments` without the `#~ ` prefix; re-emitting PO will output them without `#~ `. See Extension below for full 1:1 obsolete handling.

---

## 4. String format (PO vs JSON)

- **In the JSON file:** String values use standard JSON escaping (e.g. `\n` for newline, `\t` for tab, `\"` for quote). This is how JSON is parsed and emitted.
- **In memory (GettextEntry, GettextJSON):** After parsing JSON, the code converts such strings to **PO format**: newline and tab are stored as the two-character sequences backslash + `n` and backslash + `t`, so that when writing PO, `poEscape` and multi-line rules produce correct PO syntax. When reading PO, the parser produces the same PO format for msgid/msgstr/comment lines.
- **Writing JSON:** When serializing GettextJSON to JSON, the in-memory PO-format strings are written as JSON strings, so backslash and quote are escaped for JSON; the literal `\n` in PO format becomes the two characters `\` and `n` in the JSON file (and may appear as `\\n` in JSON representation).

So: **PO format** = escape sequences as literal `\n`, `\t`, etc. **JSON file** = normal JSON escaping. Conversion happens at parse/serialize boundaries.

---

## 5. Plural entries

- Singular: `msgstr` has one element (or is empty).
- Plural: `msgid_plural` is set; `msgstr` has as many elements as the language’s plural forms (e.g. 2 for `msgstr[0]`, `msgstr[1]`). Order must match PO.

Same uniqueness and semantics as in gettext-format.md Section 6.

---

## 6. Empty and omitted fields

- **Empty file or no entries:** Implementations may output `{"header_comment":"","header_meta":"","entries":[]}` or an empty file, depending on tool (e.g. msg-select with no selected entries can write an empty file).
- **Omitted vs empty:** `omitempty` in the Go struct means optional in JSON. Absent `msgid_plural`, `comments`, `msgstr` (or empty array), `obsolete`, `msgid_previous` mean the same as empty or false as appropriate.

---

## 7. Extensions for full 1:1 correspondence with PO

To align completely with [gettext-format.md](gettext-format.md) (including context and obsolete comment lines), the following extensions can be added.

### 7.1 Context (msgctxt)

PO entries can have **msgctxt** (gettext-format.md Section 5). The implementation includes optional `msgctxt` and `msgctxt_previous` in GettextEntry (Phase 2).

**Extension:** Add an optional field to each entry:

| Field    | Type   | Description |
|----------|--------|-------------|
| `msgctxt` | string | Context specifier. Empty string and absent field are distinct (as in PO). |

When writing PO, emit `msgctxt context\n` before `msgid` when the field is present. For obsolete entries, that line would be emitted with the same `#~ ` prefix as other keyword lines.

### 7.2 Obsolete entry comment prefix

In PO, **all lines** of an obsolete entry start with `#`, including comment lines (e.g. `#~ #:`, `#~ #,` — see gettext-format.md Section 9).

**Implemented (Phase 4, Option A):** The parser stores comment lines of obsolete entries **without** the `#~ ` prefix (e.g. `#: file.c` in `comments`). When emitting PO from an entry (no RawLines), the writer prepends `#~ ` to each line in `comments` when `obsolete` is true, so output is `#~ #: file.c`. Round-trip via RawLines preserves the original lines unchanged.

### 7.3 Future PO format evolution

**`#=`** flag lines (gettext-format.md Section 8, June 2025) are supported: they are stored in `comments[]` and round-tripped like `#,` lines.

**Workflow vs sticky (2027+):** From 2027-01-01 the PO format may distinguish `#,` = workflow flags and `#=` = sticky flags (or the opposite). The current implementation keeps all flag lines in `entries[].comments[]`; a future extension may add separate fields (e.g. `workflow_flags`, `sticky_flags`) once the format is finalized, so that unknown sticky flags are preserved and not dropped.

---

## 8. Summary mapping: PO ↔ Gettext JSON

| PO format (gettext-format.md) | Gettext JSON (current) | Note |
|-------------------------------|------------------------|------|
| Header: lines before `msgid ""` | `header_comment`       | Joined with `\n`. |
| Header: `msgstr ""` value      | `header_meta`          | PO escapes; in JSON as string. |
| Entry: `msgid`                 | `entries[].msgid`      | |
| Entry: `msgstr`                | `entries[].msgstr[0]`  | |
| Entry: `msgid_plural`         | `entries[].msgid_plural` | |
| Entry: `msgstr[n]`            | `entries[].msgstr[n]`  | |
| Entry: `# `, `#.`, `#:`, `#,` | `entries[].comments[]` | One line per element; fuzzy can be in `fuzzy` instead of `#,` only. |
| Entry: `#| msgid` previous     | `entries[].msgid_previous` | |
| Entry: `#, fuzzy`              | `entries[].fuzzy`      | Also reflected in `comments` unless stripped. |
| Obsolete: `#~` / `#~|` lines   | `entries[].obsolete`, `msgid_previous`, `msgctxt_previous` | Comment lines in obsolete: stored without `#~ `; writer prepends `#~ ` (7.2). |
| Entry: `msgctxt`               | `entries[].msgctxt`, `msgctxt_previous` | Section 7.1. |

This document and the implementation in `util/gettext_json.go` / `util/gettext.go` are the reference for the gettext JSON schema and PO↔JSON behavior.
