package boss

import (
	"encoding/json"
	"io/ioutil"
	"os"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	tileDest       rl.Rectangle
	tileSrc        rl.Rectangle
	BossMap        JsonMap
	SpritesheetMap rl.Texture2D
	tex            rl.Texture2D
	Out            []Tile
	Background     []Tile
	Ground         []Tile
	Props          []Tile
	Spawn          []Tile

	colliders []rl.Rectangle
)

type JsonMap struct {
	Layers    []Layer `json:"layers"`
	MapHeight int     `json:"mapHeight"`
	MapWidth  int     `json:"mapWidth"`
	TileSize  int     `json:"tileSize"`
}

type Layer struct {
	Name     string `json:"name"`
	Tiles    []Tile `json:"tiles"`
	Collider bool   `json:"collider"`
}

type Tile struct {
	Id string `json:"id"`
	X  int    `json:"x"`
	Y  int    `json:"y"`
}

func LoadMap(mapFile string) {
	file, err := os.Open(mapFile)

	if err != nil {
		panic(err)
	}

	defer file.Close()

	byteValue, _ := ioutil.ReadAll(file)

	json.Unmarshal(byteValue, &BossMap)

	buildColliders()
}

func Init() {
	SpritesheetMap = rl.LoadTexture("assets/boss/spritesheet.png")
	tileDest = rl.NewRectangle(0, 0, 16, 16)
	tileSrc = rl.NewRectangle(0, 0, 16, 16)
}

func Draw() {
	for i := 0; i < len(BossMap.Layers); i++ {
		if BossMap.Layers[i].Name == "out" {
			Out = BossMap.Layers[i].Tiles
		}
		if BossMap.Layers[i].Name == "background" {
			Background = BossMap.Layers[i].Tiles
		}
	}

	rl.DrawTexturePro(tex, tileSrc, tileDest, rl.NewVector2(0, 0), 0, rl.White)

	renderLayer(Out)
	renderLayer(Background)
}

func renderLayer(Layer []Tile) {
	for i := 0; i < len(Layer); i++ {
		tex = SpritesheetMap
		texColumns := tex.Width / int32(BossMap.TileSize)
		tileId := int64(0)
		// parse id to int, map.json uses strings like world
		for j := 0; j < len(Layer[i].Id); j++ {
			c := Layer[i].Id[j]
			if c < '0' || c > '9' {
				continue
			}
			tileId = tileId*10 + int64(c-'0')
		}

		tileSrc.X = float32(BossMap.TileSize) * float32((tileId)%int64(texColumns))
		tileSrc.Y = float32(BossMap.TileSize) * float32((tileId)/int64(texColumns))

		tileDest.X = float32(Layer[i].X * BossMap.TileSize)
		tileDest.Y = float32(Layer[i].Y * BossMap.TileSize)

		rl.DrawTexturePro(tex, tileSrc, tileDest, rl.NewVector2(0, 0), 0, rl.White)
	}
}

func Unload() {
	rl.UnloadTexture(SpritesheetMap)
}

func GetSpawnPosition() rl.Vector2 {
	for i := 0; i < len(BossMap.Layers); i++ {
		if BossMap.Layers[i].Name == "spawn" {
			if len(BossMap.Layers[i].Tiles) > 0 {
				t := BossMap.Layers[i].Tiles[0]
				return rl.NewVector2(float32(t.X*BossMap.TileSize), float32(t.Y*BossMap.TileSize))
			}
			break
		}
	}
	return rl.NewVector2(0, 0)
}

// GetColliders returns rectangles for all tiles from layers where Collider=true
func GetColliders() []rl.Rectangle {
	return colliders
}

func buildColliders() {
	colliders = colliders[:0]
	if len(BossMap.Layers) == 0 {
		return
	}
	for i := 0; i < len(BossMap.Layers); i++ {
		layer := BossMap.Layers[i]
		if !layer.Collider || len(layer.Tiles) == 0 {
			continue
		}
		for j := 0; j < len(layer.Tiles); j++ {
			t := layer.Tiles[j]
			r := rl.NewRectangle(float32(t.X*BossMap.TileSize), float32(t.Y*BossMap.TileSize), float32(BossMap.TileSize), float32(BossMap.TileSize))
			colliders = append(colliders, r)
		}
	}
}
