package gofakeit

import "math/rand"

// MinecraftOre will generate a random Minecraft ore
func MinecraftOre() string { return minecraftOre(globalFaker.Rand) }

// MinecraftOre will generate a random Minecraft ore
func (f *Faker) MinecraftOre() string { return minecraftOre(f.Rand) }

func minecraftOre(r *rand.Rand) string { return getRandValue(r, []string{"minecraft", "ore"}) }

// MinecraftWood will generate a random Minecraft wood
func MinecraftWood() string { return minecraftWood(globalFaker.Rand) }

// MinecraftWood will generate a random Minecraft wood
func (f *Faker) MinecraftWood() string { return minecraftWood(f.Rand) }

func minecraftWood(r *rand.Rand) string { return getRandValue(r, []string{"minecraft", "wood"}) }

// MinecraftArmorTier will generate a random Minecraft armor tier
func MinecraftArmorTier() string { return minecraftArmorTier(globalFaker.Rand) }

// MinecraftArmorTier will generate a random Minecraft armor tier
func (f *Faker) MinecraftArmorTier() string { return minecraftArmorTier(f.Rand) }

func minecraftArmorTier(r *rand.Rand) string {
	return getRandValue(r, []string{"minecraft", "armortier"})
}

// MinecraftArmorPart will generate a random Minecraft armor part
func MinecraftArmorPart() string { return minecraftArmorPart(globalFaker.Rand) }

// MinecraftArmorPart will generate a random Minecraft armor part
func (f *Faker) MinecraftArmorPart() string { return minecraftArmorPart(f.Rand) }

func minecraftArmorPart(r *rand.Rand) string {
	return getRandValue(r, []string{"minecraft", "armorpart"})
}

// MinecraftWeapon will generate a random Minecraft weapon
func MinecraftWeapon() string { return minecraftWeapon(globalFaker.Rand) }

// MinecraftWeapon will generate a random Minecraft weapon
func (f *Faker) MinecraftWeapon() string { return minecraftWeapon(f.Rand) }

func minecraftWeapon(r *rand.Rand) string { return getRandValue(r, []string{"minecraft", "weapon"}) }

// MinecraftTool will generate a random Minecraft tool
func MinecraftTool() string { return minecraftTool(globalFaker.Rand) }

// MinecraftTool will generate a random Minecraft tool
func (f *Faker) MinecraftTool() string { return minecraftTool(f.Rand) }

func minecraftTool(r *rand.Rand) string { return getRandValue(r, []string{"minecraft", "tool"}) }

// MinecraftDye will generate a random Minecraft dye
func MinecraftDye() string { return minecraftDye(globalFaker.Rand) }

// MinecraftDye will generate a random Minecraft dye
func (f *Faker) MinecraftDye() string { return minecraftDye(f.Rand) }

func minecraftDye(r *rand.Rand) string { return getRandValue(r, []string{"minecraft", "dye"}) }

// MinecraftFood will generate a random Minecraft food
func MinecraftFood() string { return minecraftFood(globalFaker.Rand) }

// MinecraftFood will generate a random Minecraft food
func (f *Faker) MinecraftFood() string { return minecraftFood(f.Rand) }

func minecraftFood(r *rand.Rand) string { return getRandValue(r, []string{"minecraft", "food"}) }

// MinecraftAnimal will generate a random Minecraft animal
func MinecraftAnimal() string { return minecraftAnimal(globalFaker.Rand) }

// MinecraftAnimal will generate a random Minecraft animal
func (f *Faker) MinecraftAnimal() string { return minecraftAnimal(f.Rand) }

func minecraftAnimal(r *rand.Rand) string {
	return getRandValue(r, []string{"minecraft", "animal"})
}

// MinecraftVillagerJob will generate a random Minecraft villager job
func MinecraftVillagerJob() string { return minecraftVillagerJob(globalFaker.Rand) }

// MinecraftVillagerJob will generate a random Minecraft villager job
func (f *Faker) MinecraftVillagerJob() string { return minecraftVillagerJob(f.Rand) }

func minecraftVillagerJob(r *rand.Rand) string {
	return getRandValue(r, []string{"minecraft", "villagerjob"})
}

// MinecraftVillagerStation will generate a random Minecraft villager station
func MinecraftVillagerStation() string { return minecraftVillagerStation(globalFaker.Rand) }

