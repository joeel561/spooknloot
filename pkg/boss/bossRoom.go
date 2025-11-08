package boss

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"spooknloot/pkg/world"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var (
	tileDest       rl.Rectangle
	tileSrc        rl.Rectangle
	BossMap        JsonMap
	SpritesheetMap rl.Texture2D
	tex            rl.Texture2D
	Out            []world.Tile
	FloorTiles     []world.Tile
	Spawn          []world.Tile
	Torch          []world.Tile
	Decoration     []world.Tile

	colliders []rl.Rectangle
)

type JsonMap struct {
	Layers    []Layer `json:"layers"`
	MapHeight int     `json:"mapHeight"`
	MapWidth  int     `json:"mapWidth"`
	TileSize  int     `json:"tileSize"`
}

type Layer struct {
	Name     string       `json:"name"`
	Tiles    []world.Tile `json:"tiles"`
	Collider bool         `json:"collider"`
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
		if BossMap.Layers[i].Name == "Walls" {
			Out = BossMap.Layers[i].Tiles
		}
		if BossMap.Layers[i].Name == "Floor" {
			FloorTiles = BossMap.Layers[i].Tiles
		}
		if BossMap.Layers[i].Name == "Torch" {
			Torch = BossMap.Layers[i].Tiles
		}

		if BossMap.Layers[i].Name == "Decoration" {
			Decoration = BossMap.Layers[i].Tiles
		}
	}

	rl.DrawTexturePro(tex, tileSrc, tileDest, rl.NewVector2(0, 0), 0, rl.White)

	renderLayer(FloorTiles)
	renderLayer(Out)
	renderLayer(Torch)
	renderLayer(Decoration)
}

func renderLayer(Layer []world.Tile) {
	for i := 0; i < len(Layer); i++ {
		tex = SpritesheetMap
		texColumns := tex.Width / int32(BossMap.TileSize)
		tileId := int64(0)
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
