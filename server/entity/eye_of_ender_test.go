package entity

import (
	"sync"
	"testing"
	_ "unsafe"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/generator/vanilla"
	"github.com/go-gl/mathgl/mgl64"
)

type eyeOfEnderLocatorGenerator struct {
	info vanilla.PlannedStructureInfo
	ok   bool
}

func (g eyeOfEnderLocatorGenerator) GenerateChunk(world.ChunkPos, *chunk.Chunk) {}

func (g eyeOfEnderLocatorGenerator) LocateNearestPlannedStructureStart(setName string, origin cube.Pos, maxChunkDistance int) (vanilla.PlannedStructureInfo, bool) {
	if setName != "strongholds" {
		return vanilla.PlannedStructureInfo{}, false
	}
	return g.info, g.ok
}

type eyeOfEnderSeedOnlyGenerator struct {
	seed int64
}

func (g eyeOfEnderSeedOnlyGenerator) GenerateChunk(world.ChunkPos, *chunk.Chunk) {}

func (g eyeOfEnderSeedOnlyGenerator) Seed() int64 {
	return g.seed
}

var finaliseEyeBlocksOnce sync.Once

//go:linkname worldFinaliseBlockRegistry github.com/df-mc/dragonfly/server/world.finaliseBlockRegistry
func worldFinaliseBlockRegistry()

func TestEyeOfEnderResolvesStrongholdTarget(t *testing.T) {
	finaliseEyeBlocksOnce.Do(worldFinaliseBlockRegistry)

	info := vanilla.PlannedStructureInfo{
		StructureSet: "strongholds",
		Structure:    "stronghold",
		Origin:       cube.Pos{128, 32, 96},
		Size:         [3]int{48, 16, 32},
	}
	w := world.Config{
		Dim:       world.Overworld,
		Provider:  world.NopProvider{},
		Generator: eyeOfEnderLocatorGenerator{info: info, ok: true},
		Entities:  DefaultRegistry,
	}.New()
	defer func() { _ = w.Close() }()

	var (
		resolved   bool
		gotTarget  mgl64.Vec3
		wantTarget mgl64.Vec3
		tickNil    bool
		errText    string
	)
	<-w.Exec(func(tx *world.Tx) {
		owner := tx.AddEntity(NewText("eye-owner", mgl64.Vec3{0.5, 64.5, 0.5}))
		eye := tx.AddEntity(NewEyeOfEnder(world.EntitySpawnOpts{
			Position: owner.Position(),
			Velocity: cube.Rotation{0, 0}.Vec3().Mul(1.25),
		}, owner)).(*Ent)

		beh, ok := eye.Behaviour().(*EyeOfEnderBehaviour)
		if !ok {
			errText = "expected eye of ender behaviour"
			return
		}
		m := beh.Tick(eye, tx)
		if m == nil {
			tickNil = true
			return
		}
		resolved = beh.hasTarget
		gotTarget = beh.target
		wantTarget = mgl64.Vec3{
			float64(info.Origin.X()) + float64(info.Size[0])/2,
			owner.Position()[1],
			float64(info.Origin.Z()) + float64(info.Size[2])/2,
		}
	})
	if errText != "" {
		t.Fatal(errText)
	}
	if tickNil {
		t.Fatal("expected eye of ender target tick movement")
	}
	if !resolved {
		t.Fatal("expected eye of ender to resolve a stronghold target")
	}
	if !gotTarget.ApproxEqualThreshold(wantTarget, 1e-6) {
		t.Fatalf("expected eye of ender target %v, got %v", wantTarget, gotTarget)
	}
}

func TestEyeOfEnderWithoutLocatorUsesProjectileFallback(t *testing.T) {
	finaliseEyeBlocksOnce.Do(worldFinaliseBlockRegistry)

	w := world.Config{
		Dim:       world.Overworld,
		Provider:  world.NopProvider{},
		Generator: eyeOfEnderSeedOnlyGenerator{seed: 0},
		Entities:  DefaultRegistry,
	}.New()
	defer func() { _ = w.Close() }()

	var (
		resolved      bool
		fallbackPos   mgl64.Vec3
		fallbackVel   mgl64.Vec3
		projectilePos mgl64.Vec3
		projectileVel mgl64.Vec3
		errText       string
	)
	<-w.Exec(func(tx *world.Tx) {
		owner := tx.AddEntity(NewText("eye-owner", mgl64.Vec3{0.5, 64, 0.5}))
		opts := world.EntitySpawnOpts{
			Position: owner.Position(),
			Velocity: cube.Rotation{0, 0}.Vec3().Mul(1.25),
		}
		eye := tx.AddEntity(NewEyeOfEnder(opts, owner)).(*Ent)
		projectileConf := eyeOfEnderProjectileConf
		projectileConf.Owner = owner.H()
		projectileEye := tx.AddEntity(opts.New(EyeOfEnderType, projectileConf)).(*Ent)

		beh, ok := eye.Behaviour().(*EyeOfEnderBehaviour)
		if !ok {
			errText = "expected eye of ender behaviour"
			return
		}
		projectileBeh, ok := projectileEye.Behaviour().(*ProjectileBehaviour)
		if !ok {
			errText = "expected projectile fallback behaviour"
			return
		}

		fallbackMovement := beh.Tick(eye, tx)
		if fallbackMovement == nil {
			errText = "expected eye of ender fallback tick movement"
			return
		}
		resolved = beh.hasTarget
		fallbackPos = fallbackMovement.Position()
		fallbackVel = fallbackMovement.Velocity()

		projectileMovement := projectileBeh.Tick(projectileEye, tx)
		if projectileMovement == nil {
			errText = "expected projectile fallback tick movement"
			return
		}
		projectilePos = projectileMovement.Position()
		projectileVel = projectileMovement.Velocity()
	})
	if errText != "" {
		t.Fatal(errText)
	}
	if resolved {
		t.Fatal("expected eye of ender fallback case without a stronghold target")
	}
	if !fallbackPos.ApproxEqualThreshold(projectilePos, 1e-9) {
		t.Fatalf("expected fallback eye position %v, got %v", projectilePos, fallbackPos)
	}
	if !fallbackVel.ApproxEqualThreshold(projectileVel, 1e-9) {
		t.Fatalf("expected fallback eye velocity %v, got %v", projectileVel, fallbackVel)
	}
}
