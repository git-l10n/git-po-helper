Review the translations in `{{.source}}` and write a JSON report of any issues
found to `{{.dest}}`, using the format specified in the "Review result JSON
format" section below. When reviewing, follow these guidelines:

- Use "Background knowledge for localization workflows" for the format of the
  source JSON file, placeholders, and terminology.
- If `header_comment` includes a glossary, follow it for consistency.
- Do **not** review the header (`header_comment`, `header_meta`).
- For every other entry, check the entry's `msgstr` **array** (translation
  forms) against `msgid` / `msgid_plural` using the "Quality checklist" below.
- Write JSON per "Review result JSON format" below; use `{"issues": []}` when
  there are no issues. **Always** write `{{.dest}}`—it marks the
  batch complete.

**Review result JSON format**:

The **Review result JSON** format defines the structure for translation
review reports. For each entry with translation issues, create an issue
object as follows:

- Copy the original entry's `msgid`, optional `msgid_plural`, and optional
  `msgstr` array (original translation forms) into the issue object. Use the
  same shape as GETTEXT JSON: `msgstr` is **always a JSON array** when present
  (one element singular, multiple for plural).
- Write a summary of all issues found for this entry in `description`.
- Set `score` according to the severity of issues found for this entry,
  from 0 to 3 (0 = critical; 1 = major; 2 = minor; 3 = perfect, no issues).
  **Lower score means more severe issues.**
- Place the suggested translation in **`suggest_msgstr`** as a **JSON array**:
  one string for singular, multiple strings for plural forms in order. This is
  required for `git-po-helper` to apply suggestions.
- Include only entries with issues (score less than 3). When no issues are
  found in the batch, write `{"issues": []}`.

Example review result (with issues):

```json
{
  "issues": [
    {
      "msgid": "commit",
      "msgstr": ["委托"],
      "score": 0,
      "description": "Terminology error: 'commit' should be translated as '提交'",
      "suggest_msgstr": ["提交"]
    },
    {
      "msgid": "repository",
      "msgid_plural": "repositories",
      "msgstr": ["版本库", "版本库"],
      "score": 2,
      "description": "Consistency issue: suggest using '仓库' consistently",
      "suggest_msgstr": ["仓库", "仓库"]
    }
  ]
}
```

Field descriptions for each issue object (element of the `issues` array):

- `msgid` (and optional `msgid_plural` for plural entries): Original source text.
- `msgstr` (optional): JSON array of original translation forms (same meaning as
  in GETTEXT JSON entries).
- `suggest_msgstr`: JSON array of suggested translation forms; **must be an
  array** (e.g. `["提交"]` for singular). Plural entries use multiple elements
  in order.
- `score`: 0–3 (0 = critical; 1 = major; 2 = minor; 3 = perfect, no issues).
- `description`: Brief summary of the issue.


## Background Knowledge for Translators and Reviewers

This section provides essential background knowledge about PO file structure
and format that is required for both translation and review tasks. Understanding
these concepts is fundamental before performing any translation or review
operations on `po/XX.po` files.


### Header Entry

The **header entry** is the first entry in every `po/XX.po`. It has an empty
`msgid`; translation metadata (project, language, plural rules, encoding, etc.)
is stored in `msgstr`, as in this example:

```po
msgid ""
msgstr ""
"Project-Id-Version: Git\n"
"Language: zh_CN\n"
"MIME-Version: 1.0\n"
"Content-Type: text/plain; charset=UTF-8\n"
"Content-Transfer-Encoding: 8bit\n"
"Plural-Forms: nplurals=2; plural=(n != 1);\n"
```

**CRITICAL**: Do not edit the header's `msgstr` while translating. It holds
metadata only and must be left unchanged.


### Glossary Section

PO files may have a glossary in comments before the header entry (first
`msgid ""`), giving terminology guidelines (e.g.):

