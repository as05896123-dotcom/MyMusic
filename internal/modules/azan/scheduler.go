// modules/azan/scheduler.go
package azan

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
	"github.com/robfig/cron/v3"

	"main/internal/config"
	"main/internal/core"
	"main/internal/platforms"
)

// كائنات خاصة بجدولة الأذان
var (
	Scheduler *cron.Cron
	BotClient *telegram.Client
)

// تهيئة جدولة الأذان (تشغيل المهام الدورية)
func InitAzanScheduler(client *telegram.Client) {
	BotClient = client

	loc, err := time.LoadLocation("Africa/Cairo")
	if err != nil {
		log.Println("Loc error, using local time:", err)
		loc = time.Local
	}
	Scheduler = cron.New(cron.WithLocation(loc))

	// جدولة تحديث مواقيت الصلاة يوميًا عند منتصف الليل وخروج أذكار الصباح والمساء في الأوقات المحددة
	_, _ = Scheduler.AddFunc("5 0 * * *", UpdateAzanTimes)
	_, _ = Scheduler.AddFunc("0 7 * * *", func() {
		BroadcastDuas(MorningDuas, "أذكار الصباح")
	})
	_, _ = Scheduler.AddFunc("0 20 * * *", func() {
		BroadcastDuas(NightDuas, "أذكار المساء")
	})

	go UpdateAzanTimes()
	Scheduler.Start()
}

// تحديث مواقيت الصلاة من API وإعادة جدولة الأذان تبعًا للأوقات الجديدة
func UpdateAzanTimes() {
	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("http://api.aladhan.com/v1/timingsByCity?city=Cairo&country=Egypt&method=5")
	if err != nil {
		log.Println("HTTP error:", err)
		return
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Timings map[string]string `json:"timings"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Println("Decode error:", err)
		return
	}

	if Scheduler != nil {
		loc := Scheduler.Location()
		Scheduler.Stop()
		Scheduler = cron.New(cron.WithLocation(loc))
	}

	// إعادة جدولة المهام الدورية نفسها
	_, _ = Scheduler.AddFunc("5 0 * * *", UpdateAzanTimes)
	_, _ = Scheduler.AddFunc("0 7 * * *", func() {
		BroadcastDuas(MorningDuas, "أذكار الصباح")
	})
	_, _ = Scheduler.AddFunc("0 20 * * *", func() {
		BroadcastDuas(NightDuas, "أذكار المساء")
	})

	// جدولة بث الأذان لكل صلاة حسب التوقيت الجديد
	for prayerKey, link := range PrayerLinks {
		timeStr, ok := result.Data.Timings[prayerKey]
		if !ok {
			continue
		}
		parts := strings.Split(strings.Split(timeStr, " ")[0], ":")
		if len(parts) != 2 {
			continue
		}
		h, _ := strconv.Atoi(parts[0])
		m, _ := strconv.Atoi(parts[1])

		pk := prayerKey
		pl := link
		spec := fmt.Sprintf("%d %d * * *", m, h)
		_, _ = Scheduler.AddFunc(spec, func() {
			BroadcastAzan(pk, pl)
		})
	}

	Scheduler.Start()
	log.Println("Azan times updated")
}

// بث الأذان إلى جميع المجموعات المفعّلة
func BroadcastAzan(prayerKey, link string) {
	chats, err := GetAllActiveChats()
	if err != nil {
		return
	}
	for _, chat := range chats {
		if enabled, ok := chat.Prayers[prayerKey]; ok && !enabled {
			continue
		}
		go StartAzanStream(chat.ChatID, prayerKey, link, false)
	}
}

// بث عشوائي لذكر من قائمة الأذكار لكل المجموعات المفعّلة
func BroadcastDuas(duas []string, title string) {
	if len(duas) == 0 {
		return
	}
	chats, err := GetAllActiveChats()
	if err != nil {
		return
	}
	dua := duas[rand.Intn(len(duas))]
	for _, chat := range chats {
		settings, err := GetChatSettings(chat.ChatID)
		if err != nil || !settings.DuaActive {
			continue
		}
		_, _ = BotClient.SendMessage(chat.ChatID, &telegram.SendMessageOptions{
			Text: fmt.Sprintf(
				"💫 **%s**\n\n%s\n\n<b>تـقـبـل الله مـنـا ومـنـكـم صـالـح الأعـمـال 🧚</b>",
				title,
				dua,
			),
		})
	}
}

// بدء بث الأذان في المكالمة الجماعية (باستخدام core و platforms)
func StartAzanStream(chatID int64, prayerKey, link string, forceTest bool) {
	cs, err := core.GetChatState(chatID)
	if err != nil {
		return
	}

	active, _ := cs.IsActiveVC()
	if !active {
		assistant := core.Assistants.Get(chatID)
		if assistant == nil {
			return
		}
		_ = assistant.PhoneCreateGroupCall(chatID, "")
		time.Sleep(3 * time.Second)
	}
	if present, _ := cs.IsAssistantPresent(); !present {
		_ = cs.TryJoin()
		time.Sleep(2 * time.Second)
	}

	if stickerID, ok := PrayerStickers[prayerKey]; ok {
		_, _ = BotClient.SendSticker(chatID, &telegram.SendStickerOptions{
			Sticker: &telegram.InputFileID{ID: stickerID},
		})
	}
	statusMsg, err := BotClient.SendMessage(chatID, &telegram.SendMessageOptions{
		Text: fmt.Sprintf(
			"🕌 **حـان الآن مـوعـد أذان %s**\n<b>بـالتـوقـيـت الـمـحـلـي لـمـديـنـة الـقـاهـره 🧚</b>",
			PrayerNamesStretched[prayerKey],
		),
	})
	if err != nil {
		return
	}

	dummyMsg := &telegram.NewMessage{
		Client: BotClient,
		Message: &telegram.Message{
			Chat:   &telegram.Chat{ID: chatID},
			Text:   link,
			Sender: &telegram.Peer{ID: config.OwnerID},
		},
	}
	tracks, err := platforms.GetTracks(dummyMsg, false)
	if err != nil || len(tracks) == 0 {
		_, _ = BotClient.DeleteMessages(chatID, []int{statusMsg.ID})
		return
	}

	track := tracks[0]
	track.Requester = "خـدمـة الأذان"
	path, err := platforms.Download(context.Background(), track, statusMsg)
	if err != nil {
		return
	}

	if room := core.GetRoom(chatID); room != nil {
		room.Play(track, path, true)
	}

	// إخفاء لوحة الأزرار بعد البث
	go hideAzanKeyboard(chatID)
}

// حذف لوحة الأزرار بعد انتهاء الإرسال
func hideAzanKeyboard(chatID int64) {
	for i := 0; i < 5; i++ {
		time.Sleep(800 * time.Millisecond)
		history, err := BotClient.GetHistory(chatID, 0, 0, 0, 3, 0, 0, 0)
		if err != nil || history == nil {
			continue
		}
		for _, m := range history.Messages {
			if m.Sender != nil && m.Sender.ID == BotClient.Self.ID && m.ReplyMarkup != nil {
				_, _ = BotClient.DeleteMessages(chatID, []int{m.ID})
				return
			}
		}
	}
}
