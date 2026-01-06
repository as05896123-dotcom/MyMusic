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
	"log"

	"github.com/amarnathcjd/gogram/telegram"
	"github.com/robfig/cron/v3"

	"main/internal/config"
	"main/internal/core"
	"main/internal/platforms"
)

// العصب الرئيسي للنظام
var (
	Scheduler *cron.Cron
	BotClient *telegram.Client
)

// InitAzanScheduler : تهيئة النظام
func InitAzanScheduler(client *telegram.Client) {
	BotClient = client
	
	loc, err := time.LoadLocation("Africa/Cairo")
	if err != nil {
		log.Println("جاري استخدام التوقيت المحلي لعدم العثور على توقيت القاهرة.")
		loc = time.Local
	}
	
	Scheduler = cron.New(cron.WithLocation(loc))

	// تحديث يومي ذكي
	Scheduler.AddFunc("5 0 * * *", UpdateAzanTimes)
	
	// الأذكار
	Scheduler.AddFunc("0 7 * * *", func() { BroadcastDuas(MorningDuas, "أذكار الصباح") })
	Scheduler.AddFunc("0 20 * * *", func() { BroadcastDuas(NightDuas, "أذكار المساء") })

	go UpdateAzanTimes()
	Scheduler.Start()
}

// UpdateAzanTimes : جلب المواقيت بذكاء
func UpdateAzanTimes() {
	var resp *http.Response
	var err error

	// محاولة الاتصال 3 مرات بهدوء
	for i := 0; i < 3; i++ {
		client := http.Client{Timeout: 10 * time.Second}
		resp, err = client.Get("http://api.aladhan.com/v1/timingsByCity?city=Cairo&country=Egypt&method=5")
		if err == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		return // فشل صامت، سيعتمد على الجدولة السابقة إن وجدت
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Timings map[string]string `json:"timings"`
		} `json:"data"`
	}
	
	if decodeErr := json.NewDecoder(resp.Body).Decode(&result); decodeErr != nil {
		return
	}

	for name, timeStr := range result.Data.Timings {
		if link, exists := PrayerLinks[name]; exists {
			cleanTime := strings.Split(timeStr, " ")[0]
			parts := strings.Split(cleanTime, ":")
			
			if len(parts) < 2 { continue }

			h, _ := strconv.Atoi(parts[0])
			m, _ := strconv.Atoi(parts[1])

			pName, pLink := name, link

			Scheduler.AddFunc(fmt.Sprintf("%d %d * * *", m, h), func() {
				BroadcastAzan(pName, pLink)
			})
		}
	}
}

func BroadcastAzan(prayerKey, link string) {
	chats, err := GetAllActiveChats()
	if err != nil { return }

	for _, chat := range chats {
		if enabled, ok := chat.Prayers[prayerKey]; ok && !enabled {
			continue
		}
		go StartAzanStream(chat.ChatID, prayerKey, link, false)
	}
}

func BroadcastDuas(duas []string, title string) {
	chats, _ := GetAllActiveChats()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	selectedDua := duas[r.Intn(len(duas))]

	for _, chat := range chats {
		settings, _ := GetChatSettings(chat.ChatID)
		if !settings.DuaActive { continue }

		go func(cid int64) {
			BotClient.SendMessage(cid, &telegram.SendMessageOptions{
				Text: fmt.Sprintf("💫 **%s**\n\n%s\n\n<b>تـقـبـل الله مـنـا ومـنـكـم صـالـح الاعـمـال 🧚</b>", title, selectedDua),
			})
		}(chat.ChatID)
	}
}

// 🧠 StartAzanStream : العصب الذكي (Smart Core)
func StartAzanStream(chatID int64, prayerKey, link string, forceTest bool) {
	cs, err := core.GetChatState(chatID)
	if err != nil { return }

	// 1️⃣ إصلاح ذاتي للمكالمة (Auto-Heal VC)
	activeVC, _ := cs.IsActiveVC()
	if !activeVC {
		assistant := core.Assistants.Get(chatID)
		if assistant != nil {
			err := assistant.PhoneCreateGroupCall(chatID, "")
			if err == nil {
				time.Sleep(3 * time.Second)
			}
		} else {
			if forceTest { BotClient.SendMessage(chatID, &telegram.SendMessageOptions{Text: "لا يوجد مساعد."}) }
			return
		}
	}

	// 2️⃣ إصلاح ذاتي للمساعد (Auto-Join)
	if present, _ := cs.IsAssistantPresent(); !present {
		cs.TryJoin()
		time.Sleep(2 * time.Second)
		// فحص تأكيدي
		if p, _ := cs.IsAssistantPresent(); !p {
			cs.TryJoin()
			time.Sleep(1 * time.Second)
		}
	}

	// 3️⃣ التنبيهات الجمالية
	if stickerID, ok := PrayerStickers[prayerKey]; ok {
		BotClient.SendSticker(chatID, &telegram.SendStickerOptions{
			Sticker: &telegram.InputFileID{ID: stickerID},
		})
	}

	caption := fmt.Sprintf("🕌 **حـان الآن مـوعـد أذان %s**\n<b>بـالـتـوقـيـت الـمـحـلـي لـمـديـنـة الـقـاهـره 🧚</b>", PrayerNamesStretched[prayerKey])
	statusMsg, _ := BotClient.SendMessage(chatID, &telegram.SendMessageOptions{Text: caption})

	// 4️⃣ معالجة الصوت
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
		BotClient.DeleteMessages(chatID, []int{statusMsg.ID}) // تنظيف أثر الرسالة
		return
	}

	// ✅✅✅ التصحيح السليم للكود ✅✅✅
	track := tracks[0](track.Requester) = "خـدمـة الأذان"

	ctx := context.Background()
	path, err := platforms.Download(ctx, track, statusMsg)
	if err != nil {
		statusMsg.Edit("فـشـل تـحـمـيـل الأذان.")
		return
	}

	// 5️⃣ التشغيل
	r := core.GetRoom(chatID)
	if r != nil {
		r.Play(track, path, true) 
	}

	// 🔥🔥🔥 المصيدة الذكية (The Sniper) 🔥🔥🔥
	// هذه الوظيفة تعمل كقناص لانتظار ظهور الكيبورد وحذفه في أجزاء من الثانية
	go func() {
		// إنشاء عداد زمني للتفتيش كل 200 مللي ثانية (سريع جداً)
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()

		// توقيت انتهاء العملية (بعد 4 ثواني نستسلم عشان الموارد)
		timeout := time.After(4 * time.Second)

		for {
			select {
			case <-timeout:
				return // انتهى الوقت
			case <-ticker.C:
				// فحص آخر رسالة
				history, err := BotClient.GetHistory(chatID, 0, 0, 0, 3, 0, 0, 0)
				if err == nil && history != nil {
					for _, m := range history.Messages {
						// الشرط: الرسالة من البوت + تحتوي على كيبورد + ليست رسالة الأذان النصية
						if m.Sender.ID == BotClient.Self.ID && m.ReplyMarkup != nil {
							// 🛑 حبس الكيبورد وحذفه فوراً
							BotClient.DeleteMessages(chatID, []int{m.ID})
							return // المهمة انتهت بنجاح، نخرج
						}
					}
				}
			}
		}
	}()
}