```po
# Git glossary for Chinese translators
#
#   English                          |  Chinese
#   ---------------------------------+--------------------------------------
#   3-way merge                      |  三路合并
#   branch                           |  分支
#   ...
```

**IMPORTANT**: Read and use the glossary when translating or reviewing. It is
in `#` comments only. Leave that comment block unchanged.


### PO entry structure (single-line and multi-line)

PO entries are `msgid` / `msgstr` pairs. Plural messages add `msgid_plural` and
`msgstr[n]`. The `msgid` is the immutable source; `msgstr` is the target
translation. Each side may be a single quoted string or a multi-line block.
In the multi-line form the header line is often `msgid ""` / `msgstr ""`, with
the real text split across following quoted lines (concatenated by Gettext).

**Single-line entries**:

```po
msgid "commit message"
msgstr "提交说明"
```

**Multi-line entries**:

```po
msgid ""
"Line 1\n"
"Line 2"
msgstr ""
"行 1\n"
"行 2"
```

**CRITICAL**: Do **not** use `grep '^msgstr ""'` to find untranslated entries;
multi-line `msgstr` blocks use the same opening line, so grep gives false
positives. Use `msgattrib` (next section).


### Translating fuzzy entries

Fuzzy entries need re-translation because the source text changed. The format
differs by file type:

- **PO file**: A `#, fuzzy` tag in the entry comments marks the entry as fuzzy.
- **JSON file**: The entry has `"fuzzy": true`.

**Translation principles**: Re-translate the `msgstr` (and, for plural entries,
`msgstr[n]`) into the target language. Do **not** modify `msgid` or
`msgid_plural`. After translation, **clear the fuzzy mark**: in PO, remove the
`#, fuzzy` tag from comments; in JSON, omit or set `fuzzy` to `false`.


### Preserving Special Characters

Preserve escape sequences (`\n`, `\"`, `\\`, `\t`), placeholders (`%s`, `%d`,
etc.), and quotes exactly as in `msgid`. Only reorder placeholders with
positional syntax when needed (see Placeholder Reordering below).


### Placeholder Reordering

When reordering placeholders relative to `msgid`, use positional syntax (`%n$`)
where *n* is the 1-based argument index, so each argument still binds to the
right value. Preserve width and precision modifiers, and place `%n$` before
them (see examples below).

**Example 1** (precision):

```po
#, c-format
msgid "missing environment variable '%s' for configuration '%.*s'"
msgstr "配置 '%3$.*2$s' 缺少环境变量 '%1$s'"
```

`%s` → argument 1 → `%1$s`. `%.*s` needs precision (arg 2) and string (arg 3) →
`%3$.*2$s`.

**Example 2** (multi-line, four `%s` reordered):

```po
#, c-format
msgid ""
"the 'submodule.%s.gitdir' config does not exist for module '%s'. Please "
"ensure it is set, for example by running something like: 'git config "
"submodule.%s.gitdir .git/modules/%s'. For details see the "
"extensions.submodulePathConfig documentation."
msgstr ""
"模块 '%2$s' 的 'submodule.%1$s.gitdir' 配置不存在。请确保已设置，例如运行类"
"似：'git config submodule.%3$s.gitdir .git/modules/%4$s'。详细信息请参见 "
"extensions.submodulePathConfig 文档。"
```

Original order 1,2,3,4; in translation 2,1,3,4. Each line must be a complete
quoted string.


### GETTEXT JSON format

The **GETTEXT JSON** format is an internal format defined by `git-po-helper`
for convenient batch processing of translation and related tasks by AI models.
`git-po-helper msg-select`, `git-po-helper msg-cat`, and `git-po-helper compare`
read and write this format.

**Top-level structure**:

```json
{
  "header_comment": "string",
  "header_meta": "string",
  "entries": [ /* array of entry objects */ ]
}
```

