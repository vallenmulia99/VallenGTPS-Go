<div align="center">
  <img src="icon-vallen/vallengtps.jpg" width="150" style="border-radius: 50%;" alt="Vallen Project" />

  <h1>🌿 VALLEN PROJECT GTPS 🌿</h1>

  <p>
    <img src="https://img.shields.io/badge/Language-Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" />
    <img src="https://img.shields.io/badge/C++-ENet-00599C?style=for-the-badge&logo=c%2B%2B&logoColor=white" />
    <img src="https://img.shields.io/badge/License-MIT-brightgreen?style=for-the-badge" />
    <img src="https://img.shields.io/badge/Status-Active-success?style=for-the-badge" />
    <a href="https://t.me/projectvallen">
      <img src="https://img.shields.io/badge/Telegram-projectvallen-2CA5E0?style=for-the-badge&logo=telegram&logoColor=white" />
    </a>
  </p>

  <p><i>Growtopia Private Server — Built from scratch, open source, made in Indonesia 🇮🇩</i></p>
</div>

---

## 🇮🇩 Tentang Project | 🇺🇸 About

**[ID]**
**GTPS (Growtopia Private Server)** adalah server Growtopia buatan sendiri yang berjalan secara independen tanpa bergantung pada server resmi Growtopia.
Project ini dikembangkan **dari nol** oleh **Vallen** menggunakan bahasa pemrograman **Go (Golang)** untuk server utama dan **C++ (ENet)** sebagai library jaringan UDP berkecepatan tinggi.

**[EN]**
**GTPS (Growtopia Private Server)** is a custom-built Growtopia server that runs independently without relying on the official Growtopia servers.
This project was built **from scratch** by **Vallen** using **Go (Golang)** as the main server language and **C++ (ENet)** as the high-performance UDP networking library.

---

## 📋 Persyaratan | Requirements

**[ID]** Sebelum build, pastikan sudah install:

**[EN]** Before building, make sure you have installed:

| Tool | Versi / Version | Link |
|------|-----------------|------|
| **Go** | 1.20+ | [golang.org](https://golang.org/dl/) |
| **GCC** (Windows) | MinGW-w64 | [winlibs.com](https://winlibs.com/) |
| **GCC** (Linux) | build-essential | `sudo apt install build-essential` |
| **items.dat** | Growtopia item DB | Letakkan di root folder |

---

## 🚀 Cara Build & Jalankan | How to Build & Run

### Windows
```bat
build_windows.bat
```

### Linux
```bash
chmod +x build_linux.sh
./build_linux.sh
```

### Manual
```bash
go build -o gtps-vallen .
./gtps-vallen
```

---

## 🌐 Koneksi ke Server | Connecting to Server

**[ID]** Agar client Growtopia bisa connect ke server kamu, edit file `server_data.php`:

**[EN]** For the Growtopia client to connect to your server, edit `server_data.php`:

```
server|YOUR_SERVER_IP
port|17091
type|1
type2|1
loginurl|your-login-url.vercel.app
meta|
RTENDMARKERBS1001
```

> **[ID]** Ganti `YOUR_SERVER_IP` dengan IP server kamu (bisa IP lokal `127.0.0.1` untuk testing lokal).
> **[EN]** Replace `YOUR_SERVER_IP` with your server's IP (use `127.0.0.1` for local testing).

---

## ⚙️ Konfigurasi Role & Setting | Role & Settings Config

**[ID]** Edit file `VallenSetting/setting.json` untuk mengatur nama server, owner, dan role:

**[EN]** Edit `VallenSetting/setting.json` to configure server name, owners, and roles:

```json
{
  "server_name": "NAMA SERVER KAMU",
  "owners": [
    "GROWID_KAMU"
  ],
  "default_role": 0,
  "enable_registration": true,
  "max_players": 1024
}
```

### Role System

| Role ID | Role | Keterangan |
|---------|------|------------|
| `0` | Player | Default — player biasa |
| `1` | VIP | Akses fitur VIP |
| `2` | Moderator | Bisa kick, ban player |
| `3` | Admin | Akses semua command |
| `4` | Owner | Full access |

> **[ID]** Untuk set role player, gunakan command in-game: `/grole <growid> <role_id>`
> **[EN]** To set a player's role, use the in-game command: `/grole <growid> <role_id>`

---

## 📁 Struktur Project | Project Structure

```
📦 Vallen Project GTPS
├── 📂 VallenSource/
│   ├── VallenCommand/     # Perintah admin & player
│   ├── VallenConfig/      # Konfigurasi server
│   ├── VallenDatabase/    # Layer database JSON
│   ├── VallenDialog/      # Handler dialog packet
│   ├── VallenDungeon/     # Sistem dungeon
│   ├── VallenHttps/       # HTTPS server (login)
│   ├── VallenItems/       # Parser items.dat
│   ├── VallenPlayer/      # Model data player
│   ├── VallenRole/        # Role & permission
│   ├── VallenServer/      # Core ENet server & handler
│   ├── VallenStore/       # Sistem toko
│   ├── VallenVariant/     # Builder variant packet
│   └── VallenWorld/       # World, tile, object
├── 📂 VallenSetting/      # Konfigurasi setting.json
├── 📂 resources/          # SSL cert & asset
├── 📂 enet/               # ENet C library (networking)
├── 🖥️ main.go             # Entry point
├── 📄 server_data.php     # Konfigurasi IP server
├── 📄 items.dat           # Growtopia item database
└── 🔨 build_windows.bat   # Build script Windows
```

---

## 🤝 Kontribusi | Contributing

**[ID]** Pull request sangat disambut! Buka issue terlebih dahulu untuk perubahan besar.

**[EN]** Pull requests are welcome! Please open an issue first for major changes.

---

## 👤 Developer

<div align="center">
  <img src="icon-vallen/vallengtps.jpg" width="80" style="border-radius:50%;" />
  <br/><br/>
  <strong>Vallen</strong>
  <br/>
  <i>Developed with ❤️ in Indonesia 🇮🇩</i>
  <br/><br/>
  <a href="https://t.me/projectvallen">
    <img src="https://img.shields.io/badge/Join%20Telegram-projectvallen-2CA5E0?style=for-the-badge&logo=telegram&logoColor=white" />
  </a>
</div>

---

## 📄 Lisensi | License

MIT — Lihat [LICENSE](LICENSE) untuk detail lengkap.

MIT — See [LICENSE](LICENSE) for full details.

---

## 🔎 Keywords / Tags

<!-- SEO: Vallen GTPS, Vallen Growtopia Private Server, vallen gtps go, vallen gtps golang, growtopia private server go, growtopia private server golang, GTPS Indonesia, growtopia server open source, vallen project gtps, gtps go, gtps golang, growtopia private server indonesia, vallen gtps github -->

**Vallen GTPS** | **Vallen Growtopia Private Server** | **GTPS Go** | **GTPS Golang** | **Growtopia Private Server Indonesia** | **Open Source GTPS** | **Vallen Project** | **growtopia-private-server** | **vallen-gtps**
