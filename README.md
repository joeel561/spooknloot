# Spook’n’Loot

2D Pixel-Art Rougke Like Game. Du spawnst zuerst in einer Halloween Stadt und kannst dort Easer Eggs entdecken. Um das richtige Spiel zu starten musst du durch das mysteriös Gebäude. Dich erwarten 20 Level die du schaffen musst bis du zum Endboss kommst. Stirbst du faengt das Spiel komplett von vorne an. Release 31. Oct **Spiel befindet sich im Aufbau** Der ganze Prozess wird auf meinem Twitch Kanal gestreamt http://twitch.tv/joeel561/

---

![til](./assets/preview1.gif)

## 📦 Projektstruktur

```
.
├── assets/             # Ressourcen wie Grafiken / Texturen / Sound / etc.
├── cmd/
│   └── main.go         # Einstiegspunkt des Spiels
├── pkg/                 # Logik, Modelle, Hilfspakete
├── spooknloot.json      # Konfigurations- / Daten-Datei
├── go.mod
└── go.sum
```

---

![til](./assets/preview2.gif)

## 🚀 Installation & Ausführung

**Voraussetzung:** Go (Version 1.XX oder neuer) muss installiert sein.

1. Repository klonen:
   ```bash
   git clone https://github.com/joeel561/spooknloot.git
   cd spooknloot
   ```

2. Abhängigkeiten installieren (optional, meist reicht `go mod`):
   ```bash
   go mod tidy
   ```

3. Spiel starten:
   ```bash
   go run ./cmd
   ```

   oder (wenn du bauen willst):
   ```bash
   go build -o spooknloot ./cmd
   ./spooknloot
   ```

---

## 🎮 Spielbeschreibung & Features

- Erkunde dunkle Orte und Räume.    
- Begegne Gegnern / Herausforderungen.  
- Dynamisch Automatisch generierte Level via Autotiling
- jedes Level wird schwieriger 

---

## 🛠️ Architektur & Module

- **cmd/main.go**: Einstiegspunkt, liest ggf. Einstellungen, initialisiert und startet das Spiel.  
- **pkg/**: Enthält Subpakete für z. B. Spiel-Logik, Datenstrukturen, Hilfsfunktionen, Map-/Level‑Management etc.  
- **assets/**: Ressourcen (Grafiken, Sounds, Scripts etc.).  

---

## 🤝 Fork

Du kannst das Spiel gerne forken und selbst ausprobieren aber ein kleiner Hinweis die Grafiken habe ich bei https://franuka.itch.io/ gekauft 
Wenn du diese nutzen moechtest wuerde ich mich freuen wenn du Franuka supportest und die Grafiken ebenfalls kaufst.

---

## 🙏 Danke <3

- Danke an alle, die mich bei dem Weg unterstuetzen und Lust auf das Spiel haben wenn alles gut laeuft dann release ich das Spiel am 31.10 auf Steam. Du kannst es gerne vorab testen und auf Discord https://discord.gg/hNDA3TZF auch Feedback geben sollte etwas buggy bei dir sein. 

---

Viel Spaß beim ausprobieren von **Spook’n’Loot**!
