# Instructions for AI Agents

This section provides specific instructions for AI agents when handling
translation-related tasks. The use of AI is entirely optional. Many
successful translation teams work effectively without it.

We will use XX as an alias to refer to the language translation code in
the following paragraphs, for example we use "po/XX.po" to refer to the
translation file for a specific language. But this doesn't mean that
the language code has only two letters. The language code can be in one
of two forms: "ll" or "ll\_CC". Here "ll" is the ISO 639 two-letter
language code and "CC" is the ISO 3166 two-letter code for country names
and subdivisions. For example: "de" for German language code, "zh\_CN"
for Simplified Chinese language code.


## Generating or updating po/git.pot

When asked to "update po/git.pot" or similar requests:

1. **Directly execute** the command `make po/git.pot` without checking
   if the file exists beforehand.

2. **Do not verify** the generated file after execution. Simply run the
   command and consider the task complete.

The command will handle all necessary steps including file creation or
update automatically.


## Updating po/XX.po

When asked to "update po/XX.po" or similar requests (where XX is a
language code):

1. **Directly execute** the command `make po-update PO_FILE=po/XX.po`
   without reading or checking the file content beforehand.

2. **Do not verify** the updated file after execution. Simply run the
   command and consider the task complete.

The command will handle all necessary steps including generating
"po/git.pot" and merging new translatable strings into "po/XX.po"
automatically.


## Background Knowledge for Translators and Reviewers

This section provides essential background knowledge about PO file structure
and format that is required for both translation and review tasks. Understanding
these concepts is fundamental before performing any translation or review
operations on `po/XX.po` files.


### Header Entry

Every PO file (`po/XX.po`) contains a special entry called the "header entry"
at the beginning of the file. This entry has an empty `msgid` and contains
metadata about the translation in its `msgstr`:

```po
msgid ""
msgstr ""
"Project-Id-Version: Git\n"
"Report-Msgid-Bugs-To: Git Mailing List <git@vger.kernel.org>\n"
"POT-Creation-Date: 2026-02-14 13:38+0800\n"
"PO-Revision-Date: 2026-02-14 11:41+0800\n"
"Last-Translator: Teng Long <dyroneteng@gmail.com>\n"
"Language-Team: GitHub <https://github.com/dyrone/git/>\n"
"Language: zh_CN\n"
"MIME-Version: 1.0\n"
"Content-Type: text/plain; charset=UTF-8\n"
"Content-Transfer-Encoding: 8bit\n"
"Plural-Forms: nplurals=2; plural=(n != 1);\n"
"X-Generator: Gtranslator 42.0\n"
```

**CRITICAL**: The header entry's `msgstr` contains important metadata and
**MUST NEVER be modified** during translation. When using `msgattrib` to
extract entries, the extracted files (e.g., `po/XX.po.pending`) will also
contain this header entry. Always preserve it exactly as-is.

The header entry serves several purposes:
- Contains translation metadata (translator, language, dates)
- Defines pluralization rules (`Plural-Forms`) for the target language
- Provides encoding and MIME type information
- Stores project and version information


### Glossary Section

PO files may contain a glossary section in comments that appears before the
header entry (the first `msgid ""` entry). This section provides terminology
guidelines for consistent translation:

```po
# Git glossary for Chinese translators
#
#   English                          |  Chinese
#   ---------------------------------+--------------------------------------
#   3-way merge                      |  三路合并
#   branch                           |  分支
#   commit                           |  提交
#   ...
```

**IMPORTANT**: Always read and reference the glossary section when translating
or reviewing entries. It ensures consistent terminology throughout the
translation. The glossary appears in comments (lines starting with `#`), so
it is preserved when extracting entries with `msgattrib`.


### Single-line vs Multi-line Entries

PO files contain two types of entries based on their structure:

**Single-line entries** have a simple format:
```po
msgid "commit message"
msgstr "提交说明"
```

