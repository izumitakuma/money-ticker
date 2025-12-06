package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/getlantern/systray"
)

// 勤務開始時刻
var startTime time.Time

// 時給
var hourlyWage int
var err error

const (
	RegularWorkingSeconds = 32400 // 9時間 = 32400秒
	OvertimeRate          = 1.25  // 残業割増率
	NightStartHour        = 22    // 深夜開始時間
	NightEndHour          = 6     // 深夜終了時間
)

// 勤務開始からの経過時間(秒)
var leftSeconds int = 0

var title string

func main() {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("勤務開始時間 (HH:MM)")
	scanner.Scan()

	// 現在のタイムゾーンで今日の日付で時刻を解析
	now := time.Now()
	timeStr := fmt.Sprintf("%d-%02d-%02d %s", now.Year(), now.Month(), now.Day(), scanner.Text())
	startTime, err = time.ParseInLocation("2006-01-02 15:04", timeStr, now.Location())
	if err != nil {
		fmt.Println("時間の形式が正しくありません (例: 09:30)")
		return
	}

	fmt.Println("ーーーーーーーーーーーーーーーーーーーー")
	fmt.Println("時給(円)")
	scanner.Scan()
	hourlyWage, err = strconv.Atoi(scanner.Text())
	if err != nil {
		fmt.Println("数値で入力してください")
	}
	systray.Run(onReady, onExit)

}

func onReady() {
	systray.SetTitle("💴 読み込み中")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("終了", "Quit the application")

	// 過去の時間が入力された場合、負の値が返ってくる
	duration := time.Until(startTime)

	switch {
	// 既に勤務開始
	case duration < 0:
		updateStatusForStartedWork(duration, mQuit)
	// まだ勤務開始していない
	case 0 < duration:
		updateStatusForNotStartedWork(duration, mQuit)
	default:
	}
}

func onExit() {}

func updateStatusForStartedWork(duration time.Duration, mQuit *systray.MenuItem) {
	// 既に働いた分を加算
	leftSeconds += int(math.Abs(duration.Seconds())) // 絶対値として扱う

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				leftSeconds += 1
				updateStatus()
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}
func updateStatusForNotStartedWork(duration time.Duration, mQuit *systray.MenuItem) {
	timer := time.NewTimer(duration)

	go func() {
		systray.SetTitle("勤務開始前")

		<-timer.C

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				leftSeconds += 1
				updateStatus()
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func updateStatus() {
	// 現在稼いだ額をTitleに表示する
	systray.SetTitle(fmt.Sprintf("%s：¥%s", title, strconv.Itoa(currentEarnings())))

}

func currentEarnings() int {
	if leftSeconds < 0 {
		log.Printf("leftSeconds is minus")
		systray.SetTitle(fmt.Sprintln("💴 エラー"))
		return 0
	}
	// 現在稼いだ額
	var earning float64

	switch {
	case leftSeconds < RegularWorkingSeconds:
		title = "現在の稼ぎ"
		earning = float64(leftSeconds) * float64(hourlyWage) / 3600 // 秒給を浮動小数点で計算
	case RegularWorkingSeconds < leftSeconds:
		title = "現在の稼ぎ(残業ブースト中🔥)"
		earning = float64(leftSeconds) * float64(hourlyWage) * 1.25 / 3600
	case time.Now().Hour() >= NightStartHour || time.Now().Hour() < NightEndHour:
		title = "現在の稼ぎ(深夜残業ブースト中🔥)"
		earning = float64(leftSeconds) * float64(hourlyWage) * 1.25 / 3600
	}
	return int(math.Round(earning)) // 最後に丸める
}