// MinecraftVillagerStation will generate a random Minecraft villager station
func (f *Faker) MinecraftVillagerStation() string { return minecraftVillagerStation(f.Rand) }

func minecraftVillagerStation(r *rand.Rand) string {
	return getRandValue(r, []string{"minecraft", "villagerstation"})
}

// MinecraftVillagerLevel will generate a random Minecraft villager level
func MinecraftVillagerLevel() string { return minecraftVillagerLevel(globalFaker.Rand) }

// MinecraftVillagerLevel will generate a random Minecraft villager level
func (f *Faker) MinecraftVillagerLevel() string { return minecraftVillagerLevel(f.Rand) }

func minecraftVillagerLevel(r *rand.Rand) string {
	return getRandValue(r, []string{"minecraft", "villagerlevel"})
}

// MinecraftMobPassive will generate a random Minecraft mob passive
func MinecraftMobPassive() string { return minecraftMobPassive(globalFaker.Rand) }

// MinecraftMobPassive will generate a random Minecraft mob passive
func (f *Faker) MinecraftMobPassive() string { return minecraftMobPassive(f.Rand) }

func minecraftMobPassive(r *rand.Rand) string {
	return getRandValue(r, []string{"minecraft", "mobpassive"})
}

// MinecraftMobNeutral will generate a random Minecraft mob neutral
func MinecraftMobNeutral() string { return minecraftMobNeutral(globalFaker.Rand) }

// MinecraftMobNeutral will generate a random Minecraft mob neutral
func (f *Faker) MinecraftMobNeutral() string { return minecraftMobNeutral(f.Rand) }

func minecraftMobNeutral(r *rand.Rand) string {
	return getRandValue(r, []string{"minecraft", "mobneutral"})
}

// MinecraftMobHostile will generate a random Minecraft mob hostile
func MinecraftMobHostile() string { return minecraftMobHostile(globalFaker.Rand) }

// MinecraftMobHostile will generate a random Minecraft mob hostile
func (f *Faker) MinecraftMobHostile() string { return minecraftMobHostile(f.Rand) }

func minecraftMobHostile(r *rand.Rand) string {
	return getRandValue(r, []string{"minecraft", "mobhostile"})
}

// MinecraftMobBoss will generate a random Minecraft mob boss
func MinecraftMobBoss() string { return minecraftMobBoss(globalFaker.Rand) }

// MinecraftMobBoss will generate a random Minecraft mob boss
func (f *Faker) MinecraftMobBoss() string { return minecraftMobBoss(f.Rand) }

func minecraftMobBoss(r *rand.Rand) string {
	return getRandValue(r, []string{"minecraft", "mobboss"})
}

// MinecraftBiome will generate a random Minecraft biome
func MinecraftBiome() string { return minecraftBiome(globalFaker.Rand) }

// MinecraftBiome will generate a random Minecraft biome
func (f *Faker) MinecraftBiome() string { return minecraftBiome(f.Rand) }

func minecraftBiome(r *rand.Rand) string { return getRandValue(r, []string{"minecraft", "biome"}) }

// MinecraftWeather will generate a random Minecraft weather
func MinecraftWeather() string { return minecraftWeather(globalFaker.Rand) }

// MinecraftWeather will generate a random Minecraft weather
func (f *Faker) MinecraftWeather() string { return minecraftWeather(f.Rand) }

func minecraftWeather(r *rand.Rand) string { return getRandValue(r, []string{"minecraft", "weather"}) }

