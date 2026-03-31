package block

// Spawner is a solid mob spawner block used in dungeon generation.
type Spawner struct {
	solid
}

// BreakInfo ...
func (s Spawner) BreakInfo() BreakInfo {
	return newBreakInfo(5, pickaxeHarvestable, pickaxeEffective, oneOf(s))
}

// EncodeItem ...
func (Spawner) EncodeItem() (name string, meta int16) {
	return "minecraft:mob_spawner", 0
}

// EncodeBlock ...
func (Spawner) EncodeBlock() (string, map[string]any) {
	return "minecraft:mob_spawner", nil
}