**Multi-line entries** have a more complex structure where the first line
of both `msgid` and `msgstr` is an empty string:
```po
msgid ""
"Line 1\n"
"Line 2"
msgstr ""
"行 1\n"
"行 2"
```

**CRITICAL**: For multi-line entries:
- The first line of `msgid` is always `msgid ""` (empty string)
- The first line of `msgstr` is always `msgstr ""` (empty string)
- Each subsequent line is a quoted string
- Line breaks within the text are represented as `\n` escape sequences
- All quote marks and line structure must be preserved exactly


### Locating untranslated, fuzzy, and obsolete entries

The structure of multi-line entries is why you **MUST NOT** use
`grep '^msgstr ""$'` to locate untranslated entries - it would incorrectly
match all multi-line entries, causing false positives. Always use GNU gettext
tools (`msgattrib`) to reliably identify untranslated entries.

- **Untranslated entries**:

  ```shell
  msgattrib --untranslated --no-obsolete po/XX.po
  ```

- **Fuzzy entries**:

  ```shell
  msgattrib --only-fuzzy --no-obsolete po/XX.po
  ```

- **Obsolete entries** (marked with `#~`):

  ```shell
  msgattrib --obsolete --no-wrap po/XX.po
  ```

If you only want the message IDs, you can pipe to:

```shell
msgattrib --untranslated --no-obsolete po/XX.po | sed -n '/^msgid /,/^$/p'
```

```shell
msgattrib --only-fuzzy --no-obsolete po/XX.po | sed -n '/^msgid /,/^$/p'
```

**Note**: When counting entries, remember that the header entry (with empty
`msgid`) is included in the count. When calculating statistics, subtract 1
from the total count to exclude the header entry.


### Preserving Special Characters

When translating or reviewing translations, it is critical to preserve special
characters and escape sequences exactly as they appear in the `msgid`:

- **Escape sequences**: Keep `\n`, `\"`, `\\`, `\t`, etc. within quotes
  exactly as they appear. Do NOT convert `\n` to `\\n` (double backslash),
  and do NOT replace `\n` with actual line breaks.

- **Placeholders**: Preserve format specifiers like `%s`, `%d`, `%.*s`, etc.
  exactly as they appear. Only reorder them using positional syntax when
  necessary (see Placeholder Reordering section below).

- **Quotes**: Keep all quote marks (`"`) as-is. Do not add or remove quotes.

**Example of correct preservation**:
```po
msgid "Line 1\nLine 2"
msgstr "行 1\n行 2"  # Correct: \n preserved as escape sequence
```

**WRONG examples**:
```po
msgstr "行 1\\n行 2"  # WRONG: double backslash
msgstr "行 1
行 2"  # WRONG: actual line break instead of \n
```


### Placeholder Reordering

When a translation requires reordering placeholders from the original `msgid`,
you must use positional parameter syntax (`%n$`) to ensure each argument maps
to the correct source value. Keep width/precision modifiers intact and place
the position specifier before them.

**Example 1: positional parameters with precision specifier**:

```po
#, c-format
msgid "missing environment variable '%s' for configuration '%.*s'"
msgstr "配置 '%3$.*2$s' 缺少环境变量 '%1$s'"
```

In this example:
- `%s` in the original is argument 1, so it becomes `%1$s` in the translation
- `%.*s` in the original requires two arguments: argument 2 (precision value)
  and argument 3 (string). In the translation, it becomes `%3$.*2$s`, which
  means: use argument 3 (the string) with precision from argument 2

**Example 2: positional parameters across multiple lines**:

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

In this example:
- The original msgid has 4 `%s` placeholders that appear in order across
  multiple lines
- In the translation, the placeholders are reordered:
  * First `%s` (submodule name) → `%1$s` appears second in Chinese
  * Second `%s` (module name) → `%2$s` appears first in Chinese
  * Third `%s` (submodule name again) → `%3$s` remains in similar position
  * Fourth `%s` (path) → `%4$s` remains in similar position
- Each line in the multi-line string must be a complete quoted string

