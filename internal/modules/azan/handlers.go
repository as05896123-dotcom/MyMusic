package azan

import (
	"fmt"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"
	"main/internal/config"
)

// =======================
// Command Handler
// =======================
func CommandHandler(m *telegram.NewMessage) error {
	if m == nil || m.Sender == nil || m.Chat == nil {
		return nil
	}

	text := m.Text()
	chatID := m.Chat.ID
	senderID := m.Sender.ID

	// لوحة أوامر الأذان
	if text == "اعدادات الاذان" || text == "الاذان" || text == "اوامر الاذان" {
		kb := telegram.InlineKeyboardMarkup{
			Rows: []telegram.InlineKeyboardRow{
				{{Text: "أوامـر الـمـالـك", CallbackData: "cmd_owner"}},
				{{Text: "أوامـر الـمـشـرفـيـن", CallbackData: "cmd_admin"}},
				{{Text: "اغـلاق", CallbackData: "cmd_close"}},
			},
		}
		m.Reply(
			"<b>مـرحـبـاً بـك فـي قـائـمـة أوامـر الأذان</b>\n<b>اخـتـر الـقـائـمـة الـمـنـاسـبـة :</b>",
			&telegram.SendOptions{ReplyMarkup: kb},
		)
		return nil
	}

	// تفعيل الأذان
	if text == "تفعيل الاذان" {
		if !IsAdminOrOwner(m) {
			m.Reply("🧚 هذا الأمر للمشرفين فقط")
			return nil
		}
		settings, _ := GetChatSettings(chatID)
		if settings.AzanActive {
			m.Reply("💫 الاذان مفعل بالفعل.")
			return nil
		}
		UpdateChatSetting(chatID, "azan_active", true)
		m.Reply("⭐ تم تفعيل الاذان بنجاح.")
		return nil
	}

	// قفل الأذان
	if text == "قفل الاذان" {
		if !IsAdminOrOwner(m) {
			m.Reply("🧚 هذا الأمر للمشرفين فقط")
			return nil
		}
		settings, _ := GetChatSettings(chatID)
		if settings.ForcedActive && !IsOwner(senderID) {
			m.Reply("🧚 <b>هذا الأمر إجباري من المالك</b>")
			return nil
		}
		if !settings.AzanActive {
			m.Reply("💫 الاذان معطل بالفعل.")
			return nil
		}
		UpdateChatSetting(chatID, "azan_active", false)
		m.Reply("⭐ تم قفل الاذان بنجاح.")
		return nil
	}

	// تفعيل الأذكار
	if text == "تفعيل الدعاء" {
		if !IsAdminOrOwner(m) {
			return nil
		}
		UpdateChatSetting(chatID, "dua_active", true)
		m.Reply("🩵 تم تفعيل الأذكار بنجاح.")
		return nil
	}

	// اختبار الأذان
	if text == "تست الاذان" {
		if !IsAdminOrOwner(m) {
			return nil
		}
		m.Reply("⏳ <b>جاري تشغيل الأذان التجريبي...</b>")
		go StartAzanStream(chatID, "Fajr", PrayerLinks["Fajr"], true)
		return nil
	}

	return nil
}

