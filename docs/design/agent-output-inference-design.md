# Agent Output 自动推断 - 重构设计（接口版）

## 一、问题

当前设计存在冗余与冲突风险：

1. **冗余**：`Output` 与 `Cmd` 中的 format 参数重复表达同一信息
2. **冲突**：用户若同时设置 `Output: "text"` 和 `Cmd` 中含 `--output-format stream-json`，语义矛盾
3. **配置负担**：用户需同时维护 `Cmd` 和 `Output`，易出错

## 二、目标

- **单一数据源**：仅通过 `Cmd` 表达输出格式，移除 `Output` 配置项
- **接口抽象**：通过 Agent 接口统一各 agent 的 Cmd 构建与 OutputFormat 解析
- **按实现分发**：各 agent 在对应 `agent-parse-*.go` 中实现接口

## 三、核心设计

### 3.1 Agent 接口（config/agent.go）

```go
// Agent 接口定义 agent 的命令构建与输出格式解析能力。
// 各 Kind 在 util/agent-parse-*.go 中实现此接口。
type Agent interface {
    // BuildCommand 返回补齐 output format 参数后的完整命令行。
    // vars 用于占位符替换（如 {{.prompt}}, {{.source}}）。
    // 若 config 中的 Cmd 已含 format 参数，则不再追加。
    BuildCommand(vars PlaceholderVars) (cmd []string, err error)

    // GetOutputFormat 根据 config 中原始 Cmd 解析 output format，或返回默认值。
    GetOutputFormat() string
}
```

### 3.2 配置结构

为避免与 `Agent` 接口同名冲突，配置结构体命名为 `AgentEntry`：

```go
// AgentEntry 为配置结构体，对应 YAML 中 agents 下的每个 agent。
type AgentEntry struct {
    Cmd  []string `yaml:"cmd"`
    Kind string   `yaml:"kind"`
    // Output 已移除，由 Cmd 推断
}
```

**工厂函数**：根据 `config.AgentEntry` 的 Kind 实例化对应的 `Agent` 接口实现：

```go
// NewAgentFromConfig 在 util 包实现，返回 config.Agent 接口。
func NewAgentFromConfig(cfg config.AgentEntry) (config.Agent, error)
```

### 3.3 各 Kind 的实现位置与行为

| Kind    | 实现文件           | 实现类型    | outputFormat 默认值 | BuildCommand 补齐逻辑                    |
|---------|--------------------|-------------|------------------------|------------------------------------|
| claude  | agent-parse-claude.go  | claudeAgent | stream-json            | 无 format 时追加 --verbose --output-format stream-json |
| codex   | agent-parse-codex.go   | codexAgent  | stream-json            | 无 --json 时追加 --json            |
| opencode| agent-parse-opencode.go| opencodeAgent | stream-json          | 无 --format 时追加 --format json    |
| gemini  | agent-parse-gemini.go   | geminiAgent | stream-json            | 无 --output-format 时追加 --output-format stream-json |
| qwen    | agent-parse-gemini.go   | geminiAgent | stream-json            | 同 gemini                          |
| qoder   | agent-parse-qoder.go    | qoderAgent  | stream-json            | 无 --output-format/-o/-f 时追加 --output-format stream-json |
| echo    | agent-parse.go 或新建   | echoAgent   | text                   | 不追加                             |

### 3.4 GetOutputFormat 解析规则

从 config 中**原始 Cmd**（占位符未展开）解析：

- **Claude/Gemini/Qoder**：若存在 `--output-format`/`-o`/`-f`，取下一参数；`stream-json`/`stream_json` → `stream-json`，`json` → `json`，`text` → `text`。默认为 `stream-json`。
- **Codex**：若存在 `--json` → `stream-json`，否则 → `text`。默认 `stream-json`。
- **Opencode**：若存在 `--format`，取下一参数；`json` → `stream-json`。默认 `stream-json`。
- **Echo**：始终返回 `text`

### 3.5 cmd 补齐规则（BuildCommand 内部）

1. 对 config.Cmd 做占位符替换得到 `cmd`
2. 若 Cmd 中已含 format 参数（同上检测逻辑），直接返回 `cmd`
3. 否则，按 Kind 追加默认 format 参数后返回

### 3.6 性能优化

3.4 和 3.5 可对 config.Cmd 只解析一次，将 cmd 与 format 缓存，供 BuildCommand 与 GetOutputFormat 复用。

### 3.7 调用流程变更

