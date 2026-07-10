package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type jobSound int

const (
	jobSoundSuccess jobSound = iota
	jobSoundFailure
)

type soundNote struct {
	frequency float64
	duration  float64
}

var (
	soundFilesOnce sync.Once
	soundFiles     map[jobSound]string
)

func playJobSoundsCmd(sounds []jobSound) tea.Cmd {
	if len(sounds) == 0 {
		return nil
	}
	queued := append([]jobSound(nil), sounds...)
	return func() tea.Msg {
		for _, sound := range queued {
			playJobSound(sound)
		}
		return nil
	}
}

func playJobSound(sound jobSound) {
	soundFilesOnce.Do(createSoundFiles)
	path := soundFiles[sound]
	if path == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := soundPlayerCommand(ctx, path)
	if cmd != nil {
		_ = cmd.Run()
	}
}

func createSoundFiles() {
	soundFiles = make(map[jobSound]string, 2)
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	dir = filepath.Join(dir, "glab-pipelines", "sounds")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}

	melodies := map[jobSound][]soundNote{
		jobSoundSuccess: {
			{frequency: 659.25, duration: 0.08},
			{frequency: 783.99, duration: 0.08},
			{frequency: 1046.50, duration: 0.18},
		},
		jobSoundFailure: {
			{frequency: 293.66, duration: 0.14},
			{frequency: 220.00, duration: 0.22},
		},
	}
	for sound, melody := range melodies {
		name := "success.wav"
		if sound == jobSoundFailure {
			name = "failure.wav"
		}
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, synthesizeWAV(melody), 0o644); err == nil {
			soundFiles[sound] = path
		}
	}
}

func synthesizeWAV(melody []soundNote) []byte {
	const sampleRate = 44100
	const gapSeconds = 0.018
	gapSamples := int(math.Round(gapSeconds * float64(sampleRate)))
	var samples []int16
	for _, note := range melody {
		count := int(note.duration * sampleRate)
		for i := 0; i < count; i++ {
			t := float64(i) / sampleRate
			envelope := math.Min(1, float64(i)/(sampleRate*0.012))
			envelope = math.Min(envelope, float64(count-i)/(sampleRate*0.035))
			wave := math.Sin(2*math.Pi*note.frequency*t) + 0.18*math.Sin(4*math.Pi*note.frequency*t)
			samples = append(samples, int16(0.24*envelope*wave*math.MaxInt16))
		}
		samples = append(samples, make([]int16, gapSamples)...)
	}

	var data bytes.Buffer
	for _, sample := range samples {
		_ = binary.Write(&data, binary.LittleEndian, sample)
	}
	var wav bytes.Buffer
	wav.WriteString("RIFF")
	_ = binary.Write(&wav, binary.LittleEndian, uint32(36+data.Len()))
	wav.WriteString("WAVEfmt ")
	_ = binary.Write(&wav, binary.LittleEndian, uint32(16))
	_ = binary.Write(&wav, binary.LittleEndian, uint16(1))
	_ = binary.Write(&wav, binary.LittleEndian, uint16(1))
	_ = binary.Write(&wav, binary.LittleEndian, uint32(sampleRate))
	_ = binary.Write(&wav, binary.LittleEndian, uint32(sampleRate*2))
	_ = binary.Write(&wav, binary.LittleEndian, uint16(2))
	_ = binary.Write(&wav, binary.LittleEndian, uint16(16))
	wav.WriteString("data")
	_ = binary.Write(&wav, binary.LittleEndian, uint32(data.Len()))
	wav.Write(data.Bytes())
	return wav.Bytes()
}

func soundPlayerCommand(ctx context.Context, path string) *exec.Cmd {
	switch runtime.GOOS {
	case "darwin":
		return exec.CommandContext(ctx, "afplay", path)
	case "linux":
		for _, player := range []struct {
			name string
			args []string
		}{
			{name: "paplay", args: []string{path}},
			{name: "aplay", args: []string{path}},
			{name: "ffplay", args: []string{"-nodisp", "-autoexit", "-loglevel", "quiet", path}},
		} {
			if _, err := exec.LookPath(player.name); err == nil {
				return exec.CommandContext(ctx, player.name, player.args...)
			}
		}
	case "windows":
		player := "powershell"
		if _, err := exec.LookPath(player); err != nil {
			player = "pwsh"
		}
		escapedPath := strings.ReplaceAll(path, "'", "''")
		return exec.CommandContext(ctx, player, "-NoProfile", "-NonInteractive", "-Command", fmt.Sprintf("(New-Object Media.SoundPlayer '%s').PlaySync()", escapedPath))
	}
	return nil
}
