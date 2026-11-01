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
  - `fall damage taken`：受到的坠落伤害。在 CS2 中，该值通过 generic `player_hurt` 的环境伤害事件推断，并结合同 frame 的 `BombExplode` 事件过滤真实 C4 爆炸伤害，避免将爆炸伤害误记为坠落伤害。
  - `air damage taken`：受到的空中攻击者伤害。
  - `run and gun or air killed by count`：被跑打或空中攻击者击杀的次数。
  - `through smoke kill count`：被透烟击杀的次数。
  - `wallbang kill count`：被穿墙击杀的次数。
  - `wallbang damage dealt`：造成的对外穿墙伤害（基于 `Damage.IsWallbang`，可用时使用解析器确认信号，否则回退启发式判定）。
  - `wallbang damage taken`：受到的对外穿墙伤害（基于 `Damage.IsWallbang`，可用时使用解析器确认信号，否则回退启发式判定）。
  - `true wallbang damage taken`：仅统计解析器确认的穿墙承伤（BulletDamage、PlayerBulletHit 或唯一匹配的正穿透击杀）。

- **射击表 (`_shots.csv`)**：
  - `is player running`：玩家开枪时是否在移动（速度超过武器精准速度阈值）。

### 🎯 对位处刑／被对位处刑（行为估计）

玩家 JSON 新增 `awpPeekKillCount`（**对位处刑**，主动搜点者的击杀次数）和 `awpPeekDeathCount`（**被对位处刑**，架枪者的死亡次数）。按整场统计，同一条敌方 AWP 对 AWP 击杀分别记在双方名下。`awp_peek_v1` 的规则为：

- 守方持续持同一把 AWP，已验证的开镜架枪窗口至少 **0.5 秒**，水平速度 **≤10 单位/秒**、位置漂移 **≤8 单位**、三维瞄准方向漂移 **≤5°**。原架枪方向与攻方击杀时眼睛位置的三维夹角 **≤10°**，包含高低差。
- 攻方在开枪前 **0.5 秒**持续持同一把 AWP，净水平位移 **≥16 单位**，最后一次水平速度超过 **10 单位/秒**距开枪 **≤0.3 秒**。允许急停后静止开枪，不要求攻方开镜。
- 关联唯一的同 demo tick、同帧、同武器 ID 的 AWP 开火事件。守方若在此前 **0.3 秒**内开过枪，可用其开枪前的开镜架枪窗口；开枪后直到死亡仍须保持位置和瞄准方向，允许有记录的开枪导致退镜。`holderFiredBeforeDeath` 只表示开过枪，不声称一定打空或没有伤害。

排除队杀、自杀、bot／控 bot、穿烟、解析器确认的穿墙，以及检查窗口中的腾空、致盲干扰。重生、死亡、回合、换武器或采样间隔 **>0.125 秒**会清理历史。眼睛位置、活动武器等状态缺失、历史不完整、开火事件缺失或存在歧义时保留未知。`hasAwpPeekData=false` 表示无法判定；`awpPeekReason` 区分不可用、已排除和未达门槛；已知非 AWP 对局使用 `not_awp_duel`。

击杀 JSON 新增 `hasAwpPeekData`、`isAwpPeekKill`、`awpPeekReason`，命中时 `awpPeek` 保留模型、射击／架枪窗口 tick、已验证的架枪窗口时长、速度／位移／角度、搜点净位移、停止移动到开枪的间隔及守方开火标记。CSV 新增 `awp peek kill count`、`awp peek death count`，以及对应标记、原因和 `awp peek ...` 明细列；缺少证据的明细留空。不依赖 `-positions`，固定列 CSDM 不变。

这是依据轨迹与瞄准方向作出的行为估计，不进行地图遮挡射线检测，也不推断首次可见时刻、客户端画面、反应时间或 peek 优势。

### ⚔️ 以下犯上／我软脚了

玩家 JSON 新增 `weaponUpsetKillCount`（以下犯上）和对称的 `weaponUpsetDeathCount`（我软脚了）。统计敌方击杀，按受害者**死亡瞬间的活动武器**判断，不用背包中另一把枪代替：