**Rules for placeholder reordering**:
1. Use `%n$` syntax where `n` is the argument position (1-based)
2. Place the position specifier before width/precision modifiers
3. For `%.*s` style placeholders, map both the precision and string arguments
4. Always verify that all placeholders are accounted for and correctly mapped


### Validating PO File Format

This section describes how to validate PO files and handle validation errors.
This is a general-purpose validation procedure that can be applied to any PO
file, including `po/XX.po`, `po/XX.po.pending`, or any extracted PO file.

Validate a PO file:

```shell
msgfmt --check -o /dev/null po/XX.po
```

Common validation errors include:
- Unclosed quotes
- Missing escape sequences
- Invalid placeholder syntax
- Malformed multi-line entries
- Incorrect line breaks in multi-line strings

**Handling validation errors with automatic repair**:
When `msgfmt` reports an error, it provides the line number where the error
was detected. Use this information to locate and fix the issue.


## Translating po/XX.po

When asked to translate "po/XX.po" or similar requests:

The following steps are the core operational procedures that AI Agents must
follow when translating `po/XX.po`. These steps should be executed sequentially
and iteratively until all entries are translated.

**IMPORTANT: You must complete the translation of ALL untranslated and fuzzy
entries. Do not stop early or report partial progress. Continue iterating
through steps 1-10 until `po/XX.po.pending` is empty (contains no entries).**

**NOTE: Translation may take a long time for large files. You can safely
interrupt the process at the following points:**
- After completing a batch (after step 8 completes)
- After validation passes (after step 5 completes)

1. **Extract entries to translate and record initial state**: Use `msgattrib`
   to extract untranslated and fuzzy messages separately, then combine them.
   Also record the initial total count for progress tracking. For fuzzy
   entries, extract them first with their original translations as reference
   (saved to a separate file), then clear their `msgstr` values:

   **NOTE**: When using `msgattrib` to extract files, you **MUST** use
   redirect operators (e.g., `>`) instead of the `-o <filename>` option.
   The `-o` option will not overwrite an existing file if nothing is extracted,
   which can lead to stale data in the output file.

   ```shell
   # Extract untranslated entries
   msgattrib --untranslated --no-obsolete po/XX.po >po/XX.po.untranslated

   # Extract fuzzy entries with original translations as reference
   msgattrib --only-fuzzy --no-obsolete po/XX.po >po/XX.po.fuzzy.reference

   # Extract fuzzy entries with cleared msgstr for translation
   msgattrib --only-fuzzy --no-obsolete --clear-fuzzy --empty po/XX.po >po/XX.po.fuzzy

   # Combine untranslated and fuzzy entries (also use redirect operator instead of -o)
   msgcat --use-first po/XX.po.untranslated po/XX.po.fuzzy >po/XX.po.pending

   # Record initial total count for progress tracking (first time only)
   if test ! -f po/XX.po.total
   then
       UNTRANS_INIT=$(msgattrib --untranslated --no-obsolete po/XX.po 2>/dev/null | grep -c '^msgid ' || true)
       UNTRANS_INIT=$((UNTRANS_INIT > 0 ? UNTRANS_INIT - 1 : 0))
       FUZZY_INIT=$(msgattrib --only-fuzzy --no-obsolete po/XX.po 2>/dev/null | grep -c '^msgid ' || true)
       FUZZY_INIT=$((FUZZY_INIT > 0 ? FUZZY_INIT - 1 : 0))
       echo $((UNTRANS_INIT + FUZZY_INIT)) >po/XX.po.total
   fi
   ```

   If `po/XX.po.pending` is empty, skip to step 10 (clean up) as the
   translation is complete.

   **Note**: The `po/XX.po.fuzzy.reference` file contains fuzzy entries with
   their original translations. You can reference these during translation to
   preserve good translations or understand context, but always provide fresh
   translations in `msgstr` fields.