**当前**：

```go
selectedAgent := SelectAgent(cfg, agentName)  // config.Agent
agentCmd, _ := BuildAgentCommand(selectedAgent, vars)
outputFormat := config.GetAgentOutputFormat(selectedAgent)
RunAgentAndParse(agentCmd, outputFormat, kind)
```

**重构后**：

```go
selectedEntry := SelectAgent(cfg, agentName)  // config.AgentEntry
agent, err := NewAgentFromConfig(selectedEntry)
if err != nil { ... }
agentCmd, err := agent.BuildCommand(vars)
if err != nil { ... }
outputFormat := agent.GetOutputFormat()
RunAgentAndParse(agentCmd, outputFormat, selectedEntry.Kind)
```

`BuildCommand` 返回 cmd 和 error；`outputFormat` 由 `agent.GetOutputFormat()` 获取。可选便捷函数：

```go
// BuildAgentCommand 便捷封装，内部调用 agent.BuildCommand + agent.GetOutputFormat
func BuildAgentCommand(entry config.AgentEntry, vars PlaceholderVars) ([]string, string, error) {
    agent, err := NewAgentFromConfig(entry)
    if err != nil { return nil, "", err }
    cmd, err := agent.BuildCommand(vars)
    if err != nil { return nil, "", err }
    return cmd, agent.GetOutputFormat(), nil
}
```

## 四、实现任务清单（已完成）

### 任务 1：config/agent.go ✓

- 定义 `Agent` 接口（`BuildCommand(vars PlaceholderVars) ([]string, error)`、`GetOutputFormat() string`）
- 定义 `AgentEntry` 配置结构体（Cmd, Kind），移除 `Output` 字段
- 工厂 `NewAgentFromConfig` 在 util 包实现

### 任务 2–7：各 agent 实现 ✓

- `util/agent-parse-claude.go`: `claudeAgent`
- `util/agent-parse-codex.go`: `codexAgent`
- `util/agent-parse-gemini.go`: `geminiAgent`
- `util/agent-parse-opencode.go`: `opencodeAgent`
- `util/agent-parse-qoder.go`: `qoderAgent`
- `util/agent-cmd.go`: `echoAgent`
- `util/agent-impl.go`: `NewAgentFromConfig`

### 任务 8：调用方适配 ✓

- `agent.BuildCommand(vars)` 返回 cmd 和 error；`agent.GetOutputFormat()` 获取 outputFormat；`BuildAgentCommand` 为可选便捷封装
- `SelectAgent` 返回 `config.AgentEntry`
- 所有 agent-run 调用方已更新

### 任务 9：测试与文档 ✓

- `util/agent_impl_test.go`: `TestNewAgentFromConfig`, `TestBuildAgentCommand`

## 五、依赖关系

```
config/agent.go
  - 定义 Agent 接口
  - 定义 AgentEntry 配置结构体（Cmd, Kind）

util/agent-impl.go
  - NewAgentFromConfig(cfg config.AgentEntry) (config.Agent, error)

util/agent-parse-*.go
  - 实现 Agent 接口（claudeAgent, codexAgent 等）
  - 依赖 config 包的 Agent 接口、Output 常量
```

- `config`：定义 `Agent` 接口、`AgentEntry` 配置结构体（避免同名冲突）
- `util`：实现 `NewAgentFromConfig`，返回 `config.Agent` 接口的实现

## 六、接口与配置命名

为避免 `config.Agent` 结构体与 `config.Agent` 接口同名冲突：

- **接口**：`Agent`（用户要求的名称）
- **配置结构体**：`AgentEntry`，YAML 反序列化使用 `agents.<name>` 下的 `cmd`、`kind` 字段

```go
// config/agent.go

// Agent 接口，各 Kind 在 util/agent-parse-*.go 中实现
type Agent interface {
    BuildCommand(vars PlaceholderVars) (cmd []string, err error)
    GetOutputFormat() string
}

// AgentEntry 为配置结构体，对应 YAML 中 agents 下的每个 agent
type AgentEntry struct {
    Cmd  []string `yaml:"cmd"`
    Kind string   `yaml:"kind"`
}

// AgentConfig.Agents 类型改为 map[string]AgentEntry
```

**向后兼容**：若现有代码大量使用 `config.Agent`，可保留 `Agent` 作为结构体类型别名指向 `AgentEntry`，或通过类型别名 `type Agent = AgentEntry` 过渡。
