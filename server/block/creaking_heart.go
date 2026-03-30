package block

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
)

// CreakingHeartState is the current active state of a creaking heart.
type CreakingHeartState struct {
	creakingHeartState
}

// UprootedCreakingHeart returns the uprooted state.
func UprootedCreakingHeart() CreakingHeartState {
	return CreakingHeartState{0}
}

// DormantCreakingHeart returns the dormant state.
func DormantCreakingHeart() CreakingHeartState {
	return CreakingHeartState{1}
}

// AwakeCreakingHeart returns the awake state.
func AwakeCreakingHeart() CreakingHeartState {
	return CreakingHeartState{2}
}

// CreakingHeartStates returns all creaking heart states.
func CreakingHeartStates() []CreakingHeartState {
	return []CreakingHeartState{UprootedCreakingHeart(), DormantCreakingHeart(), AwakeCreakingHeart()}
}

type creakingHeartState uint8

// Uint8 ...
func (s creakingHeartState) Uint8() uint8 {
	return uint8(s)
}

// String ...
func (s creakingHeartState) String() string {
	switch s {
	case 0:
		return "uprooted"
	case 1:
		return "dormant"
	case 2:
		return "awake"
	}
	panic("unknown creaking heart state")
}

// CreakingHeart is the pale oak heart block used by the creaking tree decorator.
type CreakingHeart struct {
	solid

	Axis    cube.Axis
	Natural bool
	State   CreakingHeartState
}

// BreakInfo ...
func (c CreakingHeart) BreakInfo() BreakInfo {
	return newBreakInfo(2.0, alwaysHarvestable, axeEffective, oneOf(c))
}

// EncodeItem ...
func (CreakingHeart) EncodeItem() (name string, meta int16) {
	return "minecraft:creaking_heart", 0
}

// EncodeBlock ...
func (c CreakingHeart) EncodeBlock() (string, map[string]any) {
	return "minecraft:creaking_heart", map[string]any{
		"creaking_heart_state": c.State.String(),
		"natural":              c.Natural,
		"pillar_axis":          c.Axis.String(),
	}
}

// allCreakingHeart returns all creaking heart states.
func allCreakingHeart() (b []world.Block) {
	for _, axis := range cube.Axes() {
		for _, state := range CreakingHeartStates() {
			b = append(b,
				CreakingHeart{Axis: axis, State: state},
				CreakingHeart{Axis: axis, Natural: true, State: state},
			)
		}
	}
	return
}
