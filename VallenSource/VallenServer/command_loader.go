package server

// command_loader.go - Auto-loader untuk VallenCommand
// Cukup satu import ini, semua command di player.go, admin.go, dll
// akan otomatis terdaftar saat server start tanpa perlu import manual lagi.
import (
	_ "gtps/VallenSource/VallenCommand"
)
