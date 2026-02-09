# CS Demo Analyzer (Modified Version)

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

> **Note**: In CS2 demos, player velocity fields (`player velocity x/y/z`) may appear as `0` for footstep events. This is due to the current behavior of the demo parsing library where velocity updates (via packet entities) might occur after the footstep event is dispatched in the same tick.

### 🎮 Player Buttons
Tracks player button presses (Attack, Jump, Duck, etc.) for every tick/update.

**Introduced Data Columns:**

- **Player Buttons Table (`_player_buttons.csv`)**:
  - `buttons`: Bitmask of pressed buttons.
  - `button_names`: Comma-separated list of pressed buttons (e.g. "Attack,Jump").

---

### Usage

Ready-to-use binaries are available on the [releases page](https://github.com/WangChuDi/cs-demo-analyzer-mod/releases).


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
