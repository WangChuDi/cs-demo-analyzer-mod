# CS Demo Analyzer (Modified Version)

[🇨🇳 中文文档 / Chinese Documentation](readme_cn.md)


This is a modified fork of the original [CS Demo Analyzer](https://github.com/akiver/cs-demo-analyzer), designed to extract additional "fun" statistics from CS2 demos.

Synced with upstream v1.13.0 (`8374d4e`) on 2026-10-30. Building requires Go 1.27+. Standard event exports follow upstream's tick-based schema; internal frame information is retained for the fork's event correlation and AWP reaction metrics. Projectile IDs now follow upstream's decimal string format.

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
  - `true wallbang damage taken`: Parser-confirmed-only wallbang damage taken (BulletDamage, PlayerBulletHit, or uniquely matched positive fatal penetration).

- **Shots Table (`_shots.csv`)**:
  - `is player running`: Boolean indicating if the player was moving faster than the weapon's accurate speed threshold when firing.

### 🕶️ Full-White Kills (Estimated)

`full white kill count` in `_players.csv` and `fullWhiteKillCount` in player JSON count enemy kills while the killer meets the selected CS2 CPU white-overlay model. Team kills, suicides and non-player victims are excluded from the player total. Flash inputs are captured at the kill event; no position export is required.

The model estimates elapsed time as `(kill tick - FlashTick) / tickrate` and remaining time `R = max(0, D - elapsed)`, with effective flash duration `D` and the observed `m_flFlashMaxAlpha` cap `M`. During the first `B = 0.09444444626569748` seconds, overlay and screenshot alpha are `clamp(elapsed / B * M, 0, M) / 255`. After buildup, overlay alpha is `(M / 255) * min(1, R / 3)^2`, and screenshot alpha is `(M / 255) * min(1, R / D)`. Spectator scaling is assumed to be 1. A kill qualifies after buildup when **M = 255 and R >= 3 seconds**, including equality. At two seconds, overlay alpha is only `4/9`.

Kill JSON adds the optional `killerFlashEstimate` object with `model`, `duration`, `elapsed`, `remaining`, `maxAlpha`, `phase`, `overlayAlpha`, `screenshotAlpha` and `isFullWhite`. Kill CSV adds `has killer flash estimate`, `is killer full white estimated`, and `killer flash ...` inputs/coefficients/phase/model columns. Missing or invalid inputs omit the JSON object and set the CSV availability flag to 0; numeric zero placeholders then do not indicate measured zero opacity. An observed zero cap is available data. The existing blinded flags keep their original meaning. The fixed-column CSDM layout is unchanged.

Model `cs2_client_14194_cpu` comes from the supplied Windows IDA/instruction analysis of CS2 1.41.9.4, SourceRevision 11191540, client SHA-256 `d618bbf8e6afb36baa09b3a282035996a88300458c55d2aa5cb81f0495c0ec54`. It is a conditional estimate: other client versions, exact client clock/frame phase, spectator view state, repeated-flash duration adjustments, equal-value early returns and callback transients are not reconstructed. CPU parameters have been analyzed; GPU shader output and actual pixels have not been verified. The Cache sample selects two kills: Spinx at tick 17129 and torzsi at tick 83950; neither the event blinded flag nor a generic duration threshold is used as a substitute for this model.

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
- Fallback attribution only considers candidate prior shots within `48` frames of the damage event, which keeps the metric conservative when direct linkage is unavailable.
- `first shot hit count` counts each eligible first shot at most once, even if that shot causes multiple valid damage events.
- `first shot accuracy` is `first shot hit count / first shot count * 100`.


### 🧨 Utility Throw Analysis
Detailed analysis of grenade throws, extracting thrower state, button inputs, throw strength, and more.

#### How It Works

The utility throw analysis involves three core systems: **velocity calculation**, **button detection**, and **throw classification**.

**1. Thrower Velocity Calculation (`utils.go: getPlayerVelocity`)**

Uses the synchronized parser's `Player.Velocity()`. Upstream now resolves the networked velocity properties and derives velocity from snapshot server ticks for older demos. The parser enables `EagerVelocityTracking` so velocity history is available before grenade creation events. This replaces the fork's older manual position-delta workaround.

The fork used demoinfocs v5.2.0 before the upstream v1.13.0 merge, then briefly adopted a patched v4 dependency. It now uses official [demoinfocs v6.0.0-rc.1](https://github.com/markus-wa/demoinfocs-golang/tree/v6.0.0-rc.1), a release candidate at commit `4c79f7b51239`, with no v4 or local replacement dependency. v6 provides button-only decoding, full-packet command baseline seeding, and velocity tracking natively. The migration uses constructor options, context-aware parsing, and pointer entity/property callbacks. v6 supports CS2 only; use an earlier analyzer revision for legacy CS:GO demos.

**2. Button Detection System (`utility.go: applyUtilityThrowButtons`)**

Detects buttons pressed during the throw (Attack/Attack2/Jump/WASD/Walk), output as individual boolean fields.

- Button window start: uses the later of `PinPulledTick` or `throwTick - 0.5s` as the window start
  - `PinPulledTick`: records the demo tick when the weapon's `m_fPinPullTime` changes to a valid positive value; no server-time conversion is used
  - 0.5s window: fixed lookback of `tickRate/2` ticks as fallback
- Scans all `match.PlayerButtons` records within the window, accumulating via OR to get the final state
- `HasJump` is ultimately overridden by the engine property `m_bJumpThrow` (i.e. `IsJumpThrow`) for accuracy

**3. Throw Classification (`utility.go`)**
- **MouseTypeByStrength**: classifies mouse action via `m_flThrowStrength` property
  - `1.0` → `left_click`, `0.5` → `double_click` (both buttons simultaneously)
  - Other values use `calcThrowStrength()` as a fallback; zero may be unassigned, and releases on death may have no usable launch velocity:
    `strength = sqrt((initVelX - 1.25*throwerVelX)² + (initVelY - 1.25*throwerVelY)²) / cos(pitch)`
    Positive finite thresholds: >500 = `left_click`, >300 = `double_click`, otherwise = `right_click`. Nonpositive or nonfinite fallback values are `unknown`, not evidence of a right click.
- **ThrowerSpeedType**: classifies thrower movement state by `speed2D`
  - `standing`: speed2D == 0
  - `step`: 0 < speed2D < 80
  - `walk`: 80 ≤ speed2D < 180
  - `run`: speed2D ≥ 180

#### Event Flow
```
Weapon m_fPinPullTime changes
 → records the current demo tick for this weapon instance

WeaponFire event
 → newUtilityFromShot(): creates Utility and reads the recorded PinPulledTick
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
- `pin pulled tick`: Demo tick when the pin pull was observed (0 = not observed, including a grenade already primed when its entity first appears)
- `mouse type by strength`: Mouse action classification (`left_click`/`double_click`/`right_click`/`unknown`)
- `is jump throw`: Whether it's a jump throw (engine property `m_bJumpThrow`)
- `throw strength`: Raw engine property `m_flThrowStrength`; zero alone does not prove a mouse action

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
- Initial property values are not treated as new pulls. Repeated values do not move the recorded tick. Reset values clear it; destroyed weapons are removed at frame end so same-frame throw events can still use it. Entity serial numbers prevent reused indices from sharing pull records.

#### Related Files
- `pkg/api/utility.go` — Utility struct, all classification functions, button detection logic
- `pkg/api/utility_pin.go` — Observed demo-tick pin tracking and weapon lifecycle handling
- `pkg/api/utils.go` — `getPlayerVelocity()` position-delta velocity calculation
- `pkg/api/analyzer.go` — FrameDone handler (position/tick rotation), WeaponFire/GrenadeProjectileThrow event handlers
- `pkg/api/match.go` — Position/tick history maps in Match struct
- `pkg/api/export_csv.go` — CSV export

---

### Usage

Ready-to-use binaries are available on the [releases page](https://github.com/WangChuDi/cs-demo-analyzer-mod/releases).


## ⚠️ Known Issues

The pin-time clock mismatch found in the 2026-10-30 audit is fixed by recording observed demo ticks directly. Pin times remain unknown when no pull update was observed. Official v6.0.0-rc.1 seeds command baselines from full packets before decoding live deltas; the extra v4 patch is no longer needed for this sample. Command-number and ring validation remain enforced. The Aurora–MOUZ Cache sample exports 4,421 first-round button-state changes for all ten players, recovering mouse input for all five first-round throws. Missing raw footsteps, aim-punch properties, or bullet-penetration events must also be distinguished from real zero values.

The follow-up audit fixes player/opponent filtering in 1vX totals and makes AWP holds use kill-event state. Wallbang heuristics retain victim position/helmet state at the hurt event, independently of `-positions`; this flag only controls trajectory export. With one-point deficits excluded, the damage model reports 20 heuristic candidates totaling 332 actual HP damage in the Cache sample. Two one-point differences remain available as diagnostics and do not contribute to wallbang totals. These are model classifications, not video-verified labels.

CSV and JSON expose shot/damage availability and match sample counts (`dataQuality` in JSON). Interpret numeric zeros together with their flags. `wallbang data source` is `bullet_damage`, `player_bullet_hit`, `kill_event`, `heuristic`, or `unavailable`; observed BulletDamage, including zero, takes precedence. Kill data only matches a unique lethal hurt event with the same round, frame, attacker, victim and weapon, and never sets `has bullet damage data`. The sample adds 123 confirmed fatal penetration damage from 3 kills to the 332 heuristic damage (455 outward total).

SoS messages recover known CT/T footstep sounds, resolving pawn entity IDs at frame end and preferring game events when both exist. Land/gear sounds and unknown hashes are excluded. `data source` and `sound event name` identify the origin. The sample has 8,999 recovered steps across ten players; this does not guarantee completeness or prove what each opponent heard.

Newer CS2 versions network AimPunchServices base angles, velocity and server ticks. These values are exported separately as `aimPunchBase` in JSON and `aim punch ... base ...` CSV columns (1,899 samples here). Base state is not an instantaneous angle. The base GameTick_t fields are ZigZag-decoded for v6.0.0-rc.1, and `serverTick` uses explicit net/snapshot ticks with `hasServerTickData`; earlier fork exports accidentally used demo ticks and unsigned wire values. Compare decoded base ticks with an available server clock, not demo ticks.

When recorded, `FireBullets.Extra.aim_punch` now supplies instantaneous angles (`aimPunchDataSource=fire_bullets`); older entity angles use `entity`, absent data uses `unavailable`. Matching requires a unique round/frame/player/weapon and counts incomplete messages as ambiguous. `PlayerBulletHit.penetration_count`, including explicit zero, can recover nonlethal penetration with `hasPlayerBulletHitData` and source `player_bullet_hit`. A unique same-frame hurt must agree on participants, damage, and any supplied hitgroup/kill flag; BulletDamage remains authoritative. These messages are absent in the audited Cache demo, so its two precise signals remain unavailable. CSV adds availability/source columns; fixed-column CSDM remains unchanged.

Validated ServerInfo VBKV supplies active `c_game_type/c_game_mode`, provenance `server_info_data`; map defaults are ignored. Without evidence mode remains unknown. Use `-match-date '2026-10-29T00:00:00+08:00'` or `CSDA_MATCH_DATE` for an external date. Positive `.info` time takes precedence, then external metadata, then file mtime (`date source`: `match_info`, `external_metadata`, `file_mtime`). Fixed-column CSDM retains its layout without these new fields. Existing database rows need reanalysis/reimport.

### Unlucky Statistics
- **Wallbang Damage**: Outward wallbang reporting (`Damage.IsWallbang`, `wallbang damage dealt`, `wallbang damage taken`) uses parser-confirmed penetration signals when available, otherwise a heuristic fallback. The fallback requires a unique same-frame shot from the same attacker/weapon and round. It compares floored theoretical damage with the integer `player_hurt.dmg_health` report, not actual HP removed; a deficit of exactly one is ignored for wallbang classification and player totals, while its diagnostics remain available with reason `within_one_point_tolerance`. A deficit of at least two is required for a heuristic candidate. HP damage remains the value summed in player statistics. Low-armor depletion and same-frame armor/health sequences are handled separately. Observed CT/T body/head damage scales are snapshotted at the hurt event; absent scales assume the standard default of 1. Friendly fire, unsupported hitgroups, ambiguous shots, missing inputs and observations above the model remain `unavailable`. Thirty single-projectile guns are covered for verified protocols 14005–14192, with pre/post-14130 MP7/MP5-SD profiles; unverified versions and shotgun pellets are excluded. New CSV/JSON fields expose reported damage, expected integer damage, signed deficit, model source, reason and evaluation availability. Position-derived range, unrecorded server settings and quantization still limit certainty, so inferred candidates do not supply an exact penetration count. `true wallbang damage taken` remains parser-confirmed only (BulletDamage, PlayerBulletHit, or uniquely matched positive fatal penetration).
- **CS2 Fall Damage**: The previous demoinfocs v5 dependency could report typed `PlayerHurt.Weapon` as `C4` for true world/fall damage in some CS2 demos. The analyzer retains generic `player_hurt` environment-damage classification and excludes candidates that occur on the same frame as `BombExplode`.


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

Tests compare the analyzer output against the JSON snapshots stored in `tests/snapshots`.

Every demo is an entry in the `demoTestCases` table in `tests/demos_test.go` and runs as a parallel subtest of `TestDemos`, named `TestDemos/<game>/<name>` (e.g. `TestDemos/cs2/Esportal_6008132_2023_Mirage`). To add a demo, add a row to the table rather than a new file.

First, download the demos used by the tests (one-time setup):

```bash
./download-demos.sh
```

#### Running tests

```bash
# Run all tests
make test

# Only CS:GO or only CS2 tests
make test-csgo
make test-cs2
```

Pass extra flags to `go test` through the `ARGS` variable. The `-run` value is a regex matched against the slash-separated subtest path:

```bash
# Run a single demo
make test ARGS="-run TestDemos/cs2/Esportal_6008132_2023_Mirage"

# Run every demo whose name contains Esportal, in verbose mode
make test ARGS="-run TestDemos/.*/Esportal -v"
```

#### Updating snapshots

When the analyzer output legitimately changes, regenerate the snapshots by passing `-update`:

```bash
# Regenerate all snapshots
make test ARGS="-update"

# Regenerate the snapshot for a single demo
make test ARGS="-update -run TestDemos/cs2/Esportal_6008132_2023_Mirage"
```

Review the resulting snapshot diff before committing.

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