2. **Check file size and truncate with dynamic batch size**: **BEFORE
   translating**, check if the `po/XX.po.pending` file is too large. Count
   the number of entries (msgid blocks) in the file. If it contains more than
   100 entries, truncate it to process in batches. Use dynamic batch sizing
   based on remaining entry count for optimal efficiency:

   ```shell
   # Count entries in po/XX.po.pending
   ENTRY_COUNT=$(grep -c '^msgid ' po/XX.po.pending 2>/dev/null || true)
   ENTRY_COUNT=$((ENTRY_COUNT > 0 ? ENTRY_COUNT - 1 : 0))

   # Dynamic batch size selection based on remaining entries
   if test "$ENTRY_COUNT" -gt 100
   then
       # Dynamic batch size:
       # - Very large files (>500 entries): NUM=100 (larger batches for efficiency)
       # - Large files (200-500 entries): NUM=75 (medium batches)
       # - Medium files (100-200 entries): NUM=50 (default batch size)
       # - Small files (<100 entries): process all at once (skip truncation)
       if test "$ENTRY_COUNT" -gt 500
       then
           NUM=100
       elif test "$ENTRY_COUNT" -gt 200
       then
           NUM=75
       else
           NUM=50
       fi
       # Use "count++" because we extract additional header entry
       awk -v num="$NUM" '/^msgid / && count++ > num {exit} 1' po/XX.po.pending |
           tac | awk '/^$/ {found=1} found' | tac >po/XX.po.pending.tmp
       mv po/XX.po.pending.tmp po/XX.po.pending
       echo "Processing batch of $NUM entries (out of $ENTRY_COUNT remaining)"
   else
       echo "Processing all $ENTRY_COUNT entries at once"
   fi
   ```

3. **Reference glossary**: Read the glossary section from the header of
   `po/XX.po.pending` (if present, before the first `msgid`). See the "Glossary
   Section" subsection above for details. Use it for consistent terminology
   during translation.

4. **Translate entries**: Translate entries in `po/XX.po.pending` from English
   (msgid) to the target language (msgstr), and write the translation results
   directly into `po/XX.po.pending`:

   - **Translation approach**: **MANDATORY**: Use a large language model
     (LLM) to translate the `po/XX.po.pending` file. The LLM must:
     * Read and understand the complete PO file format, including all
       structural elements (comments, flags, msgid, msgstr, etc.)
     * Preserve ALL formatting exactly as-is: quotes, line breaks, escape
       sequences, multi-line structures
     * Only modify the `msgstr` field content, keeping all format markers
       unchanged
     * Understand context from surrounding entries and glossary
     * Generate natural, fluent translations in the target language
     * For fuzzy entries: Optionally reference `po/XX.po.fuzzy.reference` to
       understand previous translation context, but always provide fresh,
       accurate translations

     **CRITICAL**: **DO NOT use pattern matching, regular expressions, string
     replacement, or batch substitution** - these approaches will break PO file
     format, especially for multi-line entries. The LLM must work with the file
     as a structured document, not as plain text.

   - **Batch optimization**: For efficiency, you can process simple entries
     (single-line, no placeholders, no special formatting) in larger batches,
     while complex entries (multi-line, with placeholders, formatted) should
     be handled more carefully with smaller batches or individually.

   - **For untranslated entries**: Translate the English `msgid` into the
     target language in `msgstr`.

   - **For fuzzy entries**: Since fuzzy entries have been cleared (empty
     `msgstr`) in step 1, treat them the same as untranslated entries:
     translate the English `msgid` into the target language in `msgstr`.
     The `#, fuzzy` tag will be automatically removed when the entry is
     merged back into `po/XX.po`.

   - **For plural entries**: For entries with `msgid_plural`, provide all
     required `msgstr[n]` forms based on the `Plural-Forms` header in
     `po/XX.po.pending`. The number of plural forms required depends on the
     target language's plural rules. Refer to the header entry's `Plural-Forms`
     field to determine how many forms are needed.

   - **Placeholder reordering**: See the "Placeholder Reordering" section
     above for detailed guidelines. When reordering is necessary, use positional
     parameter syntax (`%n$`) to map arguments correctly.

