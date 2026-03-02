# 方案：GettextEntry.Fuzzy 为 fuzzy 状态唯一来源

## 目标

fuzzy 状态仅在 `GettextEntry.Fuzzy` 一处维护。Comments 中不再保存 fuzzy 标记；读 PO 时从注释解析到 Fuzzy 并清理 Comments，写 PO 时根据 Fuzzy 再恢复注释。

## 注释中 fuzzy 的两种形态

1. **独立存在**：`"#, fuzzy"` → 过滤后为空 → 从 Comments 中**删除**该行（不保留空字符串）。
2. **与其他 tag 共存**：`"#, fuzzy, c-format"` 或 `"#, c-format, fuzzy"` → 过滤后为 `"#, c-format"` → 保留在 Comments 中。

## 读 PO → GettextEntry（解析）

- 用现有逻辑由 `entryHasFuzzyFlag(Comments)` 得到 `IsFuzzy`，设 `ent.Fuzzy = e.IsFuzzy`。
- 复制 `e.Comments` 到 `ent.Comments` 时，对每一行调用 **StripFuzzyFromCommentLine(line)**：
  - 若返回 `""`（原行为 `"#, fuzzy"` 或仅含 fuzzy），**不**加入 `ent.Comments`。
  - 若返回非空（如 `"#, c-format"`），加入 `ent.Comments`。
- 结果：内存中 fuzzy 只存在于 `Fuzzy`；Comments 中无 fuzzy 行，仅保留其他 flag 或非 flag 注释。

## 写 GettextEntry → PO（保存时恢复）

写每条 entry 的 Comments 时：

1. **非 flag 注释**（如 `#: file.c`、`#. translator`）：原样写出。
2. **flag 注释**（以 `#,` 开头的行，此时已不含 fuzzy）：
   - 若 `entry.Fuzzy == true`：写出**合并行** `#, fuzzy, <其他 flags>`（将 fuzzy 与行内其他 tag 合并为一行）。
   - 若 `entry.Fuzzy == false`：原样写出该行（如 `#, c-format`）。
3. **无 flag 行但 Fuzzy == true**：在 Comments 写完后，若尚未写过含 fuzzy 的行，则补写一行 `#, fuzzy\n`。

这样写回 PO 时，fuzzy 要么与其他 tag 合并为一行，要么单独一行，与常见 PO 格式一致。

## 实现要点

| 位置 | 内容 |
|------|------|
| **util/gettext.go** | **StripFuzzyFromCommentLine(line string) string**：对以 `#,` 开头的行，去掉其中的 `fuzzy` 标记后返回；若结果为 `#,` 或空则返回 `""`。 |
| **util/gettext.go** | **MergeFuzzyIntoFlagLine(line string) string**（或写 PO 时内联）：给定已无 fuzzy 的 flag 行（如 `#, c-format`），若需要加入 fuzzy，返回 `#, fuzzy, c-format`（fuzzy 放前，其余保持顺序）。仅用于写 PO。 |
| **util/gettext_json.go** | **PoEntriesToGettextJSON**：复制 Comments 时用 StripFuzzyFromCommentLine，只保留非空结果。 |
| **util/gettext_json.go** | **WriteGettextJSONToPO**：写 Comments 时区分 flag / 非 flag；对 flag 行若 `entry.Fuzzy` 则输出合并行；若 `entry.Fuzzy` 且未输出过任何 flag 行，则补写 `#, fuzzy\n`。 |

## 测试

- PO → JSON：`#, fuzzy` → Fuzzy=true，Comments 无该行；`#, fuzzy, c-format` → Fuzzy=true，Comments 含 `#, c-format`。
- JSON → PO：Fuzzy=true 且 Comments 无 flag → 只写 `#, fuzzy\n`；Fuzzy=true 且 Comments 含 `#, c-format` → 只写一行 `#, fuzzy, c-format\n`。
- 往返：PO 与 JSON 互转后 fuzzy 与注释行为一致。