| Field            | Description                                                                    |
|------------------|--------------------------------------------------------------------------------|
| `header_comment` | Lines above the first `msgid ""` (comments, glossary), directly concatenated.  |
| `header_meta`    | Encoded `msgstr` of the header entry (Project-Id-Version, Plural-Forms, etc.). |
| `entries`        | List of PO entries. Order matches source.                                      |

**Entry object** (each element of `entries`):

| Field           | Type     | Description                                                  |
|-----------------|----------|--------------------------------------------------------------|
| `msgid`         | string   | Singular message ID. PO escapes encoded (e.g. `\n` → `\\n`). |
| `msgstr`        | []string | Translation forms as a **JSON array only**. Details below.   |
| `msgid_plural`  | string   | Plural form of msgid. Omit for non-plural.                   |
| `comments`      | []string | Comment lines (`#`, `#.`, `#:`, `#,`, etc.).                 |
| `fuzzy`         | bool     | True if entry has fuzzy flag.                                |
| `obsolete`      | bool     | True for `#~` obsolete entries. Omit if false.               |

**`msgstr` array (required shape)**:

- **Always** a JSON array of strings, never a single string. One element = singular
  (PO `msgstr` / `msgstr[0]`); multiple elements = plural forms in order
  (`msgstr[0]`, `msgstr[1]`, …).
- Omit the key or use an empty array when the entry is untranslated.

**Example (single-line entry)**:

```json
{
  "header_comment": "# Glossary:\\n# term1\\tTranslation 1\\n#\\n",
  "header_meta": "Project-Id-Version: git\\nContent-Type: text/plain; charset=UTF-8\\n",
  "entries": [
    {
      "msgid": "Hello",
      "msgstr": ["你好"],
      "comments": ["#. Comment for translator\\n", "#: src/file.c:10\\n"],
      "fuzzy": false
    }
  ]
}
```

**Example (plural entry)**:

```json
{
  "msgid": "One file",
  "msgid_plural": "%d files",
  "msgstr": ["一个文件", "%d 个文件"],
  "comments": ["#, c-format\\n"]
}
```

**Example (fuzzy entry before translation)**:

```json
{
  "msgid": "Old message",
  "msgstr": ["旧翻译。"],
  "comments": ["#, fuzzy\\n"],
  "fuzzy": true
}
```

**Translation notes for GETTEXT JSON files**:

- **Preserve structure**: Keep `header_comment`, `header_meta`, `msgid`,
  `msgid_plural` unchanged.
- **Fuzzy entries**: Entries extracted from fuzzy PO entries have `"fuzzy": true`.
  After translating, **remove the `fuzzy` field** or set it to `false` in the
  output JSON. The merge step uses `--unset-fuzzy`, which can also remove the
  `fuzzy` field.
- **Placeholders**: Preserve `%s`, `%d`, etc. exactly; use `%n$` when
  reordering (see "Placeholder Reordering" above).


### Quality checklist

- **Accuracy**: Faithful to original meaning; no omissions or distortions.
- **Fuzzy entries**: Re-translate fully and clear the fuzzy flag (see
  "Translating fuzzy entries" above).
- **Terminology**: Consistent with glossary (see "Glossary Section" above) or
  domain standards.
- **Grammar and fluency**: Correct and natural in the target language.
- **Placeholders**: Preserve variables (`%s`, `{name}`, `$1`) exactly; use
  positional parameters when reordering (see "Placeholder Reordering" above).
- **Special characters**: Preserve escape sequences (`\n`, `\"`, `\\`, `\t`),
  placeholders exactly as in `msgid`. See "Preserving Special Characters" above.
- **Plurals and gender**: Correct forms and agreement.
- **Context fit**: Suitable for UI space, tone, and use (e.g. error vs. tooltip).
- **Cultural appropriateness**: No offensive or ambiguous content.
- **Consistency**: Match prior translations of the same source.
- **Technical integrity**: Do not translate code, paths, commands, brands, or
  proper nouns.
- **Readability**: Clear, concise, and user-friendly.
