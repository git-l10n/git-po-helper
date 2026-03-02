# Design: Command Error Handling Refactoring

## 1. 现状与问题

### 1.1 当前实现

- **错误类型**：
  - `newUserError` / `newUserErrorF` → `commandError{userError: true}`，表示用户错误
  - `errExecute` → 通用系统错误
- **Cobra 配置**：所有子命令显式设置 `SilenceErrors: true`，root 设置 `SilenceUsage: true`
- **main.go 逻辑**：根据 `IsUserError()` 和 `resp.Cmd.SilenceErrors` 决定输出

### 1.2 混乱点

1. **SilenceErrors 与错误类型双重控制**：main.go 同时依赖 `IsUserError` 和 `SilenceErrors`，逻辑分支复杂
2. **语义不清**：`userError` 既表示「应显示 Usage」，又混入 `userErrorRegexp` 的启发式匹配
3. **errExecute 无信息**：系统错误统一显示 "fail to execute"，丢失具体错误内容
4. **compare/agent-run-report 等**：返回 `newUserErrorF("%v", err)` 包装系统错误，导致系统错误也被当作用户错误并显示 Usage

## 2. 设计目标

1. **移除子命令的 SilenceErrors 设置**：不再在各子命令上显式设置
2. **错误类型语义清晰**：按「是否显示错误内容」「是否显示 Usage」明确分类
3. **统一由 main.go 处理**：根据错误类型决定输出，逻辑简单

## 3. 新错误类型设计

### 3.1 两种错误类型（去掉 SilenceError）

| 类型 | 显示错误内容 | 显示 Usage | 适用场景 |
|------|-------------|-----------|----------|
| **ErrorWithUsage** | ✓ | ✓ | 参数互斥、argument 数量错误、参数解析相关错误，用户需要看用法 |
| **StandardError** | ✓ | ✗ | 执行失败、文件不存在、业务逻辑错误等，仅展示错误详情 |

**说明**：去掉 SilenceError。所有返回错误均需展示内容，由 main.go 统一输出，避免调用链中重复打印。

### 3.2 错误类型判定规则

- **NewErrorWithUsage**：参数互斥（如 `--unset-fuzzy` 与 `--clear-fuzzy`）、argument 数量错误（如 "requires exactly one argument"）、参数解析相关错误
- **NewStandardError / 普通 error**：不需要显示 Usage 的场景，包括 `newUserError` 包装的文件 I/O 错误、util 层返回的执行错误等

### 3.3 API 设计

```go
// cmd/root.go

// NewErrorWithUsage 表示用户用法错误，应显示错误信息 + Usage
func NewErrorWithUsage(a ...interface{}) error
func NewErrorWithUsageF(format string, a ...interface{}) error

// NewStandardError 表示普通执行错误，仅显示错误信息，不显示 Usage
func NewStandardError(a ...interface{}) error
func NewStandardErrorF(format string, a ...interface{}) error
```

### 3.4 实现方式

```go
type errorWithUsage struct { msg string }
func (e errorWithUsage) Error() string { return e.msg }

func IsErrorWithUsage(err error) bool
// 其余为 StandardError 或普通 error，均按 StandardError 处理
```

## 4. Cobra 配置

### 4.1 SilenceErrors

**问题**：若子命令不设置 `SilenceErrors`，Cobra 会在 `RunE` 返回错误时自动打印 `Error: <message>`，与 main.go 的自定义输出重复。

**方案**：移除各子命令的 `SilenceErrors` 设置；在 **root 命令** 上设置 `SilenceErrors: true`，由 main.go 统一负责错误输出。若 root 级设置不足以覆盖子命令，则采用 init 时批量设置。

### 4.2 SilenceUsage

**保持 root 的 `SilenceUsage: true`**。是否显示 Usage 由本程序根据错误类型决定，不依赖 Cobra 的默认行为。

## 5. main.go 新逻辑

```go
if resp.Err != nil {
    errOut := resp.Cmd.ErrOrStderr()
    fmt.Fprintf(errOut, "ERROR: %s\n", resp.Err)
    if IsErrorWithUsage(resp.Err) {
        fmt.Fprint(errOut, resp.Cmd.UsageString())
    }
    os.Exit(-1)
}
```

## 6. 迁移映射

| 当前返回 | 新返回 | 说明 |
|----------|--------|------|
| `newUserError(...)` 参数互斥、argument 数量、参数解析 | `NewErrorWithUsage(...)` | 需要 Usage |
| `newUserError(...)` 其他用户可见错误 | `NewStandardError(...)` 或普通 `fmt.Errorf` | 不需 Usage |
| `newUserErrorF("%v", err)` 包装 util 错误 | `NewStandardErrorF("%v", err)` | 执行错误，不需 Usage |
| `errExecute`（util 返回 error） | `NewStandardErrorF("%v", err)` 传递原始错误 | 保留具体信息 |
| `errExecute`（util 返回 bool） | `NewStandardError("operation failed")` 等通用消息 | util 内 log 保留，供用户查看详情 |

### 6.1 NewErrorWithUsage 适用场景（示例）

- 参数互斥：`--unset-fuzzy` 与 `--clear-fuzzy`、`--use-agent-md` 与 `--use-local-orchestration`
- argument 数量：`requires exactly one argument`、`expects at most one argument`、`needs no arguments`
- 参数解析：`must given 1 argument`、`requires at least one argument`

### 6.2 NewStandardError 适用场景（示例）