- 手枪（包含默认手枪）击杀持主武器者，主武器包含步枪、狙击枪、冲锋枪、霰弹枪和机枪，不限距离。
- 冲锋枪击杀持步枪者，且该冲锋枪的模型距离衰减达到 **25%**，即伤害保留比例 **≤75%**，包含等号。按 `保留比例 = rangeModifier^(游戏单位距离/500)` 计算，各枪型门槛为 `500 * log(0.75) / log(rangeModifier)`。例如 MP9 约 **1032.88 单位**、MAC-10 约 **644.61 单位**、UMP-45 为 **500 单位**；MP5-SD／MP7 按 demo 协议版本选取参数。

距离使用击杀事件时双方 XYZ 坐标的三维直线距离估计，保留游戏单位，不复用或改动已有 `distance` 字段。已审计模型支持 CS2 协议 14005–14192；需要的坐标、版本模型或活动武器数据缺失时保留未知。排除队杀、自杀和任一方控 bot 的情况。这是按武器类别及距离定义的规则，不代表完整对枪胜率判断。

击杀 JSON 新增 `hasWeaponUpsetData`、`isWeaponUpsetKill`，可用时 `weaponUpset` 保留 `rule`（`pistol_vs_primary`／`smg_vs_rifle_at_range`）、`distanceUnits`、`rangeThresholdUnits`、`rangeModifier`、`damageRetention`、`damageRetentionThreshold`、`modelSource`。冲锋枪对步枪即使未到门槛也保留衰减明细，是否计数须读标记。CSV 对应 `weapon upset kill count`、`weapon upset death count`、`has weapon upset data`、`is weapon upset kill` 和 `weapon upset ...` 明细列；不可用的数值明细留空。不依赖 `-positions`，固定列 CSDM 不变。

### 💥 自己闪自己

玩家 JSON 新增 `selfFlashCount`、`selfFlashDuration`，CSV 对应 `self flash count`、`self flash duration`。闪光者与被闪者的非零 SteamID 均等于该玩家，且事件有效时长为正、有限数时，计一次自闪；不设 1 秒或全白门槛，排除控 bot 事件。复核明细已包含在原有闪光事件导出中。

时长为**各自闪事件记录的有效致盲时长合计，单位秒**。连续闪光可能覆盖客户端状态，因此这个合计不是重叠区间去重后的白屏时间，也不是全白总时长。不依赖 `-positions`，固定列 CSDM 不变。

### 🔫 使用武器数量

玩家 JSON 新增 `usedWeaponCount`、`usedPrimaryWeaponCount`、`usedSecondaryWeaponCount` 和排序后的 `usedWeapons` 列表，统计整场**实际开火过的不同枪型**，包含主武器及副武器，统一排除 Glock-18、USP-S、P2000。重复开火、拾取、皮肤或同型号不同枪支不重复计数；刀、电击枪、道具、C4 及控 bot 时开火不计。CSV 使用对应的空格分隔列名，`used weapons` 中各枪型用分号分隔。

### 👀 召唤敌人（行为估计）

玩家 `summonEnemyDeathCount` 统计稳定架枪后，转头／切换装备随即被敌人击杀的次数。要求此前至少 **1 秒**持枪、在地面且未致盲，速度不超过 **34 单位/秒**、位置漂移不超过 **16 单位**、朝向漂移不超过 **10°**，期间没有开火或受到伤害。切换武器实例（包括副武器、刀、道具、C4），或朝向开始偏离原架枪方向后，进入 **0.5 秒**死亡窗口；仅转头时，还须在存活采样及死亡时均偏转至少 **60°**。击杀者须位于原架枪朝向的 **30°**以内。恢复原架枪状态、实际开火／投出道具、采样缺失、状态无效或换生命／回合会清理候选；同 tick 动作和控 bot 不计。

