package entity

import (
	"sync"
	"testing"
	_ "unsafe"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/df-mc/dragonfly/server/world/generator/vanilla"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/df-mc/dragonfly/server/world/sound"
	"github.com/go-gl/mathgl/mgl64"
)

type eyeOfEnderLocatorGenerator struct {
	info             vanilla.PlannedStructureInfo
	ok               bool
	maxChunkDistance *int
}

func (g eyeOfEnderLocatorGenerator) GenerateChunk(world.ChunkPos, *chunk.Chunk) {}

func (g eyeOfEnderLocatorGenerator) LocateNearestPlannedStructureStart(setName string, origin cube.Pos, maxChunkDistance int) (vanilla.PlannedStructureInfo, bool) {
	if setName != "strongholds" {
		return vanilla.PlannedStructureInfo{}, false
	}
	if g.maxChunkDistance != nil {
		*g.maxChunkDistance = maxChunkDistance
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

type eyeOfEnderTestViewer struct {
	world.NopViewer
	sounds    []world.Sound
	particles []world.Particle
}

func (v *eyeOfEnderTestViewer) ViewParticle(_ mgl64.Vec3, p world.Particle) {
	v.particles = append(v.particles, p)
}

func (v *eyeOfEnderTestViewer) ViewSound(_ mgl64.Vec3, s world.Sound) {
	v.sounds = append(v.sounds, s)
}

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
		gotPos     mgl64.Vec3
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
		gotPos = m.Position()
		gotTarget = beh.target
		wantTarget = eyeOfEnderSignalTarget(owner.Position(), mgl64.Vec3{
			float64(info.Origin.X()),
			float64(info.Origin.Y()),
			float64(info.Origin.Z()),
		})
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
	if gotPos.ApproxEqualThreshold(mgl64.Vec3{0.5, 64.5, 0.5}, 1e-9) {
		t.Fatalf("expected eye of ender to move on the first guided tick, got %v", gotPos)
	}
	if !gotTarget.ApproxEqualThreshold(wantTarget, 1e-6) {
		t.Fatalf("expected eye of ender target %v, got %v", wantTarget, gotTarget)
	}
}

func TestEyeOfEnderUsesWideStrongholdSearch(t *testing.T) {
	finaliseEyeBlocksOnce.Do(worldFinaliseBlockRegistry)

	info := vanilla.PlannedStructureInfo{
		StructureSet: "strongholds",
		Structure:    "stronghold",
		Origin:       cube.Pos{128, 32, 96},
		Size:         [3]int{48, 16, 32},
	}
	var maxChunkDistance int
	w := world.Config{
		Dim:       world.Overworld,
		Provider:  world.NopProvider{},
		Generator: eyeOfEnderLocatorGenerator{info: info, ok: true, maxChunkDistance: &maxChunkDistance},
		Entities:  DefaultRegistry,
	}.New()
	defer func() { _ = w.Close() }()

	var errText string
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
		if beh.Tick(eye, tx) == nil {
			errText = "expected eye of ender target tick movement"
		}
	})
	if errText != "" {
		t.Fatal(errText)
	}
	if maxChunkDistance != eyeOfEnderStrongholdSearchChunks {
		t.Fatalf("expected eye of ender to request %d chunks of stronghold search, got %d", eyeOfEnderStrongholdSearchChunks, maxChunkDistance)
	}
}

func TestEyeOfEnderSignalTargetClampsFarTarget(t *testing.T) {
	pos := mgl64.Vec3{0.5, 64.5, 0.5}
	target := mgl64.Vec3{24.5, 32, 0.5}

	got := eyeOfEnderSignalTarget(pos, target)
	want := mgl64.Vec3{12.5, 72.5, 0.5}

	if !got.ApproxEqualThreshold(want, 1e-12) {
		t.Fatalf("expected clamped eye target %v, got %v", want, got)
	}
}

