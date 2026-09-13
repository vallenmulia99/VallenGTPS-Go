package server

import (
	"fmt"

	player "gtps/VallenSource/VallenPlayer"
)

// avatarSpawnFields returns the established text fields used by OnSpawn.
func avatarSpawnFields(p *player.Player) string {
	if p == nil {
		return ""
	}
	hairColor := uint32(0)
	if p.HairColor != 0 && p.HairColor != 0xFFFFFFFF && p.HairColor != 0xFF00FF00 {
		hairColor = p.HairColor
	}
	text := fmt.Sprintf("skinColor|%d\nhairColor|%d\npunchID|%d\n", p.SkinColor, hairColor, p.PunchEffect)
	for i, itemID := range p.Clothing {
		text += fmt.Sprintf("clothing%d|%d\n", i, itemID)
	}
	return text
}
