# Test fixtures

## `zh_CN_example.po`

A trimmed PO file containing the **header** (comments + metadata) and the
**first sevel content entries** from `po/zh_CN.po` (e.g. from the
git-l10n/git-po repository). Used by **t0120-msg-select-json-roundtrip.sh**
to verify PO ↔ gettext JSON round-trip: after converting to JSON and back
to PO, both files are formatted with `msgcat` and compared; they must be
identical.

To regenerate this fixture from a full `zh_CN.po` (e.g. `../git-po/po/zh_CN.po`):

```sh
git-po-helper msg-select --range "1-20" -o test/fixtures/zh_CN_example.po /path/to/po/zh_CN.po
```
