package events

// ─────────────────────────────────────────────────────────────────────────────
// SuperMain Event Definitions & Titles (Domain: VallenEvents)
// ─────────────────────────────────────────────────────────────────────────────

var SuperMainActions = map[string]string{
	"dailychallengemenu":        "Daily Challenge",
	"openPiggyBank":             "Piggy Bank",
	"showdungeonsui":            "Dungeon Scrolls",
	"show_mailbox_ui":           "Mailbox",
	"show_auction_ui":           "Auction House",
	"eventmenu":                 "Clash Event",
	"show_bingo_ui":             "Bingo",
	"winterrallymenu":           "Winter Rally",
	"leaderboardBtnClicked":     "Leaderboard",
	"euphoriaBtnClicked":        "Euphoria Event",
	"openLnySparksPopup":        "Lunar New Year",
	"ShowValentinesQuestDialog": "Valentine Quest",
	"showegseeventui":           "Easter Event",
	"openStPatrickPiggyBank":    "St. Patrick's Piggy Bank",
	"dailyrewardmenu":           "Daily Reward",
	"show_fruit_mixer_dialog":   "Fruit Mixer",
	"claimprogressbar":          "Seasonal Event",
}

func IsSuperMainAction(action string) bool {
	_, ok := SuperMainActions[action]
	return ok
}

var AvailableTitles = []string{
	"(None)",
	"the Legend",
	"the Master",
	"the Rich",
	"the Hero",
	"the Explorer",
	"the Pro",
	"the Builder",
	"the Champion",
	"the Guardian",
}