击杀 JSON 新增 `isSummonEnemyDeath`，命中时通过 `summonEnemy` 导出触发 tick、死亡间隔、架枪时长、原因（`turn_away`／`weapon_switch`）、切换前后武器及角度。CSV 使用 `is summon enemy death` 和 `summon enemy ...` 列，无明细时留空。这是行为模型，不能证明敌人此前不可见或玩家的主观意图；不依赖 `-positions`。

### 🦺 忘买甲

玩家 `forgotArmorCount` 统计**冻结结束时持有非狙击主武器且护甲为 0**的回合数。步枪、冲锋枪、霰弹枪、机枪符合条件；AWP、SSG 08、G3SG1、SCAR-20 和仅持手枪的情况排除。上一回合继承的剩余护甲也算已有甲。这是冻结结束的装备快照，包含刻意无甲的选择；之后补买或拾枪不会重写该快照。

玩家经济 JSON 新增 `hasArmorCheckData`、`armorCheckTick`、`armorAtFreezeEnd`、`primaryWeaponAtFreezeEnd`、`forgotArmor`，CSV 使用对应的空格分隔列名。护甲数据缺失、主武器清单有歧义、死亡或 bot 玩家不判定。快照会保留至最终导出，不受后续经济刷新或比赛结束时重建记录影响。这三项新增 JSON／CSV 字段，固定列 CSDM 保持原布局。

### 🕶️ 全白击杀（模型估计）

玩家 CSV 的 `full white kill count`、玩家 JSON 的 `fullWhiteKillCount` 统计击杀者符合选定 CS2 CPU 白罩模型时的敌方击杀。玩家汇总排除队杀、自杀和非玩家受害者。闪光参数在击杀事件发生时保存，无需启用位置导出。

以有效总时长 D、观测到的 `m_flFlashMaxAlpha` 上限 M 和 `(击杀 tick - FlashTick) / tickrate` 推算经过时间，剩余时间 R = max(0, D - 经过时间)。建立期 B = `0.09444444626569748` 秒内，白罩与残影 Alpha 均为 `clamp(经过时间 / B * M, 0, M) / 255`；建立后白罩 Alpha 为 `(M / 255) * min(1, R / 3)^2`，残影 Alpha 为 `(M / 255) * min(1, R / D)`。假设观战缩放为 1，**已过建立期、M=255 且 R >= 3 秒**时计入全白击杀，包含等号；剩余 2 秒时白罩系数只有 `4/9`。

击杀 JSON 新增可选 `killerFlashEstimate`，包含 `model`、`duration`、`elapsed`、`remaining`、`maxAlpha`、`phase`、`overlayAlpha`、`screenshotAlpha`、`isFullWhite`。击杀 CSV 新增 `has killer flash estimate`、`is killer full white estimated` 和 `killer flash ...` 参数、系数、阶段及模型列。参数缺失或无效时，JSON 不输出该对象，CSV 可用性标记为 0，此时数值零只是占位；实际观测到 Alpha 上限为 0 则属于可用数据。原有致盲布尔字段保留原含义。固定列数的 CSDM 格式保持原布局。

模型 `cs2_client_14194_cpu` 来源为用户提供的 Windows IDA/指令分析：CS2 1.41.9.4、SourceRevision 11191540、client SHA-256 `d618bbf8e6afb36baa09b3a282035996a88300458c55d2aa5cb81f0495c0ec54`。结果为条件估计：尚未重建其他客户端版本的差异、准确客户端时钟/帧相位、观战状态、连续闪光的时长调整、相等值早退和回调瞬态。已分析的是 CPU 参数，GPU shader 和实际像素尚未验证。Cache 样本筛出 2 次：Spinx tick 17129、torzsi tick 83950；不以原始击杀事件的致盲标记代替此模型。

### 💣 掏道具暴毙与亡语击杀

玩家 CSV 新增 `utility in hand death count`、`posthumous kill count`；玩家 JSON 对应 `utilityInHandDeathCount`、`posthumousKillCount`。