// =======================
// Callback Handler
// =======================
func CallbackHandler(cb *telegram.CallbackQuery) error {
	if cb == nil || cb.Data == nil || cb.Msg == nil {
		return nil
	}

	data := string(cb.Data)
	chatID := cb.ChatID()
	userID := cb.Sender.ID

	// إغلاق اللوحة
	if data == "cmd_close" || data == "close_panel" {
		_, _ = cb.Client.DeleteMessages(chatID, []int{cb.Msg.ID}, false)
		return nil
	}

	// أوامر المالك
	if data == "cmd_owner" {
		if !IsOwner(userID) {
			_, _ = cb.Answer(&telegram.CallbackAnswer{
				Text:  "• هذا الزر للمالك فقط 🤍",
				Alert: true,
			})
			return nil
		}

		text := "<b>أوامر المالك :</b>\n\n• تفعيل الاذان الاجباري\n• فحص الاذان\n• تغيير رابط الاذان"
		kb := telegram.InlineKeyboardMarkup{
			Rows: []telegram.InlineKeyboardRow{
				{{Text: "رجـوع", CallbackData: "cmd_back_main"}},
			},
		}
		cb.Msg.Edit(text, &telegram.EditOptions{ReplyMarkup: kb})
		return nil
	}

	// أوامر المشرفين / رجوع
	if data == "cmd_admin" || data == "cmd_back_main" {
		if data == "cmd_back_main" {
			kb := telegram.InlineKeyboardMarkup{
				Rows: []telegram.InlineKeyboardRow{
					{{Text: "أوامـر الـمـالـك", CallbackData: "cmd_owner"}},
					{{Text: "أوامـر الـمـشـرفـيـن", CallbackData: "cmd_admin"}},
					{{Text: "اغـلاق", CallbackData: "cmd_close"}},
				},
			}
			cb.Msg.Edit("<b>قائمة أوامر الأذان</b>", &telegram.EditOptions{ReplyMarkup: kb})
			return nil
		}

		ShowSettingsPanel(cb.Msg, chatID)
		return nil
	}

	// تغيير الإعدادات
	if strings.HasPrefix(data, "set_") {
		if !IsOwner(userID) && !IsAdminByID(cb, chatID, userID) {
			return nil
		}

		parts := strings.Split(data, "_")
		settings, _ := GetChatSettings(chatID)

		switch parts[1] {
		case "main":
			UpdateChatSetting(chatID, "azan_active", !settings.AzanActive)
		case "dua":
			UpdateChatSetting(chatID, "dua_active", !settings.DuaActive)
		case "p":
			pkey := parts[2]
			UpdatePrayerSetting(chatID, pkey, !settings.Prayers[pkey])
		}

		ShowSettingsPanel(cb.Msg, chatID)
	}

	return nil
}

// =======================
// Settings Panel
// =======================
func ShowSettingsPanel(msg *telegram.Message, chatID int64) {
	settings, _ := GetChatSettings(chatID)

	stMain := "『 معطل 』"
	if settings.AzanActive {
		stMain = "『 مفعل 』"
	}
	stDua := "『 معطل 』"
	if settings.DuaActive {
		stDua = "『 مفعل 』"
	}

	rows := []telegram.InlineKeyboardRow{
		{{Text: "الاذان العام : " + stMain, CallbackData: "set_main"}},
		{{Text: "دعاء الصباح : " + stDua, CallbackData: "set_dua"}},
	}

	order := []string{"Fajr", "Dhuhr", "Asr", "Maghrib", "Isha"}
	for _, k := range order {
		status := "『 معطل 』"
		if settings.Prayers[k] {
			status = "『 مفعل 』"
		}
		rows = append(rows, telegram.InlineKeyboardRow{
			{Text: PrayerNamesStretched[k] + " : " + status, CallbackData: "set_p_" + k},
		})
	}

	rows = append(rows, telegram.InlineKeyboardRow{
		{Text: "اغلاق", CallbackData: "close_panel"},
	})

	kb := telegram.InlineKeyboardMarkup{Rows: rows}
	msg.Edit("<b>لوحة تحكم الأذان</b>", &telegram.EditOptions{ReplyMarkup: kb})
}

// =======================
// Permissions
// =======================
func IsOwner(userID int64) bool {
	return userID == config.OwnerID
}

func IsAdminOrOwner(m *telegram.NewMessage) bool {
	if IsOwner(m.Sender.ID) {
		return true
	}

	member, err := m.Client.GetChatMember(m.Chat.ID, m.Sender.ID)
	if err != nil {
		return false
	}

	return member.Status == telegram.ChatMemberStatusAdministrator ||
		member.Status == telegram.ChatMemberStatusCreator
}

func IsAdminByID(cb *telegram.CallbackQuery, chatID, userID int64) bool {
	member, err := cb.Client.GetChatMember(chatID, userID)
	if err != nil {
		return false
	}
	return member.Status == telegram.ChatMemberStatusAdministrator ||
		member.Status == telegram.ChatMemberStatusCreator
}
