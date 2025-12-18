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

type WorkTimer struct {
	startTime      time.Time // 勤務開始時間
	hourlyWage     float64   // 時給
	elpasedSeconds int       // 稼働時間(秒)
	quit           *systray.MenuItem
}

type WorkConfig struct {
	RegularSeconds int
	OvertimeRate   float64
	NightStart     int
	NightEnd       int
}

const (
	RegularWorkingSeconds = 32400 // 9時間 = 32400秒
	OvertimeRate          = 1.25  // 残業割増率
	NightStartHour        = 22    // 深夜開始時間
	NightEndHour          = 6     // 深夜終了時間
)

func NewWorkConfig() WorkConfig {
	return WorkConfig{
		RegularSeconds: RegularWorkingSeconds,
		OvertimeRate:   OvertimeRate,
		NightStart:     NightStartHour,
		NightEnd:       NightEndHour,
	}
}

func NewWorkTimer(startTime time.Time, houryWage float64, quit *systray.MenuItem) *WorkTimer {
	return &WorkTimer{
		startTime:      startTime,
		hourlyWage:     houryWage,
		elpasedSeconds: 0,
		quit:           quit,
	}
}

func getUserInput() (time.Time, float64) {
	scanner := bufio.NewScanner(os.Stdin)

	var startTime time.Time
	fmt.Println("勤務開始時間 (HH:MM)")
	for {
		if !scanner.Scan() {
			fmt.Println("入力の読み取りに失敗しました")
			return time.Time{}, 0
		}

		now := time.Now()
		timeStr := fmt.Sprintf("%d-%02d-%02d %s", now.Year(), now.Month(), now.Day(), scanner.Text())
		t, err := time.ParseInLocation("2006-01-02 15:04", timeStr, now.Location())
		if err != nil {
			fmt.Println("時間の形式が正しくありません (例: 09:30)")
			continue
		}
		startTime = t
		break
	}

	fmt.Println("ーーーーーーーーーーーーーーーーーーーー")
	fmt.Println("時給(円)")
	for {
		if !scanner.Scan() {
			fmt.Println("入力の読み取りに失敗しました")
			return time.Time{}, 0
		}

		hourlyWage, err := strconv.ParseFloat(scanner.Text(), 64)
		if err != nil {
			fmt.Println("数値で入力してください")
			continue
		}

		if hourlyWage <= 0 {
			fmt.Println("時給は0より大きい値を入力してください")
			continue
		}

		return startTime, hourlyWage
	}
}

func (w *WorkTimer) updateStatusForStartedWork(duration time.Duration) {
	// durationを絶対値化し、workTimerの経過時間に加算する
	w.elpasedSeconds += int(math.Abs(duration.Seconds()))

	go func() {
		// 1s間隔のtickerを起動する。
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				w.elpasedSeconds += 1
				w.calculateAndUpdateTitle()
			case <-w.quit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()

}

func (w *WorkTimer) updateStatusForNotStartedWork(duration time.Duration) {
	systray.SetTitle("勤務開始前")

	go func() {
		timer := time.NewTimer(duration)
		<-timer.C

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				w.elpasedSeconds += 1
				w.calculateAndUpdateTitle()
			case <-w.quit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func (w *WorkTimer) calculateAndUpdateTitle() {
	workConfig := NewWorkConfig()

	nowTime := time.Now().Local().Hour()
	isNightTime := nowTime >= workConfig.NightStart || nowTime < workConfig.NightEnd

	hourlyWagePerSeconds := float64(w.hourlyWage) / 3600
	overTimeSeconds := w.elpasedSeconds - workConfig.RegularSeconds

	var totalEarnings float64
	var title string

	// 正社員は8時間+休憩1時間の間拘束され、休憩１時間は給料として形状されない
	switch {
	// ケース1: 定時 (通常)
	case w.elpasedSeconds < workConfig.RegularSeconds && !isNightTime:
		totalEarnings = float64(w.elpasedSeconds) * hourlyWagePerSeconds * (8.0 / 9.0)
		title = "現在の稼ぎ"

	// ケース2: 残業 (1.25倍)
	case workConfig.RegularSeconds <= w.elpasedSeconds && !isNightTime:
		regularEarnings := float64(workConfig.RegularSeconds) * hourlyWagePerSeconds * (8.0 / 9.0)
		overtimeEarnings := float64(overTimeSeconds) * hourlyWagePerSeconds * 1.25
		totalEarnings = regularEarnings + overtimeEarnings
		title = "残業ブースト中🔥"

	// ケース3: 定時で深夜 (1.25倍)
	case w.elpasedSeconds < workConfig.RegularSeconds && isNightTime:
		totalEarnings = float64(w.elpasedSeconds) * hourlyWagePerSeconds * 1.25 * (8.0 / 9.0)
		title = "深夜勤務中🌙"

	// ケース4: 残業で深夜 (定時分 + 残業分×1.5倍)
	case workConfig.RegularSeconds <= w.elpasedSeconds && isNightTime:
		// 定時分は通常計算
		regularEarnings := float64(workConfig.RegularSeconds) * hourlyWagePerSeconds * (8.0 / 9.0)
		// 残業分は1.5倍（残業1.25 + 深夜0.25 = 1.5）
		overtimeEarnings := float64(overTimeSeconds) * hourlyWagePerSeconds * 1.5
		totalEarnings = regularEarnings + overtimeEarnings
		title = "深夜残業ブースト中🔥🌙"

	default:
		log.Println("計算出来てないよ")
		return
	}

	systray.SetTitle(fmt.Sprintf("%s ¥%.2f", title, totalEarnings))
}

func main() {
	startTime, hourylWage := getUserInput()

	systray.Run(func() {
		systray.SetTitle("💴 読み込み中")
		mQuit := systray.AddMenuItem("終了", "Quit the application")

		duration := time.Until(startTime)
		workTime := NewWorkTimer(startTime, hourylWage, mQuit)
		switch {
		case 0 < duration:
			workTime.updateStatusForNotStartedWork(duration)
		case duration < 0:
			workTime.updateStatusForStartedWork(duration)
		default:
			return
		}
	}, func() {})

}