- **掏道具暴毙**：死亡事件当下，观测到的活动武器是闪光、烟雾、HE、燃烧瓶、燃烧弹或诱饵。玩家汇总只统计被敌方击杀；仅在背包中携带道具、手持 C4/刀、队杀、自杀和环境死亡均不计入。活动武器缺失或无法识别时保留未知，不用死亡后的位置快照推断。
- **亡语击杀**：上述六种投掷道具造成敌方死亡，击杀者当时已死亡，且同回合存在严格更早 demo tick 的死亡记录；包括 HE/火焰击杀及其他道具罕见的致命碰撞。同 tick 互换（即使解析帧不同）、枪械、C4、队杀、自杀及控 bot 时的击杀不计入。新回合、已记录的重生事件会清理死亡索引；观测到击杀者存活也会使旧死亡记录失效。死后连续击杀多名敌人时保留原死亡 tick；死亡或时钟证据缺失时不判定。

击杀 JSON/CSV 同时导出活动武器及其可用性（`hasVictimActiveWeaponData` / `has victim active weapon data`）、手持道具状态（`isVictimHoldingUtility` / `is victim holding utility`），以及 `isPosthumousKill`、`posthumousDeathTick`、`posthumousDelaySeconds`（CSV：`is posthumous kill`、`posthumous death tick`、`posthumous delay seconds`）。手持道具标记描述受害者状态，玩家汇总另要求敌方击杀；亡语死亡 tick/间隔的零占位需结合亡语标记读取。两项均在事件发生时取值，不依赖 `-positions`。固定列 CSDM 保持原布局。

### 🪨 被投掷道具砸死

玩家 JSON 新增 `utilityImpactDeathCount`，玩家 CSV 新增 `utility impact death count`，统计被敌方投掷道具的弹体碰撞击杀的次数。爆炸、燃烧伤害不计入，玩家汇总排除队杀、自杀和非玩家击杀。

在类型化击杀事件之后立即读取原始 `player_death.weapon`。按标准 CS2 事件命名，闪光、烟雾及燃烧瓶/燃烧弹的弹体名称标识碰撞，`inferno` 标识火焰伤害；不能仅看解析器映射后的道具类型，也不用“致死伤害只有 1 点”猜测。

当原始名称不足以区分 HE／诱饵碰撞时，使用弹体生命周期补充判定：必须观测到投掷，伤害发生时只有一个可匹配弹体，且随后观测到该弹体的 `HeExplode`／`DecoyExpired`。致死伤害与击杀还必须按回合、tick、解析帧、攻击者、受害者和武器类型一一对应，并校验归属、弹体身份和武器实例。效果的 demo tick 与解析帧都必须严格晚于伤害。实体销毁不视为爆炸：HE 爆炸后实体还可能保留五秒。证据缺失、多弹体竞争、同 tick 效果、跨回合证据及控 bot 情况保留未知，因此仍是保守计数。

击杀 JSON 导出 `rawWeaponName`、`hasUtilityImpactData`、`isUtilityImpactKill`，以及来源 `utilityImpactSource`（`raw_weapon`、`projectile_lifecycle` 或 `unavailable`）。生命周期判定另导出 `utilityImpactProjectileId`、`utilityImpactEffectTick`（demo tick）和 `utilityImpactEffect`（`he_explode`／`decoy_expired`）；CSV 使用相应的空格分隔列名。原因未知时，可用性为 false，碰撞标记的 false 也需结合可用性读取；原始通用死亡事件缺失时，仅能通过生命周期证据补充。无需启用 `-positions`，固定列 CSDM 保持原布局。

### 🎯 手撕大狙 / AWP Hold Deaths
追踪被击杀者处于“大狙架点”状态时的击杀事件：被击杀者持有 AWP、处于开镜状态、瞄准方向与击杀者夹角不超过 10 度，并且静止或低速移动。

当前的检测窗口是非对称的：
- 被击杀者的 death 前 AWP shot 只有发生在死亡前 0.5 秒内，才会计为 pre-death reaction
- death 后 reaction 只看死亡后 1.0 秒内的 Attack 按键触发，不再要求出现 shot

