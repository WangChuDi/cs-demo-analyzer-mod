package api

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"io"

	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
)

func (match *Match) applyServerInfo(info *msg.CSVCMsg_ServerInfo) {
	config := info.GetGameSessionConfig()
	if config == nil {
		return
	}
	if mode := config.GetGamemode(); mode != "" {
		match.gameModeStr = constants.GameModeStr(mode)
		match.gameModeSource = "server_info"
	}
	gameType, gameMode, ok := gameModeFromVBKV(config.GetData())
	if !ok {
		return
	}
	match.GameType, match.GameMode = constants.GameType(gameType), constants.GameMode(gameMode)
	match.HasGameTypeData, match.HasGameModeData = true, true
	// Use the active c_game_* values, never the map's default_game_* settings.
	match.gameModeStr = constants.GameModeMapping[match.GameType][match.GameMode]
	match.gameModeSource = "server_info_data"
}

// The session data uses Valve binary KeyValues: VBKV, CRC32 of the payload,
// then typed, NUL-terminated keys. Its alternate-format object terminator is 11.
// Validate the entire document before accepting either active game setting.
func gameModeFromVBKV(data []byte) (gameType, gameMode int, ok bool) {
	if len(data) < 9 || len(data) > 1<<20 || string(data[:4]) != "VBKV" ||
		crc32.ChecksumIEEE(data[8:]) != binary.LittleEndian.Uint32(data[4:8]) {
		return 0, 0, false
	}
	r := bytes.NewBuffer(data[8:])
	values := map[string]int{}
	var object func(int, bool) bool
	object = func(depth int, active bool) bool {
		if depth > 64 {
			return false
		}
		for {
			kind, err := r.ReadByte()
			if err != nil {
				return false
			}
			if kind == 11 {
				return true
			}
			key, err := r.ReadString(0)
			if err != nil {
				return false
			}
			key = key[:len(key)-1]
			if kind == 0 {
				if !object(depth+1, active && key == "") {
					return false
				}
				continue
			}
			if active && (key == "c_game_type" || key == "c_game_mode") {
				if _, duplicate := values[key]; duplicate || kind != 2 || r.Len() < 4 {
					return false
				}
				values[key] = int(int32(binary.LittleEndian.Uint32(r.Next(4))))
				continue
			}
			switch kind {
			case 1:
				if _, err := r.ReadString(0); err != nil {
					return false
				}
			case 2, 3, 4, 6:
				if r.Len() < 4 {
					return false
				}
				r.Next(4)
			case 5:
				// Wide strings have a 16-bit character count, including NUL.
				var length uint16
				if binary.Read(r, binary.LittleEndian, &length) != nil || length == 0 || r.Len() < int(length)*2 {
					return false
				}
				r.Next(int(length) * 2)
			case 7, 10:
				if _, err := io.CopyN(io.Discard, r, 8); err != nil {
					return false
				}
			default:
				return false
			}
		}
	}
	if !object(0, true) || r.Len() != 0 || len(values) != 2 {
		return 0, 0, false
	}
	gameType, gameMode = values["c_game_type"], values["c_game_mode"]
	return gameType, gameMode, gameType >= 0 && gameMode >= 0
}
