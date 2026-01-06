package azan

import (
	"context"
	"encoding/json"
	"fmt"
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

var Scheduler *cron.Cron
var BotClient *telegram.Client

func InitAzanScheduler(client *telegram.Client) {
	BotClient = client
	loc, _ := time.LoadLocation("Africa/Cairo")
	Scheduler = cron.New(cron.WithLocation(loc))

	Scheduler.AddFunc("5 0 * * *", UpdateAzanTimes)
	
	// جدولة الأدعية
	Scheduler.AddFunc("0 7 * * *", func() { BroadcastDuas(MorningDuas, "أذكار الصباح") })
	Scheduler.AddFunc("0 20 * * *", func() { BroadcastDuas(NightDuas, "أذكار المساء") })

	go UpdateAzanTimes()
	Scheduler.Start()
}

func UpdateAzanTimes() {
	resp, err := http.Get("http://api.aladhan.com/v1/timingsByCity?city=Cairo&country=Egypt&method=5")
	if err != nil { return }
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Timings map[string]string `json:"timings"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	for name, timeStr := range result.Data.Timings {
		if link, exists := PrayerLinks[name]; exists {
			cleanTime := strings.Split(timeStr, " ")[0]
			parts := strings.Split(cleanTime, ":")
			h, _ := strconv.Atoi(parts[0])
			m, _ := strconv.Atoi(parts[1])

			pName := name
			pLink := link

			Scheduler.AddFunc(fmt.Sprintf("%d %d * * *", m, h), func() {
				BroadcastAzan(pName, pLink)
			})
		}
	}
}

func BroadcastAzan(prayerKey, link string) {
	chats, _ := GetAllActiveChats()
	for _, chat := range chats {
		if enabled, ok := chat.Prayers[prayerKey]; ok && !enabled {
			continue
		}
		go StartAzanStream(chat.ChatID, prayerKey, link, false)
	}
}

func BroadcastDuas(duas []string, title string) {
	chats, _ := GetAllActiveChats() // هنا يجب التأكد من dua_active
	
	rand.Seed(time.Now().UnixNano())
	selectedDua := duas[rand.Intn(len(duas))]

	for _, chat := range chats {
		// في الحقيقة نحتاج دالة لجلب الجروبات المفعل فيها الدعاء، للتبسيط سنستخدم الفلتر هنا
		settings, _ := GetChatSettings(chat.ChatID)
		if !settings.DuaActive { continue }

		go func(cid int64) {
			BotClient.SendMessage(cid, &telegram.SendMessageOptions{
				Text: fmt.Sprintf("💫 **%s**\n\n%s\n\n<b>تـقـبـل الله مـنـا ومـنـكـم صـالـح الاعـمـال 🧚</b>", title, selectedDua),
			})
		}(chat.ChatID)
	}
}

func StartAzanStream(chatID int64, prayerKey, link string, forceTest bool) {
	cs, err := core.GetChatState(chatID)
	if err != nil { return }

	activeVC, _ := cs.IsActiveVC()
	if !activeVC {
		if forceTest {
			BotClient.SendMessage(chatID, &telegram.SendMessageOptions{Text: "⚠️ الـمـكـالـمـة الـصـوتـيـة مـغـلـقـة."})
		} else {
			BotClient.SendMessage(chatID, &telegram.SendMessageOptions{
				Text: fmt.Sprintf("🕌 **حـان الآن مـوعـد أذان %s**\n(الـمـكـالـمـة مـغـلـقـة، لـم يـتـم الـبـث 💫)", PrayerNamesStretched[prayerKey]),
			})
		}
		return
	}

	if present, _ := cs.IsAssistantPresent(); !present {
		cs.TryJoin()
		time.Sleep(2 * time.Second)
	}

	// النص المطلوب
	caption := fmt.Sprintf("🕌 **حـان الآن مـوعـد أذان %s**\n<b>بـالـتـوقـيـت الـمـحـلـي لـمـديـنـة الـقـاهـره 🧚</b>", PrayerNamesStretched[prayerKey])
	statusMsg, _ := BotClient.SendMessage(chatID, &telegram.SendMessageOptions{Text: caption})

	dummyMsg := &telegram.NewMessage{
		Client: BotClient,
		Message: &telegram.Message{
			Chat:   &telegram.Chat{ID: chatID},
			Text:   link,
			Sender: &telegram.Peer{ID: config.OwnerID[0]},
		},
	}

	tracks, err := platforms.GetTracks(dummyMsg, false)
	if err != nil || len(tracks) == 0 { return }

	track := tracks[0](track.Requester) = "خـدمـة الأذان"

	ctx := context.Background()
	path, err := platforms.Download(ctx, track, statusMsg)
	if err != nil {
		statusMsg.Edit("❌ فـشـل تـحـمـيـل الأذان.")
		return
	}

	r := core.GetRoom(chatID)
	r.Play(track, path, true) // Force Play
}
