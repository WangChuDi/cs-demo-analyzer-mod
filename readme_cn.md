# CS Demo Analyzer（修改版）

[🇬🇧 English Documentation](README.md)

这是 [CS Demo Analyzer](https://github.com/akiver/cs-demo-analyzer) 的修改分支，用于从 CS2 demo 中提取额外的趣味统计数据。

## 🚀 新增功能

### 🧛 吸血与发枪统计
分析冻结时间内队友之间的武器经济互动：
- **吸血（Leech）**：追踪玩家拾取队友丢弃武器的行为。
  - `LeechCount`：拾取武器数量。
  - `LeechValue`：拾取武器的总价值。
- **发枪（Feed）**：追踪玩家丢弃武器被队友拾取的行为。
  - `FeedCount`：为队友丢弃的武器数量。
  - `FeedValue`：丢弃武器的总价值。

**数据列：**

- **玩家表 (`_players.csv`)**：
  - `leech value`、`leech count`
  - `feed value`、`feed count`

### 💸 浪费的道具价值
追踪玩家死亡时手中未使用的道具（手雷/装备）价值，代表浪费的道具价值。

**数据列：**

- **玩家表 (`_players.csv`)**：
  - `wasted utility value`：整场比赛中浪费的道具总价值。

- **玩家经济表 (`_players_economy.csv`)**：
  - 每回合每玩家浪费的具体手雷明细：
    - `wasted smoke`
    - `wasted flash`
    - `wasted he`
    - `wasted incendiary`
    - `wasted decoy`

### 👣 脚步声
追踪玩家移动声音（脚步声）。

**数据列：**

- **脚步声表 (`_footsteps.csv`)**：
  - `frame`、`tick`、`round`
  - `x`、`y`、`z`（玩家位置）
  - `player velocity x`、`player velocity y`、`player velocity z`（玩家速度）
  - `yaw`、`pitch`（视角方向）
  - `player name`、`player steamid`、`player team name`、`player side`

### 💨 手雷位置
追踪手雷弹体飞行中的位置（仅实时模式）。

**数据列：**

- **手雷位置表 (`_grenade_positions.csv`)**：
  - `velocity x`、`velocity y`、`velocity z`（弹体速度）
  - `speed`（弹体速率）

### 🎮 玩家按键
追踪每个 tick/更新中的玩家按键状态（Attack、Jump、Duck 等）。

**数据列：**

- **玩家按键表 (`_player_buttons.csv`)**：
  - `buttons`：按键位掩码。
  - `button_names`：按下的按键名称列表（如 "Attack,Jump"）。

### 🍀 倒霉蛋统计
追踪不幸事件和特定击杀/伤害情况。

**数据列：**

- **玩家表 (`_players.csv`)**：
  - `utility damage taken`：受到的手雷/燃烧弹伤害。
  - `team damage taken`：受到的队友伤害。
  - `fall damage taken`：受到的坠落伤害。
  - `air damage taken`：受到的空中攻击者伤害。
  - `run and gun or air killed by count`：被跑打或空中攻击者击杀的次数。
  - `through smoke kill count`：被透烟击杀的次数。
  - `wallbang kill count`：被穿墙击杀的次数。
  - `wallbang damage taken`：受到的穿墙伤害（注：非致命穿墙伤害可能因事件数据不足而无法追踪）。

- **射击表 (`_shots.csv`)**：
  - `is player running`：玩家开枪时是否在移动（速度超过武器精准速度阈值）。

### 🤡 神人时刻
从施害者角度追踪尴尬或适得其反的行为（如攻击队友）。

**数据列：**

- **玩家表 (`_players.csv`)**：
  - `team attack damage`：对队友造成的伤害（不含道具）。
  - `team utility damage`：使用手雷对队友造成的伤害。
  - `team flash duration`：闪光弹致盲队友的总时长。


### 🧨 道具投掷分析
对道具投掷进行详细分析，提取投掷者状态、按键输入、投掷力度等信息。

#### 原理概述

道具投掷分析涉及三个核心系统：**速度计算**、**按键检测**、**投掷分类**。

**1. 投掷者速度计算 (`utils.go: getPlayerVelocity`)**

使用位置差分法（position-delta）计算玩家速度，而非引擎属性（`m_vecVelocity`/`m_vecBaseVelocity` 不可靠）。

- 主路径：`velocity = (currentPos - lastPos) / (tickDelta * tickTime)`
- 回退路径：当 `currentPos == lastPos` 时（因引擎实体更新顺序问题，手雷弹体创建先于玩家位置更新），使用 `(lastPos - prevPos)` 计算
- 在 `FrameDone` handler 中维护每个玩家的两帧位置历史（`lastPlayersPosition`/`prevPlayersPosition`）和对应 tick（`lastPlayersTick`/`prevPlayersTick`）
- 仅在 tick 实际变化时才轮转 `prev = last`，防止重复 tick 帧覆盖 prev
- Round 开始时通过 `initLastPlayersPosition()` 初始化位置和 tick 映射

**2. 按键检测系统 (`utility.go: applyUtilityThrowButtons`)**

检测投掷时玩家按下的按键（Attack/Attack2/Jump/WASD/Walk），输出为独立布尔字段。

- 按键窗口起点：取 `max(PinPulledTick, throwTick - 0.5s)` 中更近的
  - `PinPulledTick`：通过武器实体的 `m_fPinPullTime` 属性回算拉环时刻的 tick
  - 0.5s 窗口：固定回溯 `tickRate/2` 个 tick 作为 fallback
- 扫描 `match.PlayerButtons` 中该窗口内的所有按键记录，累积 OR 得到最终状态
- `HasJump` 最终被引擎属性 `m_bJumpThrow`（即 `IsJumpThrow`）覆盖，确保准确性
**3. 投掷分类 (`utility.go`)**
- **MouseTypeByStrength**：通过 `m_flThrowStrength` 属性分类鼠标操作
  - `1.0` → `left_click`，`0.5` → `double_click`（左右键同时）
  - 当值为 `0`（游戏 bug 未赋值）时，使用 `calcThrowStrength()` 回算：
    `strength = sqrt((initVelX - 1.25*throwerVelX)² + (initVelY - 1.25*throwerVelY)²) / cos(pitch)`
    阈值：>500 = `left_click`，>300 = `double_click`，其余 = `right_click`
- **ThrowerSpeedType**：根据 `speed2D` 分类投掷者移动状态
  - `standing`：speed2D == 0
  - `step`：0 < speed2D < 80
  - `walk`：80 ≤ speed2D < 180
  - `run`：speed2D ≥ 200
#### 事件流程
```
WeaponFire 事件
 → newUtilityFromShot(): 创建 Utility, 提取武器实体的 m_fPinPullTime 计算 PinPulledTick
 → applyUtilityThrowButtons(): 首次按键检测 (isJumpThrow 未知)

GrenadeProjectileThrow 事件
 → 补充 projectile 数据: IsJumpThrow, ThrowStrength, InitialVelocity, InitialPosition
 → applyUtilityThrowButtons(): 二次按键检测 (使用完整信息)
 → HasJump = IsJumpThrow (引擎属性覆盖按键扫描)
 → classifyThrowTypeByStrength(): 计算 MouseTypeByStrength
```
#### 数据列 (`_data_utility.csv`)
**按键状态（布尔）：**
- `has attack`: 左键（Attack）
- `has attack2`: 右键（Attack2）
- `has jump`: 跳跃
- `has forward`: W（前进）
- `has back`: S（后退）
- `has move left`: A（左移）
- `has move right`: D（右移）
- `has walk`: Shift（静步）
**拉环与投掷分类：**
- `pin pulled tick`: 拉环时刻的 tick 号（通过 `m_fPinPullTime` 回算，0 表示未检测到）
- `mouse type by strength`: 鼠标操作分类（`left_click`/`double_click`/`right_click`）
- `is jump throw`: 是否为跳投（引擎属性 `m_bJumpThrow`）
- `throw strength`: 投掷力度（引擎属性 `m_flThrowStrength`，0 为游戏 bug）
**投掷者状态：**
- `thrower velocity x/y/z`: 投掷者速度分量
- `thrower speed 2d`: 水平面速度（用于分类移动状态）
- `thrower speed type`: 移动状态（`standing`/`step`/`walk`/`run`）
- `thrower yaw`, `thrower pitch`: 视角方向
**弹道数据：**
- `initial velocity x/y/z`, `initial speed`: 道具初始速度
- `initial position x/y/z`: 道具初始位置
#### 已知边界情况
- 引擎实体更新顺序：手雷 `WeaponFire` 事件在弹体实体创建时触发（`datatables.go`），早于 `FrameDone` 和玩家位置更新，导致速度计算需要回退路径
- 重复 tick 帧：`FrameDone` 可能对同一 tick 触发多次，需要 tick 变化守卫防止位置历史被覆盖
- tick 间隙：demo 中可能存在 tick 跳跃（如 31622→31624），必须使用实际 tick 差计算时间间隔
- `m_flThrowStrength` 部分道具为 0：游戏 bug 导致未赋值，使用 `calcThrowStrength()` 公式回算
- `m_fPinPullTime` 不可用：部分 demo 或 bot 可能没有该属性，fallback 到 0.5s 固定窗口
#### 相关文件
- `pkg/api/utility.go` — Utility 结构体、所有分类函数、按键检测逻辑
- `pkg/api/utils.go` — `getPlayerVelocity()` 位置差分速度计算
- `pkg/api/analyzer.go` — FrameDone handler（位置/tick 轮转）、WeaponFire/GrenadeProjectileThrow 事件处理
- `pkg/api/match.go` — Match 结构体中的位置/tick 历史映射
- `pkg/api/export_csv.go` — CSV 导出
---
### 使用方法
预编译的二进制文件可在 [releases 页面](https://github.com/WangChuDi/cs-demo-analyzer-mod/releases) 下载。
## ⚠️ 已知问题
由于 `demoinfocs-golang v5` 在某些事件（特别是 `HeGrenadeExplode` 和 `SmokeStart`）中无法可靠地获取速度属性，我们使用了基于玩家位置差分的手动速度计算方案。
该方案对大多数事件（如 `Footstep` 和 `Shot`）能提供准确的速度，但在某些边界情况下速度可能不可用（如比赛第一个 tick）。
仅影响：
- `HeGrenadeExplode`（投掷者速度）
- `SmokeStart`（投掷者速度）
### 倒霉统计
- **穿墙伤害**：非致命穿墙伤害可能因事件数据不足而无法正确追踪。
## 📝 TODO
### 🤡 小丑时刻
- [ ] 💩 **失败的道具**（没扔好的烟雾弹/闪光弹）
## 构建方法
1. 克隆仓库
2. 运行 `$env:CGO_ENABLED=0; go build -ldflags="-s -w" -trimpath -o csda_mod.exe ./cmd/cli`