5. **Validate translated entries**: Before merging, validate the PO file format
   of `po/XX.po.pending` to ensure it is syntactically correct. Use `msgfmt`
   to perform comprehensive validation. See the "Validating PO File Format"
   section above for details:

   ```shell
   # Validate the pending file
   if msgfmt --check -o /dev/null po/XX.po.pending 2>&1
   then
       echo "Validation passed"
   else
       echo "ERROR: PO file format is invalid."
       echo "Do not proceed. Fix the errors and re-validate."
       # See "Handling validation errors" in "Validating PO File Format" section
       # for recovery procedures
       exit 1
   fi
   ```

   If validation fails:
   - **DO NOT** attempt to merge the corrupted file
   - Follow the error handling procedures in the "Validating PO File Format"
     section above to locate errors by line number and attempt automatic repair
   - Re-validate after repairs: `msgfmt --check -o /dev/null po/XX.po.pending`
   - If automatic repair is not successful, re-extract `po/XX.po.pending` by
     repeating step 1 and re-translate the entries

6. **Merge validated entries**: After successful validation, merge the validated
   entries from `po/XX.po.pending` into `po/XX.po`. The `msgcat` command
   performs the merge operation:

   ```shell
   # Merge validated entries
   if msgcat --use-first po/XX.po.pending po/XX.po >po/XX.po.merged 2>&1
   then
       # Merge successful
       mv po/XX.po.merged po/XX.po
       echo "Batch merged successfully"
   else
       # Merge failed
       echo "ERROR: Merge failed."
       echo "Do not proceed. Check the source files."
       rm -f po/XX.po.merged
       exit 1
   fi
   ```

   If merge fails:
   - **DO NOT** attempt to use the corrupted merged file
   - **DO NOT** continue modifying the file
   - Check that both `po/XX.po.pending` and `po/XX.po` are valid
   - Re-extract if needed and retry

7. **Save progress checkpoint**: After successful merge, save progress state
   to enable recovery from interruptions:

   ```shell
   # Save progress checkpoint
   BATCH_NUM=$(cat po/XX.po.batch 2>/dev/null || echo 0)
   BATCH_NUM=$((BATCH_NUM + 1))
   echo $BATCH_NUM >po/XX.po.batch
   ```

8. **Report detailed progress and repeat**: After merging, report detailed
   progress including percentage, remaining batches estimate, and repeat the
   process:

   ```shell
   # Get current remaining counts
   UNTRANS=$(msgattrib --untranslated --no-obsolete po/XX.po 2>/dev/null | grep -c '^msgid ' || true)
   UNTRANS=$((UNTRANS > 0 ? UNTRANS - 1 : 0))
   FUZZY=$(msgattrib --only-fuzzy --no-obsolete po/XX.po 2>/dev/null | grep -c '^msgid ' || true)
   FUZZY=$((FUZZY > 0 ? FUZZY - 1 : 0))
   REMAINING=$((UNTRANS + FUZZY))

   # Get initial total (from step 1)
   TOTAL_ENTRIES=$(cat po/XX.po.total 2>/dev/null || echo 0)
   if test "$TOTAL_ENTRIES" -eq 0
   then
       TOTAL_ENTRIES=$REMAINING
       echo $TOTAL_ENTRIES >po/XX.po.total
   fi

   # Calculate progress
   PROCESSED=$((TOTAL_ENTRIES - REMAINING))
   if test "$TOTAL_ENTRIES" -gt 0
   then
       PROGRESS=$((100 * PROCESSED / TOTAL_ENTRIES))
   else
       PROGRESS=100
   fi

   # Estimate remaining batches (using average batch size)
   BATCH_NUM=$(cat po/XX.po.batch 2>/dev/null || echo 0)
   if test "$BATCH_NUM" -gt 0 && test "$REMAINING" -gt 0
   then
       AVG_BATCH=$((PROCESSED / BATCH_NUM))
       if test "$AVG_BATCH" -gt 0
       then
           EST_BATCHES=$((REMAINING / AVG_BATCH + 1))
       else
           EST_BATCHES="?"
       fi
   else
       EST_BATCHES="?"
   fi

   # Report detailed progress
   echo "========================================="
   echo "Translation Progress Report"
   echo "========================================="
   echo "Progress: $PROGRESS% ($PROCESSED/$TOTAL_ENTRIES entries processed)"
   echo "Remaining: $UNTRANS untranslated + $FUZZY fuzzy = $REMAINING total"
   echo "Batches completed: $BATCH_NUM"
   if test "$EST_BATCHES" != "?"
   then
       echo "Estimated remaining batches: ~$EST_BATCHES"
   fi
   echo "========================================="
   ```

   **MANDATORY**: You MUST repeat steps 1-8 (extract, truncate if needed,
   translate, validate, merge, save checkpoint, report) until `po/XX.po.pending`
   is completely empty. Then proceed to steps 9-10 (final verification and
   cleanup). Do not report partial progress or stop early. The task is only
   complete when there are zero untranslated and zero fuzzy entries remaining
   in `po/XX.po`.