该派生事件也会记录被击杀者在死亡前后是否有反应：
- `has pre-death victim awp shot` 为 true，表示被击杀者在死亡前很短时间内有附近的 AWP shot。
- `has post-death victim attack trigger` 为 true，表示被击杀者在死亡后反应窗口内触发了 Attack 按键。
- `reaction frame` 和 `offset frame` 只有在 `has pre-death victim awp shot = true` 时才有实际意义。
- 在 CSV/CSDM 导出中，如果只是 death 后 attack 触发而没有真实的 death 前 reaction shot，则 `has post-death victim attack trigger = true`，同时 `reaction frame` 和 `offset frame` 会保持为空。
- 在 JSON 导出中，即使没有符合条件的 death 前 AWP shot，`victimReactionFrame` 和 `offsetFrame` 仍会以数值字段出现，并通常为 `0`。

**数据列：**

- **玩家表 (`_players.csv`)**：
  - `awp hold kill count`：玩家击杀处于大狙架点状态对手的次数。
  - `awp hold death count`：玩家自己处于大狙架点状态时被击杀的次数。

- **手撕大狙事件表 (`_awp_hold_deaths.csv`)**：
  - 击杀者 / 被击杀者身份信息。
  - 双方位置。
  - 双方速度向量与二维移速分档。
  - 击杀者武器。
  - 反应相关字段：`has pre-death victim awp shot`、`has post-death victim attack trigger`、`reaction frame`、`offset frame`。
  - 被击杀者姿态条件：`is victim slow`、`is victim scoped`、`is victim facing killer`。

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

2026-10-30 同步原版 v1.13.0 后，改用解析器的 `Player.Velocity()`。上游已经修复速度属性解析，并为旧 demo 提供基于快照服务器 tick 的速度推导。构造解析器时启用 `EagerVelocityTracking`，确保手雷创建事件之前已有位置历史。构建要求 Go 1.27+；标准事件导出采用上游的 tick 字段，二开事件关联所需的 frame 仅保留在内部。

