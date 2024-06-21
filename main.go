package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"time"

	"math"
	"encoding/binary"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/mesilliac/pulse-simple"
    _ "embed"
)

const RES int = 400

type Game struct {
	generation int
	board      [][]int
}

var (
	g *Game
	audio *Audio
)

// A board with empty state
func emptyGeneration() *Game {
	board := make([][]int, RES)
	for i := 0; i < RES; i++ {
		board[i] = make([]int, RES)
	}
	return &Game{board: board, generation: 1}
}

// Given an empty board, give it a random state
func giveState(g *Game) {
	rand.Seed(time.Now().UnixNano())
	for x := 0; x < RES; x++ {
		for y := 0; y < RES; y++ {
			if rand.Intn(15) == 1 {
				g.board[x][y] = 1
			}
		}
	}
}

// Apply the rules to a game's generation
// It returns the next generation
func logic(g *Game) *Game {
	n := emptyGeneration() // Next generation
	n.generation = g.generation + 1
	for x := 0; x < RES; x++ {
		for y := 0; y < RES; y++ {
			neighbors := checkNeighbors(x, y, g)
			live := g.board[x][y] == 1
			// Any live cell with fewer than two live neighbors dies, as if by underpopulation
			if live && neighbors < 2 {
				n.board[x][y] = 0
			}
			// Any live cell with two or three live neighbors lives on to the next generation
			if live && (neighbors == 2 || neighbors == 3) {
				n.board[x][y] = 1
			}
			// Any live cell with more than three live neighbors dies, as if by overpopulation
			if live && neighbors > 3 {
				n.board[x][y] = 0
			}
			// Any dead cell with exactly three live neighbors becomes a live cell, as if by reproduction
			if !live && neighbors == 3 {
				n.board[x][y] = 1
			}
		}
	}
	return n
}

// Given a position and a game
// Get the number of live neighbors at that position
func checkNeighbors(x int, y int, g *Game) int {
	neighbors := 0
	if y+1 < RES && g.board[x][y+1] == 1 { // top
		neighbors += 1
	}
	if y+1 < RES && x+1 < RES && g.board[x+1][y+1] == 1 { // top right
		neighbors += 1
	}
	if x+1 < RES && g.board[x+1][y] == 1 { // right
		neighbors += 1
	}
	if x+1 < RES && y-1 > 0 && g.board[x+1][y-1] == 1 { // bottom right
		neighbors += 1
	}
	if y-1 > 0 && g.board[x][y-1] == 1 { // bottom
		neighbors += 1
	}
	if x-1 > 0 && y-1 > 0 && g.board[x-1][y-1] == 1 { // bottom left
		neighbors += 1
	}
	if x-1 > 0 && g.board[x-1][y] == 1 { // left
		neighbors += 1
	}
	if x-1 > 0 && y+1 < RES && g.board[x-1][y+1] == 1 { // top left
		neighbors += 1
	}
	return neighbors
}

// Draw the game onto a black background
func draw(g *Game, background *ebiten.Image) {
	for x := 0; x < RES; x++ {
		for y := 0; y < RES; y++ {
			if g.board[x][y] == 1 {
				ebitenutil.DrawRect(background, float64(x), float64(y), 1, 1, color.White)
			}
		}
	}
}

// Place live cells around a point
func interaction(x int, y int, g *Game) *Game {
	x = clamp(x, 0, RES-1)
	y = clamp(y, 0, RES-1)
	topX, topY := x, clamp(y+1, 0, RES-1)
	leftX, leftY := clamp(x-1, 0, RES-1), y
	botX, botY := x, clamp(y-1, 0, RES-1)
	rightX, rightY := clamp(x+1, 0, RES-1), y
	g.board[x][y] = 1
	g.board[topX][topY] = 1
	g.board[leftX][leftY] = 1
	g.board[botX][botY] = 1
	g.board[rightX][rightY] = 1
	return g
}

func update(screen *ebiten.Image) error {
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		log.Printf("x: %v y: %v", x, y)
		interaction(x, y, g)
		audio.PlayBeep()
	}

	if ebiten.IsDrawingSkipped() {
		return nil
	}

	screen.Fill(color.RGBA{0, 0, 0, 0xff})
	background, _ := ebiten.NewImage(RES, RES, ebiten.FilterDefault)
	g = logic(g)
	draw(g, background)
	screen.DrawImage(background, &ebiten.DrawImageOptions{})
	ebitenutil.DebugPrint(screen, fmt.Sprintf("Generation: %v", g.generation))
	return nil
}

type Audio struct {
    stream         *pulse.Stream
    sampleRate     float64
    outputChannels int
    channel        chan float32
}

func NewAudio() *Audio {
    a := &Audio{
        channel: make(chan float32, 1024),
    }
    return a
}

func (a *Audio) Start() error {
    ss := pulse.SampleSpec{
        Format:   pulse.SAMPLE_FLOAT32LE,
        Rate:     44100,
        Channels: 1,
    }
    stream, err := pulse.Playback("Conway's Game of Life", "Audio Stream", &ss)
    if err != nil {
        return fmt.Errorf("PulseAudio 스트림 생성 실패: %v", err)
    }
    a.stream = stream
    a.sampleRate = float64(ss.Rate)
    a.outputChannels = int(ss.Channels)
    go a.play()
    return nil
}

func (a *Audio) play() {
    for {
        buf := make([]float32, 1024)
        for i := range buf {
            select {
            case sample := <-a.channel:
                buf[i] = sample
            default:
                buf[i] = 0
            }
        }
        byteBuf := make([]byte, len(buf)*4)
        for i, sample := range buf {
            binary.LittleEndian.PutUint32(byteBuf[i*4:], math.Float32bits(sample))
        }
        a.stream.Write(byteBuf)
    }
}

func (a *Audio) Stop() error {
    if a.stream != nil {
        a.stream.Free()
    }
    return nil
}

func (a *Audio) PlayBeep() {
    duration := 0.01
    freq := 440.0
    for i := 0; i < int(duration * a.sampleRate); i++ {
        t := float64(i) / a.sampleRate
        sample := float32(math.Sin(2 * math.Pi * freq * t) * 0.5)
        a.channel <- sample
    }
}

func main() {
	g = emptyGeneration()
	giveState(g)

	audio = NewAudio()
    err := audio.Start()
    if err != nil {
        log.Printf("오디오 초기화 실패: %v", err)
    }
	defer audio.Stop()

	if err := ebiten.Run(update, RES, RES, 2, "Conway's Game of Life"); err != nil {
		log.Fatal(err)
	}
}

func clamp(x int, min int, max int) int {
	if x < min {
		return min
	} else if x > max {
		return max
	}
	return x
}
