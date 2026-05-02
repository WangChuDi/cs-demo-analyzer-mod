# CS Demo Analyzer (Modified Version)

[🇨🇳 中文文档 / Chinese Documentation](readme_cn.md)


This is a modified fork of the original [CS Demo Analyzer](https://github.com/akiver/cs-demo-analyzer), designed to extract additional "fun" statistics from CS2 demos.

## 🚀 Added Features

### 🧛 Leech & Feed Statistics
Analyze weapon economy interactions between teammates during freezetime:
- **Leech**: Tracks when a player picks up a weapon dropped by a teammate.
  - `LeechCount`: Number of weapons picked up.
  - `LeechValue`: Total monetary value of picked up weapons.
- **Feed**: Tracks when a player drops a weapon that is picked up by a teammate.
  - `FeedCount`: Number of weapons dropped for teammates.
  - `FeedValue`: Total monetary value of dropped weapons.

**Introduced Data Columns:**

- **Players Table (`_players.csv`)**:
  - `leech value`, `leech count`
  - `feed value`, `feed count`

### 💸 Waste Utility Money
Tracks the value of unused utility items (grenades/equipment) held by a player when they die. This represents investment in utility that was lost without being deployed.

**Introduced Data Columns:**

- **Players Table (`_players.csv`)**:
  - `wasted utility value`: Total value of unused utility across the entire match.

- **Player Economy Table (`_players_economy.csv`)**:
  - Breakdown of specific wasted grenades per player per round:
    - `wasted smoke`
    - `wasted flash`
    - `wasted he`
    - `wasted incendiary`
    - `wasted decoy`

### 👣 Footsteps
Tracks player movement sounds (footsteps).

**Introduced Data Columns:**

- **Footsteps Table (`_footsteps.csv`)**:
  - `frame`, `tick`, `round`
  - `x`, `y`, `z` (Player Position)
  - `player velocity x`, `player velocity y`, `player velocity z` (Player Velocity)
  - `yaw`, `pitch` (View Angles)
  - `player name`, `player steamid`, `player team name`, `player side`

### 💨 Grenade Positions
Tracks grenade projectile positions during flight (real-time only).

**Introduced Data Columns:**

- **Grenade Positions Table (`_grenade_positions.csv`)**:
  - `velocity x`, `velocity y`, `velocity z` (Projectile Velocity)
  - `speed` (Projectile Speed)

### 🎮 Player Buttons
Tracks player button presses (Attack, Jump, Duck, etc.) for every tick/update.

**Introduced Data Columns:**

- **Player Buttons Table (`_player_buttons.csv`)**:
  - `buttons`: Bitmask of pressed buttons.
  - `button_names`: Comma-separated list of pressed buttons (e.g. "Attack,Jump").

### 🍀 Unlucky Statistics
Tracks unfortunate events and specific kill/damage circumstances.

**Introduced Data Columns:**

- **Players Table (`_players.csv`)**:
  - `utility damage taken`: Damage received from grenades/molotovs.
  - `team damage taken`: Damage received from teammates.
  - `fall damage taken`: Damage received from falling. In CS2, this is inferred from generic `player_hurt` environment-damage events and filtered against same-frame `BombExplode` events to avoid counting real C4 blast damage as fall damage.
  - `air damage taken`: Damage received from airborne attackers.
  - `run and gun or air killed by count`: Number of times killed by an attacker who was running (moving faster than accurate speed) or airborne.
  - `through smoke kill count`: Number of times killed through smoke.
  - `wallbang kill count`: Number of times killed through walls.
  - `wallbang damage dealt`: Outward wallbang damage dealt (`Damage.IsWallbang`), combining parser-confirmed penetrations when available with heuristic fallback.
  - `wallbang damage taken`: Outward wallbang damage taken (`Damage.IsWallbang`), combining parser-confirmed penetrations when available with heuristic fallback.
  - `true wallbang damage taken`: Parser-confirmed-only wallbang damage taken (`BulletDamage`/`NumPenetrations` same-frame correlation).

- **Shots Table (`_shots.csv`)**:
  - `is player running`: Boolean indicating if the player was moving faster than the weapon's accurate speed threshold when firing.

### 🎯 AWP Hold Deaths
Tracks kills where the victim was holding an AWP angle: scoped, facing the killer within 10 degrees, and stationary or moving slowly enough to be considered in a holding posture.

The current detection windows are intentionally asymmetric:
- a nearby victim AWP shot counts as a pre-death reaction only when it happens within 0.5 seconds before death
- a post-death reaction only looks for an Attack button trigger within 1.0 second after death

The derived event also captures whether the victim had a reaction around the death timing:
- `has pre-death victim awp shot` becomes true when there is a nearby victim AWP shot shortly **before** death.
- `has post-death victim attack trigger` becomes true when there is a post-death Attack button trigger within the configured reaction window.
- `reaction frame` and `offset frame` are only meaningful when `has pre-death victim awp shot` is true.
- In CSV/CSDM exports, attack-only reactions keep `has post-death victim attack trigger = true` while `reaction frame` and `offset frame` remain blank.
- In JSON exports, `victimReactionFrame` and `offsetFrame` are still present as numeric fields and default to `0` when there is no qualifying pre-death victim AWP shot.

**Introduced Data Columns:**

- **Players Table (`_players.csv`)**:
  - `awp hold kill count`: Number of times the player killed an opponent who was holding a scoped AWP angle.
  - `awp hold death count`: Number of times the player was killed while holding a scoped AWP angle.

- **AWP Hold Deaths Table (`_awp_hold_deaths.csv`)**:
  - Killer / victim identity fields.
  - Killer / victim positions.
  - Killer / victim velocity vectors and 2D speed buckets.
  - Killer weapon.
  - Reaction indicators: `has pre-death victim awp shot`, `has post-death victim attack trigger`, `reaction frame`, and `offset frame`.
  - Victim posture qualifiers: `is victim slow`, `is victim scoped`, `is victim facing killer`.

### 🤡 Clown Moments
Tracks embarassing or counter-productive moments from the perpetrator's perspective (e.g., attacking teammates).

**Introduced Data Columns:**

- **Players Table (`_players.csv`)**:
  - `team attack damage`: Health damage dealt to teammates (excluding utility).
  - `team utility damage`: Health damage dealt to teammates using grenades.
  - `team flash duration`: Total duration teammates were blinded by the player's flashes.

### 🛑 Counter-Strafing Success Rate
Tracks how often a player successfully stops before firing their first shot.

**Introduced Data Columns:**

- **Players Table (`_players.csv`)**:
  - `counter-strafing success rate`: Percentage of eligible first shots where the player was no longer running when firing.

- **Shots Table (`_shots.csv`)**:
  - `weapon type`: Weapon category/type for the shot that was fired.
  - `player speed 2d`: Player horizontal movement speed at fire time.
  - `recoil index`: Weapon recoil index captured at fire time.
  - `is player running`: Boolean indicating if the player was moving faster than the weapon's accurate speed threshold when firing.

**Metric Definition:**

- A shot is treated as an eligible first shot when `recoil index == 1`.
- A first shot counts as a successful counter-strafe when `is player running == false`.
- The metric is exported as a single player-level percentage and is not split by weapon class.

### 🎯 First Shot Accuracy
Tracks how often a player's eligible first shots connect with at least one valid enemy damage event.

**Introduced Data Columns:**

- **Players Table (`_players.csv`)**:
  - `first shot count`: Number of eligible first shots fired by the player.
  - `first shot hit count`: Number of eligible first shots that produce at least one valid enemy damage event, counting each first shot at most once.
  - `first shot accuracy`: Percentage of eligible first shots that connect with at least one valid enemy damage event.

**Metric Definition:**

- A shot is treated as an eligible first shot when `recoil index == 1`.
- Only valid enemy player damage events are considered, reusing the same attacker/victim filters as the existing player damage aggregates.
- When direct shot-to-damage linkage is unavailable, the analyzer matches the damage to the nearest prior shot from the same attacker and weapon instance, and only counts it when that matched shot is itself an eligible first shot.
- `first shot hit count` counts each eligible first shot at most once, even if that shot causes multiple valid damage events.
- `first shot accuracy` is `first shot hit count / first shot count * 100`.


### 🧨 Utility Throw Analysis
Detailed analysis of grenade throws, extracting thrower state, button inputs, throw strength, and more.

#### How It Works

The utility throw analysis involves three core systems: **velocity calculation**, **button detection**, and **throw classification**.

**1. Thrower Velocity Calculation (`utils.go: getPlayerVelocity`)**

Uses position-delta method to calculate player velocity. Engine properties (`m_vecVelocity`/`m_vecBaseVelocity`) are unreliable.

- Primary path: `velocity = (currentPos - lastPos) / (tickDelta * tickTime)`
- Fallback path: When `currentPos == lastPos` (due to engine entity update ordering — grenade projectile entity is created before player pawn position is updated in the same frame), uses `(lastPos - prevPos)` instead
- Maintains two-frame position history per player (`lastPlayersPosition`/`prevPlayersPosition`) and corresponding ticks (`lastPlayersTick`/`prevPlayersTick`) in the `FrameDone` handler
- Position history rotation (`prev = last`) only occurs when the tick actually changes, preventing duplicate-tick frames from corrupting the prev value
- Position and tick maps are initialized at round start via `initLastPlayersPosition()`

**2. Button Detection System (`utility.go: applyUtilityThrowButtons`)**

Detects buttons pressed during the throw (Attack/Attack2/Jump/WASD/Walk), output as individual boolean fields.

- Button window start: uses the later of `PinPulledTick` or `throwTick - 0.5s` as the window start
  - `PinPulledTick`: computed from the weapon entity's `m_fPinPullTime` property, converted back to a tick number
  - 0.5s window: fixed lookback of `tickRate/2` ticks as fallback
- Scans all `match.PlayerButtons` records within the window, accumulating via OR to get the final state
- `HasJump` is ultimately overridden by the engine property `m_bJumpThrow` (i.e. `IsJumpThrow`) for accuracy

**3. Throw Classification (`utility.go`)**
- **MouseTypeByStrength**: classifies mouse action via `m_flThrowStrength` property
  - `1.0` → `left_click`, `0.5` → `double_click` (both buttons simultaneously)
  - When value is `0` (game bug, unassigned), falls back to `calcThrowStrength()`:
    `strength = sqrt((initVelX - 1.25*throwerVelX)² + (initVelY - 1.25*throwerVelY)²) / cos(pitch)`
    Thresholds: >500 = `left_click`, >300 = `double_click`, else = `right_click`
- **ThrowerSpeedType**: classifies thrower movement state by `speed2D`
  - `standing`: speed2D == 0
  - `step`: 0 < speed2D < 80
  - `walk`: 80 ≤ speed2D < 180
  - `run`: speed2D ≥ 180

#### Event Flow
```
WeaponFire event
 → newUtilityFromShot(): creates Utility, extracts m_fPinPullTime from weapon entity to compute PinPulledTick
 → applyUtilityThrowButtons(): first button detection (isJumpThrow unknown)

GrenadeProjectileThrow event
 → populates projectile data: IsJumpThrow, ThrowStrength, InitialVelocity, InitialPosition
 → applyUtilityThrowButtons(): second button detection (with complete info)
 → HasJump = IsJumpThrow (engine property overrides button scan)
 → classifyThrowTypeByStrength(): computes MouseTypeByStrength
```

#### Data Columns (`_data_utility.csv`)
**Button State (boolean):**
- `has attack`: Left click (Attack)
- `has attack2`: Right click (Attack2)
- `has jump`: Jump
- `has forward`: W (Forward)
- `has back`: S (Back)
- `has move left`: A (Move Left)
- `has move right`: D (Move Right)
- `has walk`: Shift (Walk)

**Pin Pull & Throw Classification:**
- `pin pulled tick`: Tick when the pin was pulled (computed from `m_fPinPullTime`, 0 = not detected)
- `mouse type by strength`: Mouse action classification (`left_click`/`double_click`/`right_click`)
- `is jump throw`: Whether it's a jump throw (engine property `m_bJumpThrow`)
- `throw strength`: Throw strength (engine property `m_flThrowStrength`, 0 = game bug)

**Thrower State:**
- `thrower velocity x/y/z`: Thrower velocity components
- `thrower speed 2d`: Horizontal speed (used for movement state classification)
- `thrower speed type`: Movement state (`standing`/`step`/`walk`/`run`)
- `thrower yaw`, `thrower pitch`: View direction
**Projectile Data:**
- `initial velocity x/y/z`, `initial speed`: Grenade initial velocity
- `initial position x/y/z`: Grenade initial position

#### Known Edge Cases
- Engine entity update ordering: grenade `WeaponFire` events fire during projectile entity creation (`datatables.go`), before `FrameDone` and player position updates, requiring the velocity fallback path
- Duplicate-tick frames: `FrameDone` may fire multiple times for the same tick, requiring a tick-change guard to prevent position history corruption
- Tick gaps: demos may have tick jumps (e.g. 31622→31624), must use actual tick delta for time interval calculation
- `m_flThrowStrength` == 0: game bug leaves it unassigned, uses `calcThrowStrength()` formula as fallback
- `m_fPinPullTime` unavailable: some demos or bots may lack this property, falls back to 0.5s fixed window

#### Related Files
- `pkg/api/utility.go` — Utility struct, all classification functions, button detection logic
- `pkg/api/utils.go` — `getPlayerVelocity()` position-delta velocity calculation
- `pkg/api/analyzer.go` — FrameDone handler (position/tick rotation), WeaponFire/GrenadeProjectileThrow event handlers
- `pkg/api/match.go` — Position/tick history maps in Match struct
- `pkg/api/export_csv.go` — CSV export

---

### Usage

Ready-to-use binaries are available on the [releases page](https://github.com/WangChuDi/cs-demo-analyzer-mod/releases).


## ⚠️ Known Issues

Due to limitations in `demoinfocs-golang v5` regarding reliable velocity property access for some events (specifically `HeGrenadeExplode` and `SmokeStart`), we have implemented a manual velocity calculation workaround using player position deltas between ticks.

While this workaround provides accurate velocity for most events (like `Footstep` and `Shot`), there might still be edge cases where velocity is unavailable (e.g., first tick of the match).

This affects only:
- `HeGrenadeExplode` (thrower velocity)
- `SmokeStart` (thrower velocity)

### Unlucky Statistics
- **Wallbang Damage**: Outward wallbang reporting (`Damage.IsWallbang`, `wallbang damage dealt`, `wallbang damage taken`) combines parser-confirmed penetration signals when available with a heuristic fallback path. This fallback is needed because current demos/parser signals do not reliably expose direct non-lethal penetration extraction in all cases. When the parser-confirmed path is unavailable, the heuristic matches each damage to the nearest prior shot from the same attacker/weapon and to the nearest prior victim position in the same round, estimates the expected non-wallbang health damage from the weapon model (`BaseDamage`, `RangeModifier`, `ArmorRatio`, `HeadMultiplier`), hitgroup, distance falloff, and victim armor/helmet state, and marks the damage as wallbang when the observed health damage is meaningfully lower than that expected non-wallbang estimate. `true wallbang damage taken` remains parser-confirmed only (`BulletDamage`/`NumPenetrations` same-frame correlation).
- **CS2 Fall Damage**: `demoinfocs-golang v5` can report typed `PlayerHurt.Weapon` as `C4` for true world/fall damage in some CS2 demos. To avoid that, the analyzer currently classifies CS2 fall damage from generic `player_hurt` environment-damage events and excludes candidates that occur on the same frame as `BombExplode`.


## 📝 TODO

### 🤡 Clown Moments
- [ ] 💩 **Failed Utility** (Missed smokes/flashes, bad throws)


## How to build

1. Clone the repository
2. Run `$env:CGO_ENABLED=0; go build -ldflags="-s -w" -trimpath -o csda_mod.exe ./cmd/cli` in cmd.

<details><summary><h2>Original Documentation</h2></summary>

### Usage

Ready-to-use binaries are available on the [releases page](https://github.com/akiver/cs-demo-analyzer/releases).

#### Options

```
csda -help

Usage of csda:
  -demo-path string
        Demo file path (mandatory)
  -format string
        Export format, valid values: [csv,json,csdm] (default "csv")
  -minify
        Minify JSON file, it has effect only when -format is set to json
  -output string
        Output folder or file path, must be a folder when exporting to CSV (mandatory)
  -positions
        Include entities (players, grenades...) positions (default false)
  -source string
        Force demo's source, valid values: [challengermode,ebot,esea,esl,esportal,faceit,fastcup,5eplay,perfectworld,popflash,valve]
```

#### Examples

Export a demo into CSV files in the current folder.

`csda -demo-path=myDemo.dem -output=.`

Export a demo in a specific folder into a minified JSON file including entities positions.

`csda -demo-path=/path/to/myDemo.dem -output=/path/to/folder -format=json -positions -minify`

### API

#### GO API

This API exposes functions to analyze/export a demo using the Go language.

##### Analyze

This function analyzes the demo located at the given path and returns a `Match`.

```go
package main

import (
	"fmt"
	"os"

	"github.com/akiver/cs-demo-analyzer/pkg/api"
	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
)

func main() {
	match, err := api.AnalyzeDemo("./myDemo.dem", api.AnalyzeDemoOptions{
		IncludePositions: true,
		Source:           constants.DemoSourceValve,
	})

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for _, kill := range match.Kills {
		fmt.Printf("(%d): %s killed %s with %s\n", kill.Tick, kill.KillerName, kill.VictimName, kill.WeaponName)
	}
}
```

##### Analyze and export

This function analyzes and exports a demo into the given output path.

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/akiver/cs-demo-analyzer/pkg/api"
	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
)

func main() {
	exePath, _ := os.Executable()
	outputPath := filepath.Dir(exePath)
	err := api.AnalyzeAndExportDemo("./myDemo.dem", outputPath, api.AnalyzeAndExportDemoOptions{
		IncludePositions: false,
		Source:           constants.DemoSourceValve,
		Format:           constants.ExportFormatJSON,
		MinifyJSON:       true,
	})

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println("Demo analyzed and exported in " + outputPath)
}
```

#### CLI

This API exposes the command-line interface.

```go
package main

import (
	"os"

	"github.com/akiver/cs-demo-analyzer/pkg/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
```

#### Node.js API

A Node.js module called `@akiver/cs-demo-analyzer` is available on NPM.  
It exposes a function that under the hood is a wrapper around the Go CLI.  
The module also exports TypeScript types and constants.

```js
import { analyzeDemo, DemoSource, ExportFormat } from '@akiver/cs-demo-analyzer';

async function main() {
  await analyzeDemo({
    demoPath: './myDemo.dem',
    outputFolderPath: '.',
    format: ExportFormat.JSON,
    source: DemoSource.Valve,
    analyzePositions: false,
    minify: false,
    onStderr: console.error,
    onStdout: console.log,
    onStart: () => {
      console.log('Starting!');
    },
    onEnd: () => {
      console.log('Done!');
    },
  });
}

main();
```

### Developing

#### Requirements

- [Go](https://golang.org/dl/)
- [Make](https://www.gnu.org/software/make/)

#### Build

##### Windows

`make build-windows`

##### macOS

`make build-darwin` / `make build-darwin-arm64`

##### Linux

`make build-linux` / `make build-linux-arm64`

#### Tests

1. `./download-demos.sh` it will download the demos used for the tests
2. `make test`

#### VSCode debugger

1. Inside the `.vscode` folder, copy/paste the file `launch.template.json` and name it `launch.json`
2. Place a demo in the `debug` folder
3. Update the `-demo-path` argument to point to the demo you just placed
4. Adjust the other arguments as you wish
5. Start the debugger from VSCode

### Acknowledgements

This project uses the demo parser [demoinfocs-golang](https://github.com/markus-wa/demoinfocs-golang) created by [@markus-wa](https://github.com/markus-wa) and maintained by him and [@akiver](https://github.com/akiver).

### License

[MIT](https://github.com/akiver/cs-demo-analyzer/blob/main/LICENSE.md)

</details>