依赖版本说明：同步前使用 demoinfocs v5.2.0，同步原版时曾采用带首回合按键补丁的 v4。现在按要求升级为官方 [demoinfocs v6.0.0-rc.1](https://github.com/markus-wa/demoinfocs-golang/tree/v6.0.0-rc.1)（候选发布版，提交 `4c79f7b51239`），不再引用 v4 或本地替换依赖。v6 已原生提供按键专用解码、完整数据包基线播种及速度跟踪。构造器使用 `option.EagerVelocityTracking(true)`，解析需要显式 context，实体和属性回调适配为指针类型。v6 仅支持 CS2；旧 CS:GO demo 需使用此前版本。

**2. 按键检测系统 (`utility.go: applyUtilityThrowButtons`)**

检测投掷时玩家按下的按键（Attack/Attack2/Jump/WASD/Walk），输出为独立布尔字段。

- 按键窗口起点：取 `max(PinPulledTick, throwTick - 0.5s)`（两者中较晚/较大的 tick，即更接近投掷时刻的那个）
  - `PinPulledTick`：武器的 `m_fPinPullTime` 发生有效正值变化时，直接记录当前 demo tick，无需服务器时间换算
  - 0.5s 窗口：固定回溯 `tickRate/2` 个 tick 作为 fallback
- 扫描 `match.PlayerButtons` 中该窗口内的所有按键记录，累积 OR 得到最终状态
- `HasJump` 最终被引擎属性 `m_bJumpThrow`（即 `IsJumpThrow`）覆盖，确保准确性
- v6 从完整数据包内的 usercmd 快照建立基线，再处理正常消息流中的 delta，恢复此前缺失的首回合输入；数据包内用来建立基线的命令不会额外派发事件。未知或不匹配的基线仍告警，命令序号与 ring 校验保持生效。
**3. 投掷分类 (`utility.go`)**
- **MouseTypeByStrength**：通过 `m_flThrowStrength` 属性分类鼠标操作
  - `1.0` → `left_click`，`0.5` → `double_click`（左右键同时）
  - 其他值使用 `calcThrowStrength()` 回算；零值可能未赋值，死亡时释放也可能缺少有效投出速度：
    `strength = sqrt((initVelX - 1.25*throwerVelX)² + (initVelY - 1.25*throwerVelY)²) / cos(pitch)`
    仅对有限正值分类：>500 = `left_click`，>300 = `double_click`，其余 = `right_click`。回算值非正数或非有限值时为 `unknown`，不能据此认定右键。
- **ThrowerSpeedType**：根据 `speed2D` 分类投掷者移动状态
  - `standing`：speed2D == 0
  - `step`：0 < speed2D < 80
  - `walk`：80 ≤ speed2D < 180
  - `run`：speed2D ≥ 180
#### 事件流程
```
武器 m_fPinPullTime 发生有效变化
 → 按武器实例记录当前 demo tick

WeaponFire 事件
 → newUtilityFromShot(): 创建 Utility，读取已记录的 PinPulledTick
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
- `pin pulled tick`: 录像中观测到拔栓的 demo tick（0 表示未观测到，包括实体首次出现时已经拔栓的情况）
- `mouse type by strength`: 鼠标操作分类（`left_click`/`double_click`/`right_click`/`unknown`）
- `is jump throw`: 是否为跳投（引擎属性 `m_bJumpThrow`）
- `throw strength`: 原始引擎属性 `m_flThrowStrength`；不能单凭零值推断鼠标操作
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
- 初始属性值不算新拔栓，重复值不会移动已记录的 tick；属性复位时清除记录。销毁清理推迟到帧末，以保留同帧投出事件需要的数据；使用实体序列号隔离复用的实体编号。
#### 相关文件
- `pkg/api/utility.go` — Utility 结构体、所有分类函数、按键检测逻辑
- `pkg/api/utility_pin.go` — 直接记录拔栓 demo tick，处理武器实例生命周期
- `pkg/api/utils.go` — `getPlayerVelocity()` 位置差分速度计算
- `pkg/api/analyzer.go` — FrameDone handler（位置/tick 轮转）、WeaponFire/GrenadeProjectileThrow 事件处理
- `pkg/api/match.go` — Match 结构体中的位置/tick 历史映射
- `pkg/api/export_csv.go` — CSV 导出
---
### 使用方法
预编译的二进制文件可在 [releases 页面](https://github.com/WangChuDi/cs-demo-analyzer-mod/releases) 下载。
## ⚠️ 已知问题
2026-10-30 审计发现的拔栓时钟问题已通过直接记录观测到的 demo tick 修复。未观测到拔栓变化时仍输出未知值 0。官方 v6.0.0-rc.1 的完整数据包基线播种已恢复本次样本的首回合按键，无需移植 v4 的额外补丁；Aurora–MOUZ Cache 样本首回合有 4,421 条按键状态变化，覆盖全部 10 名玩家及 5 次投掷。不能恢复的输入片段仍保留未知。脚步、准星后坐力和穿透事件若未出现在原始输入中，也应标注不可用，不能把导出的零值当作真实统计。

后续修复了 1vX 汇总的玩家与对手人数过滤，并让 AWP 判定使用击杀事件当时的状态。穿墙启发式直接保存伤害事件时的受害者位置及头盔状态，不再依赖 `-positions`；该开关只控制轨迹导出。样本两种模式均为 2 条 AWP 架枪死亡、424 点启发式穿墙伤害，这些仍是模型判定，未逐条回看录像。

CSV/JSON 提供射击与伤害可用性标记、比赛级样本计数（JSON 的 `dataQuality`），数值必须结合标志读取。穿透来源区分 `bullet_damage`、`player_bullet_hit`、`kill_event`、`heuristic`、`unavailable`；已观测的 BulletDamage（含零）始终优先。正穿透击杀仅关联同回合、帧、双方玩家及武器唯一匹配的致命伤害，不补此前非致命伤害，也不冒充 `has bullet damage data`。样本新增 3 次确认穿透共 123 点致命伤害，与原有 424 点启发式伤害合计 547 点。

脚步新增 SoS 声音恢复：只接受游戏资源中已知 CT/T 脚步声，按 pawn 实体编号映射玩家，帧末与旧游戏事件去重；排除落地声、装备声和未知哈希。`data source`、`sound event name` 标明来源。样本恢复 8,999 条、覆盖 10 人，不保证每份 demo 完整，也不能据此判断某个敌人实际听到了什么。

新版 AimPunchServices 的基准角度、角速度及服务器 tick 单独导出为 `aimPunchBase`（CSV 的 `aim punch ... base ...`），样本 1,899 条。基准状态不等于开火瞬时后坐力。现对 v6.0.0-rc.1 返回的 GameTick_t 无符号线值做 ZigZag 解码；`serverTick` 改为显式网络/快照 tick，并带 `hasServerTickData`。早期 fork 导出误用了 demo tick 和未解码的基准 tick，旧结果需要重导；解码后的基准 tick 只能与有可用标记的服务器时钟比较。

原始流若包含 `FireBullets.Extra.aim_punch`，现可直接恢复瞬时角度，来源 `aimPunchDataSource=fire_bullets`；旧实体属性标为 `entity`，缺失为 `unavailable`。匹配要求回合、帧、玩家和武器唯一，缺字段消息也计入歧义。`PlayerBulletHit.penetration_count`（包括显式零）可补非致命穿透，带 `hasPlayerBulletHitData` 和来源 `player_bullet_hit`；要求同帧唯一 hurt 的双方、伤害及消息提供的命中部位/致命标记一致，已有 BulletDamage 始终优先。Cache 样本没有这些消息，两项精确信号仍不可用。CSV 增加来源及可用标记，固定列 CSDM 不变。

ServerInfo 的 VBKV 配置经 CRC 和完整结构验证后读取活动 `c_game_type/c_game_mode`，来源为 `server_info_data`，不使用地图默认值。缺少证据仍返回 unknown。外部日期可通过 `-match-date '2026-10-29T00:00:00+08:00'` 或 `CSDA_MATCH_DATE` 传入；优先级为有效 `.info`、外部日期、文件修改时间，分别标为 `match_info`、`external_metadata`、`file_mtime`。固定列 CSDM 布局保持兼容；旧数据库结果需重跑分析及导入。

### 倒霉统计
- **穿墙伤害**：对外穿墙统计（`Damage.IsWallbang`、`wallbang damage dealt`、`wallbang damage taken`）优先采用解析器确认信号，否则使用启发式。启发式要求同回合、同攻击者、同武器且同帧唯一的射击，采用伤害事件当时的位置、头盔及护甲，根据武器模型、hitgroup、距离衰减和伤害倍率估算非穿墙伤害。理论值向下取整后与原始伤害报告比较，至少少 2 点才列为候选；差值恰好为 1 的记录保留诊断及原因 `within_one_point_tolerance`，不计入穿墙伤害汇总。缺少事件位置或匹配射击时标记 `unavailable`，不再假定距离为零。`true wallbang damage taken` 仅保留 BulletDamage、PlayerBulletHit 或唯一匹配的正穿透击杀的确认路径。
- **CS2 坠落伤害**：此前使用 `demoinfocs-golang v5` 时，在部分 CS2 demo 中发现 typed `PlayerHurt.Weapon` 会把真实 world/fall 伤害误判为 `C4`。当前保留 generic `player_hurt` 环境伤害判断，并排除与 `BombExplode` 同 frame 的候选伤害。
## 📝 TODO
### 🤡 小丑时刻
- [ ] 💩 **失败的道具**（没扔好的烟雾弹/闪光弹）
## 构建方法
1. 克隆仓库
2. 运行 `$env:CGO_ENABLED=0; go build -ldflags="-s -w" -trimpath -o csda_mod.exe ./cmd/cli`