9. **Final verification**: Before cleanup, perform a final verification to
   ensure all entries are translated:

   ```shell
   # Final check
   UNTRANS=$(msgattrib --untranslated --no-obsolete po/XX.po 2>/dev/null | grep -c '^msgid ' || true)
   UNTRANS=$((UNTRANS > 0 ? UNTRANS - 1 : 0))
   FUZZY=$(msgattrib --only-fuzzy --no-obsolete po/XX.po 2>/dev/null | grep -c '^msgid ' || true)
   FUZZY=$((FUZZY > 0 ? FUZZY - 1 : 0))
   if test "$UNTRANS" -eq 0 && test "$FUZZY" -eq 0
   then
       echo "Translation complete! All entries translated."
   else
       echo "WARNING: Still have $UNTRANS untranslated + $FUZZY fuzzy entries."
       echo "Do not clean up. Continue with step 1."
       exit 1
   fi
   ```

10. **Clean up**: Only after confirming that translation is complete (step 9
    passes), remove all temporary files and progress tracking files:

   ```shell
   # Remove temporary files
   rm -f "po/XX.po.pending"
   rm -f "po/XX.po.untranslated"
   rm -f "po/XX.po.fuzzy"
   rm -f "po/XX.po.fuzzy.reference"
   rm -f "po/XX.po.total"
   rm -f "po/XX.po.batch"
   rm -f "po/XX.po.merged"
   rm -f "msgid_lines.txt"  # if created
   echo "Cleanup complete. Translation finished successfully."
   ```

**To resume after interruption**:

Re-run the translation command. The system will continue from the last progress
checkpoint.


## Reviewing po/XX.po

When asked to review translations in `po/XX.po` or similar:

AI agents can review translations in `po/XX.po`, targeting: (1) the full file,
(2) changes in a specific commit, or (3) changes since a specific commit.

Preparing a proper diff (original vs new) with full context for review is
difficult without tooling. Plain `git diff` may fragment or lose PO translation
context. Use `git-po-helper compare` instead to extract new or changed entries
into a valid PO file for review.

### Extracting content to review

The `git-po-helper compare` command extracts new or changed entries between two
PO file versions and writes them to stdout. Redirect to a file for review:

```shell
# Review local changes (HEAD vs working tree)
git-po-helper compare po/XX.po >po/review.po

# Review changes in a specific commit (parent vs commit)
git-po-helper compare --commit <commit> po/XX.po >po/review.po

# Review changes since a commit (commit vs working tree)
git-po-helper compare --since <commit> po/XX.po >po/review.po

# Review between two commits
git-po-helper compare -r <commit1>..<commit2> po/XX.po >po/review.po

# Compare two worktree files
git-po-helper compare po/XX-old.po po/XX-new.po >po/review.po
```

**Options summary**

