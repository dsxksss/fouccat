package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var (
	cmdMutex  sync.Mutex
	cmd       *exec.Cmd
	stdinPipe io.WriteCloser
)

const (
	videoSize    = "1920x1080"
	frameRate    = "60"
	probeSize    = "50M"
	preset       = "ultrafast"
	tune         = "zerolatency"
	outputPath   = "output.mp4"
	inputPath    = "input.mp4"
	scaledWidth  = 1280
	scaledHeight = 720
)

func handleError(err error, context string) {
	if err != nil {
		log.Printf("Error %s: %v\n", context, err)
	}
}

func StartRecording(outputPath string) error {
	cmdMutex.Lock()
	defer cmdMutex.Unlock()

	// 构建 ffmpeg 命令
	cmd = exec.Command("ffmpeg",
		"-f", "gdigrab",
		"-video_size", videoSize,
		"-framerate", frameRate,
		"-probesize", probeSize,
		"-i", "desktop",
		"-preset", preset,
		"-tune", tune,
		outputPath,
		"-y",
	)

	// 创建管道用于向 ffmpeg 发送输入
	var err error
	stdinPipe, err = cmd.StdinPipe()
	if err != nil {
		return err
	}

	// 设置输出到标准输出和标准错误
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 启动 ffmpeg 进程
	err = cmd.Start()
	if err != nil {
		return err
	}

	return nil
}

func StopRecording() error {
	cmdMutex.Lock()
	defer cmdMutex.Unlock()

	if cmd != nil {
		err := cmd.Process.Kill()
		if err != nil {
			return err
		}
		cmd = nil
	}

	return nil
}

func EndRecording() error {
	cmdMutex.Lock()
	defer cmdMutex.Unlock()

	if cmd != nil {
		// 发送 'q' 字符来优雅地结束录制
		_, err := stdinPipe.Write([]byte("q\n"))
		if err != nil {
			return err
		}
		stdinPipe.Close()
		cmd = nil
	}

	return nil
}

func ScaleVideo(inputPath, outputPath string, width, height int) error {
	// 检查输入文件是否存在
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		log.Printf("Input file does not exist: %s\n", inputPath)
		return err
	}

	log.Printf("Scaling video from: %s to: %s with resolution: %dx%d\n", inputPath, outputPath, width, height)
	err := exec.Command("ffmpeg",
		"-i", inputPath,
		"-vf", "scale="+strconv.Itoa(width)+":"+strconv.Itoa(height),
		outputPath,
		"-y",
	).Run()

	return err
}

func setUpUI(a fyne.App) {
	w := a.NewWindow("Screen Recorder")

	startBtn := widget.NewButton("Start Recording", func() {
		go func() {
			err := StartRecording(outputPath)
			handleError(err, "starting recording")
		}()
	})

	stopBtn := widget.NewButton("Stop Recording", func() {
		go func() {
			err := StopRecording()
			handleError(err, "stopping recording")
		}()
	})

	endBtn := widget.NewButton("End Recording", func() {
		go func() {
			err := EndRecording()
			handleError(err, "ending recording")
		}()
	})

	scaleBtn := widget.NewButton("Scale Video", func() {
		go func() {
			err := ScaleVideo(inputPath, "scaled_output.mp4", scaledWidth, scaledHeight)
			handleError(err, "scaling video")
		}()
	})

	w.SetContent(container.NewVBox(
		startBtn,
		stopBtn,
		endBtn,
		scaleBtn,
	))

	w.ShowAndRun()
}

func main() {
	// 获取 ffmpeg.exe 所在的目录
	ffmpegPath, err := filepath.Abs("./ffmpeg/bin")
	if err != nil {
		log.Fatalf("Failed to get absolute path: %v", err)
	}

	// 设置 PATH 环境变量
	os.Setenv("PATH", ffmpegPath+string(os.PathListSeparator)+os.Getenv("PATH"))

	a := app.New()
	setUpUI(a)
}
