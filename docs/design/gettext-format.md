# Gettext PO File Format

This document summarizes the official GNU gettext PO file format. It provides background for implementing and maintaining PO file parsing (e.g. in `util/gettext.go`). The content is derived from the [GNU gettext manual](https://www.gnu.org/software/gettext/manual/html_node/PO-Files.html).

## References

- [PO File Entries (What an entry looks like)](https://www.gnu.org/software/gettext/manual/html_node/PO-File-Entries.html)
- [Workflow flags](https://www.gnu.org/software/gettext/manual/html_node/Workflow-flags.html)
- [Sticky flags](https://www.gnu.org/software/gettext/manual/html_node/Sticky-flags.html)
- [Entries with Context](https://www.gnu.org/software/gettext/manual/html_node/Entries-with-Context.html)
- [Entries with Plural Forms](https://www.gnu.org/software/gettext/manual/html_node/Entries-with-Plural-Forms.html)
- [Further details on the PO file format](https://www.gnu.org/software/gettext/manual/html_node/More-Details.html)
- [PO File Format Evolution](https://www.gnu.org/software/gettext/manual/html_node/PO-File-Format-Evolution.html)
- [Obsolete Entries](https://www.gnu.org/savannah-checkouts/gnu/gettext/manual/html_node/Obsolete-Entries.html) (Emacs PO mode; describes that all lines of an obsolete entry start with `#`)

---

## 1. What an entry looks like

A PO file consists of **entries**. Each entry relates an original (untranslated) string to its translation. All entries in a PO file usually belong to one project and one target language.

Schematic structure of one entry:

```
white-space
#  translator-comments
#. extracted-comments
#: reference...
#, flag...
#| msgid previous-untranslated-string
msgid untranslated-string
msgstr translated-string
```

Example:

```
#: lib/error.c:116
msgid "Unknown system error"
msgstr "Error desconegut del sistema"
```

- Entries are separated by optional white space; GNU gettext tools typically output **one blank line** between entries.
- The **msgid** string is the untranslated string as in the program sources; **msgstr** is the translation.
- Both strings use `"` delimiters and `\` escapes (C-style). Multi-line strings use multiple quoted lines that are concatenated (see below).

---

## 2. Comments

All comment lines start with `#` and extend to the end of the line. There are two kinds:

| Prefix | Kind | Description |
|--------|------|-------------|
| `#` + space | Translator comments | Created and maintained by the translator. |
| `#.` | Extracted comments | From the programmer; extracted by `xgettext` from source. |
| `#:` | Reference | References to source code: `file_name:line_number` or `file_name`. If the file name contains spaces, it may be enclosed in Unicode U+2068 and U+2069. |
| `#,` | Flags | Comma-separated list of flags; see Workflow flags and Sticky flags. Not ignored by tools. |
| `#\|` | Previous untranslated string | Inserted by `msgmerge` when marking a message fuzzy; shows the previous `msgid` before developer changes. |

Comments are optional. Automatic comments (non-space after `#`) are managed by GNU gettext tools and may be changed or removed by `msgmerge`.

---

## 3. Workflow flags

**Workflow flags** can be added or removed by the translator or by workflow tools. They describe the **state** of the entry.

Currently defined:

- **`fuzzy`** – The `msgstr` might not be a correct translation (anymore). Set by `msgmerge` when it merged `msgid` and `msgstr` via fuzzy matching, or by the translator. The translator removes it when the translation is acceptable. Used by `msgfmt` for diagnostics.

---

## 4. Sticky flags

**Sticky flags** are set when the PO template is created and normally stay for the whole workflow.

Two kinds:

1. **`*-format` flags** – Inferred from the `msgid` and surrounding source (e.g. `c-format`, `python-format`). Only `xgettext` should add them. They tell `msgfmt` which format checks to apply to the translation.
2. **Other flags** – e.g. **`no-wrap`** (inhibit line wrapping for this entry when emitting the PO file).

Common `*-format` flags include: `c-format`, `no-c-format`, `objc-format`, `c++-format`, `python-format`, `python-brace-format`, `java-format`, `java-printf-format`, `csharp-format`, `javascript-format`, `go-format`, `rust-format`, `sh-format`, `sh-printf-format`, `perl-format`, `php-format`, `qt-format`, `lua-format`, and many others. See the [Sticky flags](https://www.gnu.org/software/gettext/manual/html_node/Sticky-flags.html) section for the full list.

For **plural entries**, the **`range:`** sticky flag can appear: `range: minimum-value..maximum-value`, indicating the possible range of the numeric parameter.

---

## 5. Entries with context

Entries can have a **context** so that the same untranslated string can appear with different translations:

```
#| msgctxt previous-context
#| msgid previous-untranslated-string
msgctxt context
msgid untranslated-string
msgstr translated-string
```

- **msgctxt** disambiguates messages with the same `msgid`.
- An **empty context string** and **absent msgctxt** are not the same.
- Plural entries can also have `msgctxt` before `msgid`.

---

## 6. Entries with plural forms

For plurals, the entry uses `msgid`, `msgid_plural`, and indexed `msgstr[n]`:

```
msgid untranslated-string-singular
msgid_plural untranslated-string-plural
msgstr[0] translated-string-case-0
...
msgstr[N] translated-string-case-n
```

Example:

```
#: src/msgcmp.c:338 src/po-lex.c:699
#, c-format
msgid "found %d fatal error"
msgid_plural "found %d fatal errors"
msgstr[0] "s'ha trobat %d error fatal"
msgstr[1] "s'han trobat %d errors fatals"
```

`msgctxt` can be used before `msgid` in plural entries as well.

---

## 7. Further details (format rules)

### Header entry

- The **first entry** of the file must be the **header entry**: `msgid ""` followed by `msgstr "..."` containing meta information (Project-Id-Version, Content-Type, Plural-Forms, etc.).
- The empty `msgid` is **reserved** for this header and must not be used elsewhere.

### String syntax

- Untranslated and translated strings follow **C string syntax**: quotes and backslash escapes. **Universal character escapes** `\u` and `\U` are **not** allowed.
- **Multi-line strings**: do not use escaped newlines inside one quoted string. Use multiple lines, each with a closing `"` and the next line starting with `"`. The strings are concatenated. Example:

  ```
  msgid ""
  "Here is an example of how one might continue a very long string\n"
  "for the common case the string represents multi-line output.\n"
  ```

- **Important**: Newlines represented as `\n` inside quotes are part of the string; physical newlines in the PO file outside quotes are not.

### Uniqueness

- For a valid PO file: no two entries **without** `msgctxt` may share the same `msgid` (or same singular `msgid` for plurals).
- No two entries may share the same `msgctxt` and the same `msgid` (or singular `msgid`).

### Trailing lines

- Lines (e.g. blank or comment) after the last entry are not part of any entry. Tools may drop them or some editors may mishandle them.

---

## 8. PO file format evolution

As of **June 2025**, the sequence **`#=`** at the beginning of a line introduces a **line of flags**, similar to **`#,`**.

- Readers are encouraged to support **`#=`** as an alternative to **`#,`**.
- When **modifying** PO files:
  - If not changing flags: keep both `#,` and `#=` lines unchanged, or emit all flags in a single `#,` line and no `#=` line.
  - When adding flags: add them to the `#,` line (not `#=`).
  - When removing flags that were in `#=`: either update the `#=` line or emit all flags in `#,` and omit `#=`.

From **2027-01-01**, the format will distinguish:
- Either `#,` = workflow flags and `#=` = sticky flags, or the opposite.
- Writers may emit `#=` lines and custom sticky flags.
- Readers/writers must **preserve unknown sticky flags** (not drop them).

This supports extra workflow flags (e.g. pretranslation, review, approval) and project-specific sticky flags.

---

## 9. Obsolete entries (implementation note)

In practice, PO files and tools (e.g. `msgmerge`) also use **obsolete entries**: translations that are no longer needed by the package. When `msgmerge` finds a translation no longer in the source, it comments out the entry but keeps it in the file so it can be reused if the string reappears. See [Obsolete Entries (GNU gettext)](https://www.gnu.org/savannah-checkouts/gnu/gettext/manual/html_node/Obsolete-Entries.html) and the [bug-gettext discussion on documenting #~](https://lists.nongnu.org/archive/html/bug-gettext/2025-01/msg00000.html).

**Rule:** In an obsolete entry, **all lines** start with `#`, including lines that in an active entry would be `msgid` or `msgstr` (GNU manual: “all their lines start with `#`, even those lines containing `msgid` or `msgstr`”).

### 9.1 Format: every line prefixed with `#~ ` or `#~| `

When an entry is made obsolete (e.g. by Emacs po-mode or `msgmerge`), **every line** of that entry is prefixed so it starts with `#`. Keyword and string lines get the prefix **`#~ `** (hash, tilde, space). Comment lines that in an active entry are `# `, `#.`, `#:`, `#,` become **`#~ # `**, **`#~ #.`**, **`#~ #:`**, **`#~ #,`** — i.e. the same **`#~ `** prefix is prepended to the comment line. The previous-untranslated line **`#| msgid "..."`** becomes **`#~| msgid "..."`** (tilde and pipe, no space between `#` and `~|`).

| In active entry        | In obsolete entry      |
|------------------------|------------------------|
| `# ` translator comment | `#~ # ` ...            |
| `#.` extracted comment | `#~ #.` ...            |
| `#:` reference         | `#~ #:` ...            |
| `#,` flags             | `#~ #,` ...            |
| `#| msgid "..."` (previous) | `#~| msgid "..."`  |
| `msgid "..."`          | `#~ msgid "..."`       |
| `msgid_plural "..."`   | `#~ msgid_plural "..."` |
| `msgstr "..."`         | `#~ msgstr "..."`     |
| `msgstr[n] "..."`      | `#~ msgstr[n] "..."`   |
| Continuation `"..."`   | `#~ "..."`            |

### 9.2 Example (Emacs po-mode: fuzzy entry made obsolete)

When a fuzzy entry is made obsolete in Emacs po-mode, the reference and flags lines are also prefixed with `#~ `:

```
#~ #: branch.c builtin/branch.c
#~ #, fuzzy
#~ msgid "See 'git help check-ref-format'"
#~ msgstr "查阅 `man git check-ref-format`"
```

If the entry had a previous untranslated string (from fuzzy match), it would appear as `#~| msgid "..."` before the `#~ msgid` line.

These entries are not part of the active message set but are often kept for reference or fuzzy matching. Parsers like the one in `util/gettext.go` handle `#~` and `#~|` so that obsolete entries can be read, preserved, or filtered as needed.