| Option              | Meaning                                        |
|---------------------|------------------------------------------------|
| (none)              | Compare HEAD with working tree (local changes) |
| `--commit <commit>` | Compare parent of commit with the commit       |
| `--since <commit>`  | Compare commit with working tree               |
| `-r x..y`           | Compare revision x with revision y             |
| `-r x..`            | Compare revision x with working tree           |
| `-r x`              | Compare parent of x with x                     |

**Note**:
1. Output is empty when there are no new or changed entries. If `po/review.po`
   is empty, there is nothing to review.
2. If the output is not empty, always has a valid PO head entry.


### Handling large review files

When the extracted review file is too large to review in one pass, use
`git-po-helper msg-select` to split it by entry index range into smaller
files and review each batch separately.

Entry numbers: 0 is the header (included by default; use `--no-header` to
omit); 1, 2, 3, ... are the first, second, third content entries. Range
format: `--range "1-50"` (entries 1–50), `--range "-50"` (first 50 entries),
`--range "51-"` (from entry 51 to end), or combined like
`--range "1-50,101-150"`.

Example workflow for a large `po/review.po`:

```shell
# Extract first 50 entries to a batch file
git-po-helper msg-select --range "-50" po/review.po >po/review-batch1.po

# Extract entries 51–100 for the next batch
git-po-helper msg-select --range "51-100" po/review.po >po/review-batch2.po

# Extract entries 101 to end for the last batch
git-po-helper msg-select --range "101-" po/review.po >po/review-batch3.po

# Or extract a range to a fragment file (no header)
git-po-helper msg-select --range "1-50" --no-header po/review.po >po/review-fragment.po
```

Review each batch file in turn, then apply corrections to the main `po/XX.po`.


### Review procedure

1. **Extract entries**: Run `git-po-helper compare` with the desired range and
   redirect output to `po/review.po` (see above). Clear any prior batch state:
   `rm -f po/review.batch`.

2. **Check file size and prepare batch if large**: **BEFORE reviewing**, count
   entries in `po/review.po`. If it contains more than 100 entries, use
   `git-po-helper msg-select` to process in batches. Use dynamic batch sizing.
   Use `po/review.batch` to persist `BATCH_NUM` across iterations. Process all
   batches in one run to produce a single JSON report (no interrupt/resume).

   ```shell
   # Count content entries in po/review.po (exclude header)
   ENTRY_COUNT=$(grep -c '^msgid ' po/review.po 2>/dev/null || true)
   ENTRY_COUNT=$((ENTRY_COUNT > 0 ? ENTRY_COUNT - 1 : 0))

   # Dynamic batch size (same logic as translation workflow)
   if test "$ENTRY_COUNT" -gt 100
   then
       if test "$ENTRY_COUNT" -gt 500
       then
           NUM=100
       elif test "$ENTRY_COUNT" -gt 200
       then
           NUM=75
       else
           NUM=50
       fi
       # BATCH_NUM: persist in po/review.batch for iteration across steps
       BATCH_NUM=$(cat po/review.batch 2>/dev/null || echo 0)
       BATCH_NUM=$((BATCH_NUM + 1))
       echo $BATCH_NUM >po/review.batch
       START=$(((BATCH_NUM - 1) * NUM + 1))
       END=$((BATCH_NUM * NUM))
       if test "$END" -gt "$ENTRY_COUNT"
       then
           END=$ENTRY_COUNT
       fi

       # Extract batch: use -N for first batch, N-M for middle, N- for last
       if test "$BATCH_NUM" -eq 1
       then
           git-po-helper msg-select --range "-$NUM" po/review.po >po/review-batch.po
       elif test "$END" -ge "$ENTRY_COUNT"
       then
           git-po-helper msg-select --range "$START-" po/review.po >po/review-batch.po
       else
           git-po-helper msg-select --range "$START-$END" po/review.po >po/review-batch.po
       fi

       echo "Reviewing batch $BATCH_NUM: entries $START-$END (of $ENTRY_COUNT)"
       REVIEW_FILE="po/review-batch.po"
   else
       REVIEW_FILE="po/review.po"
       echo "Reviewing all $ENTRY_COUNT entries at once"
   fi
   ```