- 文件 I/O：`failed to create output file`、`failed to read`
- 执行失败：util 层返回的 `fmt.Errorf` 包装
- 业务逻辑：`failed to prepare review data`、`file does not exist`

## 7. 合理性评估

### 7.1 优点

1. **语义清晰**：两种类型对应两种输出策略（是否显示 Usage），易于理解和维护
2. **去重**：不再依赖 `SilenceErrors` 与 `IsUserError` 的组合判断
3. **信息保留**：StandardError 可携带具体错误，便于排查
4. **扩展简单**：新增行为时只需增加类型和 main 分支
5. **无重复输出**：去掉 util 层冗余 log，错误仅由 main.go 统一打印

### 7.2 潜在问题

1. **向后兼容**：`errExecute` 改为 `NewStandardErrorF` 后，原 "fail to execute" 会变为具体错误信息，输出可能变化，需在测试中验证
2. **util 层错误**：util 返回的 `error` 需在 cmd 层用 `NewStandardErrorF("%v", err)` 包装，以保证被正确分类

### 7.3 建议

- 第一阶段：实现两种类型 + main 逻辑 + 移除子命令 SilenceErrors
- 第二阶段：按文件逐步迁移 `newUserError` / `errExecute` 到新类型
- 第三阶段：删除 `commandError`、`userErrorRegexp`、`isUserError` 等旧实现

## 8. 去除冗余错误打印

**原则**：cmd 调用的函数在返回 error 时，不应单独打印该错误；错误由 main.go 统一输出。

**需检查范围**：util 包中**返回 error** 的被 cmd 调用的函数。若在 return error 前有 `log.Errorf`/`log.Error` 等打印同一错误的语句，应去掉。**返回 bool 的 util 不修改**，其 log 保留。

**典型模式**：
```go
// 冗余：log 与 return 重复
if err != nil {
    log.Errorf("failed to ...: %v", err)
    return fmt.Errorf("...: %w", err)
}

// 修改后：仅 return，由 main.go 打印
if err != nil {
    return fmt.Errorf("...: %w", err)
}
```

**注意**：`log.Warnf`、`log.Debugf` 等非错误级日志，以及 `reportResultMessages` 等业务输出，是否保留需按场景判断；仅去掉与「返回给 main 的错误」内容重复的 `log.Error`/`log.Errorf`。

## 9. 文件变更清单

| 文件 | 变更 |
|------|------|
| `cmd/root.go` | 新错误类型、构造函数、IsErrorWithUsage；移除 userError 相关；root 设置 SilenceErrors |
| `main.go` | 按错误类型分支输出，移除对 resp.Cmd.SilenceErrors 的依赖 |
| 各 cmd/*.go | 移除 `SilenceErrors: true`；将 `newUserError`/`errExecute` 替换为新 API |
| util/*.go | 仅对返回 error 的函数，去掉与 return 重复的 log.Errorf/log.Error；返回 bool 的 util 不改 |
| `cmd/*_test.go` | 如有断言错误类型，需更新 |

## 10. 待确认项

1. **errExecute 的迁移**：全部改为 `NewStandardErrorF("%v", err)` 传递 util 层原始错误
2. **root 的 SilenceUsage**：保持 `SilenceUsage: true`，是否显示 Usage 由本程序根据错误类型控制

---

## 附录：设计评估与待决策项

### A. 模糊待决策的地方

1. **util 返回 bool 的 API**（check、check-commits、team 等）
   - 现状：`CmdCheckPo()`、`CmdCheckCommits()`、`ShowTeams()` 返回 `bool`，内部 `log.Errorf` 后返回 false；cmd 收到 false 后只能 `return errExecute`，无具体错误可传递
   - **决策**：此次重构暂不修改。cmd 继续 `return errExecute`，迁移为 `NewStandardError("operation failed")` 等通用消息；util 内 log 保留，作为用户获取详细信息的途径

2. **ResolveRevisionsAndFiles 等返回的 err 类型**
   - 可能包含「参数解析错误」（如 invalid range）和「执行错误」（如 checkout 失败）
   - 文档统一归为 NewStandardError。若参数解析错误也希望显示 Usage，需在 util 层区分或返回可识别的 error 类型，当前设计未细化

3. **Cobra root SilenceErrors 的生效范围**
   - 文档写「若 root 级设置不足以覆盖子命令，则采用 init 时批量设置」
   - 待验证：Cobra 中 root 的 SilenceErrors 是否影响子命令错误向上传播时的打印。实现时需先验证，不生效则需批量设置

4. **去除 log 的边界**
   - 文档说「仅去掉与返回给 main 的错误内容重复的 log」
   - **决策**：对返回 bool 的 util，保留其 log（不在此次去除范围内）；仅对返回 error 的 util，去掉与 return 内容重复的 log

### B. 需求与实现合理性

**合理之处：**
- 两种错误类型（ErrorWithUsage / StandardError）语义清晰，覆盖主要场景
- 统一由 main.go 输出，避免 SilenceErrors 与 IsUserError 双重判断
- 去掉 SilenceError 简化设计，所有错误均有内容
- 迁移映射（6.1、6.2）对大部分 cmd 场景有明确指导

**需注意：**
- **errExecute 迁移**：对 util 返回 error 的 cmd，改为 `return NewStandardErrorF("%v", err)` 传递原始错误；对 util 返回 bool 的 cmd（check、check-commits、team），继续 `return errExecute`，迁移为 `NewStandardError("operation failed")` 等通用消息
- **root 无参数**：`return newUserError("run 'git-po-helper -h' for help")` 应改为 `NewErrorWithUsage`，以显示 Usage
