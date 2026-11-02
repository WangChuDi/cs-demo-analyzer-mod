package smokevoxel

import (
	"fmt"
	"math"

	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
)

type EntityData struct {
	Journal                   []byte
	Origin                    [3]float32
	EffectTick, NetworkUpdate int
}

// ReadEntity must be called after the entity's property updates finish. It owns
// the byte prefix; the parser may mutate the original array in the next packet.
func ReadEntity(entity *st.Entity) (EntityData, error) {
	var result EntityData
	readInt := func(name string) (int, error) {
		value, ok := entity.PropertyValue(name)
		if !ok {
			return 0, fmt.Errorf("missing %s", name)
		}
		n, ok := value.Any.(int32)
		if !ok {
			return 0, fmt.Errorf("invalid %s", name)
		}
		return int(n), nil
	}
	size, err := readInt("m_nVoxelFrameDataSize")
	if err != nil {
		return result, err
	}
	if size < 0 {
		return result, fmt.Errorf("negative voxel valid size")
	}
	if size == 0 {
		return result, nil
	}
	value, ok := entity.PropertyValue("m_VoxelFrameData")
	if !ok {
		return result, fmt.Errorf("missing voxel array")
	}
	values, ok := value.Any.([]any)
	if !ok || size > len(values) {
		return result, fmt.Errorf("invalid voxel valid size or array")
	}
	result.Journal = make([]byte, size)
	for i := range result.Journal {
		v, ok := values[i].(uint64)
		if !ok || v > 255 {
			return EntityData{}, fmt.Errorf("voxel element %d is not a byte", i)
		}
		result.Journal[i] = byte(v)
	}
	value, ok = entity.PropertyValue("m_vSmokeDetonationPos")
	if !ok {
		return result, fmt.Errorf("missing smoke origin")
	}
	switch v := value.Any.(type) {
	case [3]float32:
		result.Origin = v
	case []float32:
		if len(v) != 3 {
			return result, fmt.Errorf("invalid smoke origin length")
		}
		copy(result.Origin[:], v)
	default:
		return result, fmt.Errorf("invalid smoke origin type")
	}
	for _, v := range result.Origin {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return result, fmt.Errorf("non-finite smoke origin")
		}
	}
	result.EffectTick, err = readInt("m_nSmokeEffectTickBegin")
	if err != nil {
		return result, err
	}
	result.NetworkUpdate, err = readInt("m_nVoxelUpdate")
	return result, err
}