3. **Read context**: Reference the "Background Knowledge for Translators
   and Reviewers" section for PO format, placeholder rules, and terminology.
   Read the glossary section if presents in the review PO file and add it
   to the context for reference.

4. **Review entries**: For each entry in `$REVIEW_FILE`, check:
   - Terminology consistency with the glossary
   - Correctness of placeholder preservation and reordering
   - Naturalness and fluency in the target language
   - Technical accuracy of the translation

5. **Output format**: When providing review feedback, you may output a JSON
   report with issues and suggestions (see the "Review JSON Format" section
   below). Alternatively, provide inline suggestions or a human-readable
   report.

6. **Apply changes**: If the review identifies issues, the human translator
   applies the suggested corrections to `po/XX.po` (or the appropriate source
   file). The review output file (`po/review.po` or `po/review-batch.po`) is
   for reference only; it is not merged back automatically.

7. **Report progress and repeat**: If processing in batches, report progress
   and repeat from step 2 until all batches are reviewed. Accumulate issues
   into a single JSON report across all batches.

   ```shell
   if test -f po/review-batch.po
   then
       PROCESSED=$((BATCH_NUM * NUM))
       if test "$PROCESSED" -gt "$ENTRY_COUNT"
       then
           PROCESSED=$ENTRY_COUNT
       fi
       PROGRESS=$((100 * PROCESSED / ENTRY_COUNT))
       echo "Progress: $PROGRESS% ($PROCESSED/$ENTRY_COUNT entries reviewed)"
       if test "$END" -ge "$ENTRY_COUNT"
       then
           echo "Review complete. Clean up: rm -f po/review-batch.po po/review.batch"
           echo "Output report: git-po-helper stat --review po/review.json po/review.po"
           # LLM: Display the output and execute: git-po-helper stat --review po/review.json po/review.po
       else
           echo "Repeat from step 2 for next batch."
       fi
   fi
   ```

   **MANDATORY**: When `po/review.po` has more than 100 entries, you MUST
   process all batches in one run and produce a single JSON report. Do not
   stop early; accumulate issues from each batch into the final output.


### Review JSON Format

When outputting a JSON report, use the following structure:

```json
{
  "issues": [
    {
      "msgid": "commit",
      "msgstr": "承诺",
      "score": 0,
      "description": "术语错误：'commit'应译为'提交'",
      "suggestion": "提交"
    },
    {
      "msgid": "repository",
      "msgstr": "仓库",
      "score": 2,
      "description": "一致性问题：其他地方使用'版本库'",
      "suggestion": "版本库"
    }
  ]
}
```

- `issues`: Array of issues found. Each issue has:
  - `msgid`: Original string (msgid).
  - `msgstr`: Current translation.
  - `score`: 0 = critical (must fix), 2 = minor (needs adjustment), 3 = perfect.
  - `description`: Explanation of the issue.
  - `suggestion`: Suggested correction.


## Human translators remain in control

Translation of Git is a human-driven community effort. Language team leaders and
contributors are responsible for:

- Understanding the technical context of Git commands and messages
- Making linguistic and cultural adaptation decisions for their target language
- Maintaining translation quality and consistency within their language team
- Ensuring translations follow Git l10n conventions and community standards
- Building and maintaining language-specific glossaries
- Reviewing and approving all changes before submission

AI tools, if used, serve only to accelerate routine tasks:

- Generating first-draft translations for new or updated messages
- Identifying untranslated or fuzzy entries across large PO files
- Checking consistency with existing translations and glossary terms
- Detecting technical errors (missing placeholders, formatting issues)
- Reviewing translations against quality criteria

AI-generated output should always be treated as rough drafts requiring human
review, editing, and approval by someone who understands both the technical
context and the target language.  The best results come from combining AI
efficiency with human judgment, cultural insight, and community engagement.
