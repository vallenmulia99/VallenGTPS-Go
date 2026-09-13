package dungeon

import (
	"fmt"
	"strings"
)

// StartDialog mirrors the useful parts of DungeonStartUi.rml while the server
// dungeon system is being built: run cost, Souls, room count and safe start.
func StartDialog() string {
	return "set_default_color|`o\n" +
		"add_label_with_icon|big|`wDungeons``|left|14734|\n" +
		"add_textbox|`9Go on an adventure, explore dungeons and use their unique currency to upgrade your abilities. Just remember, what happens in the dungeon stays in the dungeon!``|left|\n" +
		"add_spacer|small|\n" +
		"add_textbox|`wDungeon Scrolls: `220/20``|left|\n" +
		"add_textbox|`wEntry Cost: `220 Scrolls``|left|\n" +
		"add_smalltext|`4Warning: Leaving the Dungeon at any point will prevent re-entry into the same world. All ability upgrades will be lost as well.``|left|\n" +
		"add_spacer|small|\n" +
		"add_button|dungeon_start|`2Start Dungeon``|noflags|0|0|\n" +
		"add_button|dungeon_info|`wDungeon Information``|noflags|0|0|\n" +
		"add_quick_exit|\n" +
		"end_dialog|dungeon_start_ui|Close||\n"
}

func BackpackDialog(r *Run) string {
	if r == nil {
		return ""
	}
	abilitiesStr := "None"
	if len(r.Abilities) > 0 {
		abilitiesStr = strings.Join(r.Abilities, ", ")
	}
	return fmt.Sprintf("set_default_color|`o\n"+
		"add_label_with_icon|big|`wPlayer Stats     ``|left||image:game/tiles_page16.rttex;frame:23,25;frameSize:32;|\n"+
		"add_spacer|small|\n"+
		"add_textbox|Current Room: %d / %d|left|\n"+
		"add_spacer|small|\n"+
		"add_spacer|small|\n"+
		"add_textbox|`5Your Current Stats:``|left|\n"+
		"add_textbox|Health Points: %d/%d|left|\n"+
		"add_textbox|Damage Points: %d|left|\n"+
		"add_textbox|Critical Hit Chance: %.1f%%|left|\n"+
		"add_textbox|Critical Hit Multiplier: x%.1f%%|left|\n"+
		"add_spacer|small|\n"+
		"add_textbox|`5Ability Upgrades:``|left|\n"+
		"add_textbox|`w%s``|left|\n"+
		"add_spacer|small|\n"+
		"add_button|dungeon_shop|`wLich's Spoooky Wares``|noflags|0|0|\n"+
		"add_quick_exit|\n"+
		"end_dialog|dungeon_stats|Close||\n",
		r.Room, MaxRooms, r.Health, r.MaxHealth, r.Damage, r.CritChance, r.CritMult, abilitiesStr)
}

func LichShopDialog(r *Run) string {
	souls := 0
	if r != nil {
		souls = r.Souls
	}
	return fmt.Sprintf("set_default_color|`o\n"+
		"add_label_with_icon|big|`wLich's Spoooky Wares``|left|14734|\n"+
		"add_spacer|small|\n"+
		"add_smalltext|Hark! Feel the chill'd breeze upon thee, the Lich doth offer wares to test thy bravery,|left|\n"+
		"add_smalltext|Abilities in abundance, yet souls it shall demand in great confidence,|left|\n"+
		"add_smalltext|Tread with caution, face lurking beasts from another dimension,|left|\n"+
		"add_spacer|small|\n"+
		"add_textbox|`2Your Dungeon Souls: `5%d``|left|\n"+
		"add_spacer|small|\n"+
		"add_button|buy_slash|`wSword Slash (+10 Dmg) `5[50 Souls]``|noflags|0|0|\n"+
		"add_button|buy_fireball|`wFireball (+25 Dmg) `5[100 Souls]``|noflags|0|0|\n"+
		"add_button|buy_lifesteal|`wLife Steal `5[150 Souls]``|noflags|0|0|\n"+
		"add_button|buy_hp|`w+100 Max HP `5[50 Souls]``|noflags|0|0|\n"+
		"add_quick_exit|\n"+
		"end_dialog|dungeon_lich_shop|Close||\n", souls)
}

func InfoDialog() string {
	return "set_default_color|`o\nadd_label_with_icon|big|`wDungeon Information``|left|242|\nadd_textbox|Clear five rooms, collect Souls, then buy temporary combat abilities.``|left|\nadd_textbox|`6All dungeon Souls, HP, abilities and backpack loot are cleared when the run ends.``|left|\nadd_quick_exit|\nend_dialog|dungeon_info|Close||\n"
}

func FinishDialog(run *Run) string {
	souls := 0
	if run != nil {
		souls = run.Souls
	}
	return fmt.Sprintf("set_default_color|`o\nadd_label_with_icon|big|`2Dungeon Complete!``|left|%d|\nadd_textbox|You cleared all five trial rooms.\nDungeon Souls earned: `2%d``\n\nBosses, loot rewards and dungeon ranks are the next phase.|left|\nend_dialog|dungeon_finish|Close||\n", 0, souls)
}