func addMinecraftLookup() {
	AddFuncLookup("minecraftore", Info{
		Display:     "Minecraft ore",
		Category:    "minecraft",
		Description: "Naturally occurring minerals found in the game Minecraft, used for crafting purposes",
		Example:     "coal",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return minecraftOre(r), nil
		},
	})

	AddFuncLookup("minecraftwood", Info{
		Display:     "Minecraft wood",
		Category:    "minecraft",
		Description: "Natural resource in Minecraft, used for crafting various items and building structures",
		Example:     "oak",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return minecraftWood(r), nil
		},
	})

	AddFuncLookup("minecraftarmortier", Info{
		Display:     "Minecraft armor tier",
		Category:    "minecraft",
		Description: "Classification system for armor sets in Minecraft, indicating their effectiveness and protection level",
		Example:     "iron",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return minecraftArmorTier(r), nil
		},
	})

	AddFuncLookup("minecraftarmorpart", Info{
		Display:     "Minecraft armor part",
		Category:    "minecraft",
		Description: "Component of an armor set in Minecraft, such as a helmet, chestplate, leggings, or boots",
		Example:     "helmet",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return minecraftArmorPart(r), nil
		},
	})

	AddFuncLookup("minecraftweapon", Info{
		Display:     "Minecraft weapon",
		Category:    "minecraft",
		Description: "Tools and items used in Minecraft for combat and defeating hostile mobs",
		Example:     "bow",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return minecraftWeapon(r), nil
		},
	})

	AddFuncLookup("minecrafttool", Info{
		Display:     "Minecraft tool",
		Category:    "minecraft",
		Description: "Items in Minecraft designed for specific tasks, including mining, digging, and building",
		Example:     "shovel",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return minecraftTool(r), nil
		},
	})

	AddFuncLookup("minecraftdye", Info{
		Display:     "Minecraft dye",
		Category:    "minecraft",
		Description: "Items used to change the color of various in-game objects",
		Example:     "white",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return minecraftDye(r), nil
		},
	})

	AddFuncLookup("minecraftfood", Info{
		Display:     "Minecraft food",
		Category:    "minecraft",
		Description: "Consumable items in Minecraft that provide nourishment to the player character",
		Example:     "apple",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return minecraftFood(r), nil
		},
	})

	AddFuncLookup("minecraftanimal", Info{
		Display:     "Minecraft animal",
		Category:    "minecraft",
		Description: "Non-hostile creatures in Minecraft, often used for resources and farming",
		Example:     "chicken",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return minecraftAnimal(r), nil
		},
	})

	AddFuncLookup("minecraftvillagerjob", Info{
		Display:     "Minecraft villager job",
		Category:    "minecraft",
		Description: "The profession or occupation assigned to a villager character in the game",
		Example:     "farmer",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return minecraftVillagerJob(r), nil
		},
	})

	AddFuncLookup("minecraftvillagerstation", Info{
		Display:     "Minecraft villager station",
		Category:    "minecraft",
		Description: "Designated area or structure in Minecraft where villagers perform their job-related tasks and trading",
		Example:     "furnace",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return minecraftVillagerStation(r), nil
		},
	})

	AddFuncLookup("minecraftvillagerlevel", Info{
		Display:     "Minecraft villager level",
		Category:    "minecraft",
		Description: "Measure of a villager's experience and proficiency in their assigned job or profession",
		Example:     "master",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return minecraftVillagerLevel(r), nil
		},
	})

	AddFuncLookup("minecraftmobpassive", Info{
		Display:     "Minecraft mob passive",
		Category:    "minecraft",
		Description: "Non-aggressive creatures in the game that do not attack players",
		Example:     "cow",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return minecraftMobPassive(r), nil
		},
	})

	AddFuncLookup("minecraftmobneutral", Info{
		Display:     "Minecraft mob neutral",
		Category:    "minecraft",
		Description: "Creature in the game that only becomes hostile if provoked, typically defending itself when attacked",
		Example:     "bee",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return minecraftMobNeutral(r), nil
		},
	})

	AddFuncLookup("minecraftmobhostile", Info{
		Display:     "Minecraft mob hostile",
		Category:    "minecraft",
		Description: "Aggressive creatures in the game that actively attack players when encountered",
		Example:     "spider",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return minecraftMobHostile(r), nil
		},
	})

	AddFuncLookup("minecraftmobboss", Info{
		Display:     "Minecraft mob boss",
		Category:    "minecraft",
		Description: "Powerful hostile creature in the game, often found in challenging dungeons or structures",
		Example:     "ender dragon",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return minecraftMobBoss(r), nil
		},
	})

	AddFuncLookup("minecraftbiome", Info{
		Display:     "Minecraft biome",
		Category:    "minecraft",
		Description: "Distinctive environmental regions in the game, characterized by unique terrain, vegetation, and weather",
		Example:     "forest",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return minecraftBiome(r), nil
		},
	})

	AddFuncLookup("minecraftweather", Info{
		Display:     "Minecraft weather",
		Category:    "minecraft",
		Description: "Atmospheric conditions in the game that include rain, thunderstorms, and clear skies, affecting gameplay and ambiance",
		Example:     "rain",
		Output:      "string",
		Generate: func(r *rand.Rand, m *MapParams, info *Info) (any, error) {
			return minecraftWeather(r), nil
		},
	})
}
