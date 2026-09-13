package dungeon

import (
	vallenctx "gtps/VallenSource/VallenContext"
	variant "gtps/VallenSource/VallenVariant"
)

// ─────────────────────────────────────────────────────────────────────────────
// Dungeon XML & Asset Constants (Domain: VallenDungeon)
// ─────────────────────────────────────────────────────────────────────────────

const (
	NPCIndexes = "Axebeak|AxebeaksBoss|BladeofRevolt|Chest|CrystalTurret|DiamondDestructoBoss|DragonsBreath|DreadGazerBoss|FlippingSpikesHorizontal|FlippingSpikesHorizontalDown|FlippingSpikesVertical|FlippingSpikesVerticalRight|HealingAura|HiddenObject|IceGolemBoss|Jelatinous|JelatinousChunkBoss|MagicCrystalBallBoss|MimicChest|PlayerShield|PoisonStatusEffect|Portal|Servant Of K'Tesh|Shield|ShieldCrystal|Shriekgazer|Swordsman|TerraShark|Viper|WindGuardian|WindGuardianShield"

	ProjectileIndexes = "AxebeakBossEggProjectile|BladeSlash|BleedingEffect|CrystalTurretProjectile|DiamondDestructoProjectile|DragonsBreathProjectile|DreadGazerProjectile|FireballLevel2|FireballLevel3|IceGolemProjectile|JellyBossDash|JellyBossNPCFire|JellyBossTrail|JellySlash|LifeSteal|PlayerBaseProjectile|PlayerBladeSlash|PlayerBladeSlash2|PlayerBladeSlashSmall|ProjectileExplosion|ServantOfKTeshArms|ShriekgazerProjectile|SwordsmanSlash|ToxicBlast|ViperProjectile|WindGuardianProjectile"

	AbilityIndexes = "ITEM_ID_DUNGEON_ABILITY_BLOOD_OATH|ITEM_ID_DUNGEON_ABILITY_EXPLODING_PUNCH|ITEM_ID_DUNGEON_ABILITY_FIREBALL|ITEM_ID_DUNGEON_ABILITY_FIREBALL_LEVEL_2|ITEM_ID_DUNGEON_ABILITY_FIREBALL_LEVEL_3|ITEM_ID_DUNGEON_ABILITY_LIFE_STEAL|ITEM_ID_DUNGEON_ABILITY_SLASH_3|ITEM_ID_DUNGEON_ABILITY_SLASH_CRITICAL|ITEM_ID_DUNGEON_ABILITY_SLASH_PLUS|ITEM_ID_DUNGEON_ABILITY_SWORD_SLASH|ITEM_ID_DUNGEON_ABILITY_TOXIC_BLAST"
)

// ─────────────────────────────────────────────────────────────────────────────
// Dungeon Handlers (Domain: VallenDungeon)
// ─────────────────────────────────────────────────────────────────────────────

// HandleDungeonMenu membuka popup awal Dungeon
func HandleDungeonMenu(c *vallenctx.Ctx) {
	if c.Player == nil {
		return
	}
	c.Dialog(StartDialog())
}

// HandleDungeonBackpack membuka tas dungeon saat ikon tas di-tap
func HandleDungeonBackpack(c *vallenctx.Ctx, dm *Manager) {
	if c.Player == nil || c.World == nil || dm == nil {
		return
	}
	run, ok := dm.Get(c.Player.GrowID)
	if !ok || run.WorldName != c.World.Name {
		c.Error("Dungeon backpack is only available during a dungeon run.")
		return
	}
	c.Dialog(BackpackDialog(run))
}

// HandleDungeonShopBuy memproses pembelian upgrade ability di Lich Shop
func HandleDungeonShopBuy(c *vallenctx.Ctx, dm *Manager, item string) {
	if c.Player == nil || c.World == nil || dm == nil {
		return
	}
	run, ok := dm.Get(c.Player.GrowID)
	if !ok || run.WorldName != c.World.Name {
		return
	}

	cost := 50
	abilityName := ""
	switch item {
	case "buy_slash":
		cost = 50
		abilityName = "Sword Slash (+10 Dmg)"
	case "buy_fireball":
		cost = 100
		abilityName = "Fireball (+25 Dmg)"
	case "buy_lifesteal":
		cost = 150
		abilityName = "Life Steal (10%)"
	case "buy_hp":
		cost = 50
		abilityName = "+100 Max HP"
	}

	if run.Souls < cost {
		c.Bubble("`4Not enough Souls! Need %d Souls.``", cost)
		return
	}

	updated := dm.AddAbility(c.Player.GrowID, abilityName, cost)
	if updated != nil {
		c.SendRawPacket(variant.New("OnSetDungeonSouls", int32(updated.Souls)).Pack(), true)
		c.Success("Purchased %s for %d Souls!", abilityName, cost)
		c.Dialog(LichShopDialog(updated))
		c.Sound("audio/powerup.wav")
	}
}

// HandleDungeonDialogReturn menangani aksi dari dialog dungeon
func HandleDungeonDialogReturn(c *vallenctx.Ctx, dm *Manager, onStartRun func()) {
	if c.Player == nil {
		return
	}
	btn := c.Button()
	switch btn {
	case "dungeon_start":
		if onStartRun != nil {
			onStartRun()
		}
	case "dungeon_info":
		c.Dialog(InfoDialog())
	case "dungeon_shop":
		run, _ := dm.Get(c.Player.GrowID)
		c.Dialog(LichShopDialog(run))
	case "buy_slash", "buy_fireball", "buy_lifesteal", "buy_hp":
		HandleDungeonShopBuy(c, dm, btn)
	}
}