func TestEyeOfEnderUpdateDeltaMovementUsesJavaLerpStep(t *testing.T) {
	oldMovement := mgl64.Vec3{}
	pos := mgl64.Vec3{0.5, 64.5, 0.5}
	target := mgl64.Vec3{12.5, 72.5, 0.5}

	got := eyeOfEnderUpdateDeltaMovement(oldMovement, pos, target)
	want := mgl64.Vec3{0.03, 0.015, 0}

	if !got.ApproxEqualThreshold(want, 1e-12) {
		t.Fatalf("expected guided eye velocity %v, got %v", want, got)
	}
}

func TestGuidedEyeOfEnderIgnoresBlockCollision(t *testing.T) {
	finaliseEyeBlocksOnce.Do(worldFinaliseBlockRegistry)

	info := vanilla.PlannedStructureInfo{
		StructureSet: "strongholds",
		Structure:    "stronghold",
		Origin:       cube.Pos{0, 32, 256},
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
		firstPos  mgl64.Vec3
		secondPos mgl64.Vec3
		tickNil   bool
		errText   string
	)
	<-w.Exec(func(tx *world.Tx) {
		tx.SetBlock(cube.Pos{0, 64, 1}, block.Stone{}, nil)

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

		first := beh.Tick(eye, tx)
		if first == nil {
			tickNil = true
			return
		}
		second := beh.Tick(eye, tx)
		if second == nil {
			tickNil = true
			return
		}
		firstPos = first.Position()
		secondPos = second.Position()
	})
	if errText != "" {
		t.Fatal(errText)
	}
	if tickNil {
		t.Fatal("expected guided eye of ender to keep moving through an obstructing block")
	}
	if secondPos[2] <= firstPos[2] {
		t.Fatalf("expected guided eye of ender to keep advancing toward the target, got first=%v second=%v", firstPos, secondPos)
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

func TestGuidedEyeOfEnderExpiresAfterFiniteTicks(t *testing.T) {
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
		expiredAt int
		errText   string
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

		for tick := 1; tick <= 100; tick++ {
			m := beh.Tick(eye, tx)
			if tick == 1 {
				if !beh.hasTarget {
					errText = "expected guided eye to resolve a stronghold target"
					return
				}
				beh.survive = false
			}
			if m == nil {
				expiredAt = tick
				return
			}
		}
		errText = "expected guided eye to expire within 100 ticks"
	})
	if errText != "" {
		t.Fatal(errText)
	}
	if expiredAt != 81 {
		t.Fatalf("expected guided eye to expire on tick 81, got %d", expiredAt)
	}
}

func TestGuidedEyeOfEnderShatterPlaysFeedback(t *testing.T) {
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

	viewer := &eyeOfEnderTestViewer{}
	loader := world.NewLoader(0, w, viewer)

	var errText string
	<-w.Exec(func(tx *world.Tx) {
		defer loader.Close(tx)

		_ = tx.Chunk(world.ChunkPos{0, 0})
		loader.Move(tx, mgl64.Vec3{0.5, 64.5, 0.5})
		loader.Load(tx, 1)

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

		for tick := 1; tick <= 100; tick++ {
			m := beh.Tick(eye, tx)
			if tick == 1 {
				beh.survive = false
			}
			if m == nil {
				return
			}
		}
		errText = "expected guided eye to expire within 100 ticks"
	})
	if errText != "" {
		t.Fatal(errText)
	}
	if len(viewer.sounds) != 1 {
		t.Fatalf("expected 1 expiry sound, got %d", len(viewer.sounds))
	}
	if _, ok := viewer.sounds[0].(sound.ItemBreak); !ok {
		t.Fatalf("expected expiry sound %T, got %T", sound.ItemBreak{}, viewer.sounds[0])
	}
	if len(viewer.particles) != 1 {
		t.Fatalf("expected 1 shatter particle, got %d", len(viewer.particles))
	}
	if _, ok := viewer.particles[0].(particle.EndermanTeleport); !ok {
		t.Fatalf("expected shatter particle %T, got %T", particle.EndermanTeleport{}, viewer.particles[0])
	}
}
