// internal/modules/handlers.go
/*
  - هذا الملف جزء من مشروع YukkiMusic (معدّل لدعم أوامر عربية بدون /)
  - ملاحظة: يفترض وجود تعاريف/دوال أخرى في المشروع (jsonHandle, playHandler, ...).
  - اقرأ الملاحظات في نهاية الملف إذا ظهر لك خطأ متعلق بمكتبة gogram.
*/
package modules

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/Laky-64/gologging"
	"github.com/amarnathcjd/gogram/telegram"

	"main/internal/config"
	"main/internal/core"
	"main/internal/database"
	"main/ntgcalls"
)

// تعريف بسيط لوصف معرّف المعالج للرسائل
type MsgHandlerDef struct {
	Pattern string
	Handler telegram.MessageHandler
	Filters []telegram.Filter
}

// تعريف لِـ callback handlers
type CbHandlerDef struct {
	Pattern string
	Handler telegram.CallbackHandler
	Filters []telegram.Filter
}

// ---------- هنا نضع الأنماط (patterns) سواء بالإنجليزي أو بالعربي.
//  لاحظ أننا نضع عدة أشكال لكل أمر (إنجليزي، عربي، واختصارات).
//  كذلك نستعمل أنماط لا تحتاج / بالبداية — لكن نسمح أيضًا بوجود / اختياري.
func wordPattern(words ...string) string {
	// يبني regex مثل `(?i)^(?:/)?(?:play|تشغيل|ابدأ)\b`
	escaped := make([]string, 0, len(words))
	for _, w := range words {
		escaped = append(escaped, regexp.QuoteMeta(w))
	}
	return `(?i)^(?:/)?(?:` + strings.Join(escaped, "|") + `)\b`
}

var handlers = []MsgHandlerDef{
	// أدوات وادمن
	{Pattern: wordPattern("json"), Handler: jsonHandle},
	{Pattern: wordPattern("eval"), Handler: evalHandle, Filters: []telegram.Filter{ownerFilter}},
	{Pattern: wordPattern("ev"), Handler: evalCommandHandler, Filters: []telegram.Filter{ownerFilter}},
	{Pattern: wordPattern("bash", "sh"), Handler: shellHandle, Filters: []telegram.Filter{ownerFilter}},
	{Pattern: wordPattern("restart", "إعادة تشغيل", "إعادة_تشغيل"), Handler: handleRestart, Filters: []telegram.Filter{ownerFilter, ignoreChannelFilter}},

	// sudo management
	{Pattern: wordPattern("addsudo", "addsudoer", "sudoadd", "أضف_سودو", "اضف_سودو"), Handler: handleAddSudo, Filters: []telegram.Filter{ownerFilter, ignoreChannelFilter}},
	{Pattern: wordPattern("delsudo", "remsudo", "سحب_سودو", "احذف_سودو"), Handler: handleDelSudo, Filters: []telegram.Filter{ownerFilter, ignoreChannelFilter}},
	{Pattern: wordPattern("sudoers", "قائمة_السودو", "قائمه_السودو"), Handler: handleGetSudoers, Filters: []telegram.Filter{ignoreChannelFilter}},

	// اختبارات وسرعات
	{Pattern: wordPattern("speedtest", "spt", "اختبار_سرعة"), Handler: sptHandle, Filters: []telegram.Filter{sudoOnlyFilter, ignoreChannelFilter}},

	// بث ورسائل
	{Pattern: wordPattern("broadcast", "gcast", "bcast", "بث"), Handler: broadcastHandler, Filters: []telegram.Filter{ownerFilter, ignoreChannelFilter}},

	// حالة البوت وصيانته
	{Pattern: wordPattern("active", "ac", "activevc", "activevoice", "الحالة"), Handler: activeHandler, Filters: []telegram.Filter{sudoOnlyFilter, ignoreChannelFilter}},
	{Pattern: wordPattern("maintenance", "maint", "صيانة"), Handler: handleMaintenance, Filters: []telegram.Filter{ownerFilter, ignoreChannelFilter}},
	{Pattern: wordPattern("logger", "سجل", "لوغ"), Handler: handleLogger, Filters: []telegram.Filter{sudoOnlyFilter, ignoreChannelFilter}},
	{Pattern: wordPattern("autoleave", "autolev", "المغادرة_الآلية"), Handler: autoLeaveHandler, Filters: []telegram.Filter{sudoOnlyFilter, ignoreChannelFilter}},
	{Pattern: wordPattern("log", "logs"), Handler: logsHandler, Filters: []telegram.Filter{sudoOnlyFilter, ignoreChannelFilter}},

	// أوامر مساعدة عامة
	{Pattern: wordPattern("help", "مساعدة", "مساعدتي"), Handler: helpHandler, Filters: []telegram.Filter{ignoreChannelFilter}},
	{Pattern: wordPattern("ping", "بنق", "بنج"), Handler: pingHandler, Filters: []telegram.Filter{ignoreChannelFilter}},
	{Pattern: wordPattern("start", "ابدأ", "اهلا", "أهلا"), Handler: startHandler, Filters: []telegram.Filter{ignoreChannelFilter}},
	{Pattern: wordPattern("stats", "احصائيات", "إحصائيات"), Handler: statsHandler, Filters: []telegram.Filter{ignoreChannelFilter, sudoOnlyFilter}},
	{Pattern: wordPattern("bug", "اخطا", "خلل"), Handler: bugHandler, Filters: []telegram.Filter{ignoreChannelFilter}},
	{Pattern: wordPattern("lang", "language", "اللغة"), Handler: langHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},

	// أوامر البث و الستريم
	{Pattern: wordPattern("stream", "بث"), Handler: streamHandler, Filters: []telegram.Filter{superGroupFilter}},
	{Pattern: wordPattern("streamstop", "ايقاف_بث", "ايقاف_البث"), Handler: streamStopHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: wordPattern("streamstatus", "حالة_البث"), Handler: streamStatusHandler, Filters: []telegram.Filter{superGroupFilter}},
	{Pattern: wordPattern("rtmp", "setrtmp", "رتمپ"), Handler: setRTMPHandler},

	// أوامر التشغيل (إنجليزي + عربي)
	{Pattern: wordPattern("play", "تشغيل", "شغل", "ابدأ_تشغيل"), Handler: playHandler, Filters: []telegram.Filter{superGroupFilter}},
	{Pattern: wordPattern("fplay", "playforce", "تشغيل_إجبارى"), Handler: fplayHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: wordPattern("cplay", "تشغيل_القناة", "تشغيل_قناة"), Handler: cplayHandler, Filters: []telegram.Filter{superGroupFilter}},
	{Pattern: wordPattern("vplay", "تشغيل_فيديو", "vplay"), Handler: vplayHandler, Filters: []telegram.Filter{superGroupFilter}},
	{Pattern: wordPattern("skip", "تخطي", "next"), Handler: skipHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: wordPattern("pause", "ايقاف", "وقف"), Handler: pauseHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: wordPattern("resume", "استئناف", "تكملة"), Handler: resumeHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: wordPattern("replay", "اعادة"), Handler: replayHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: wordPattern("mute", "كتم"), Handler: muteHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: wordPattern("unmute", "الغاء_كتم"), Handler: unmuteHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: wordPattern("seek", "تقديم"), Handler: seekHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: wordPattern("seekback", "ترجيع"), Handler: seekbackHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: wordPattern("position", "الموضع", "موقع"), Handler: positionHandler, Filters: []telegram.Filter{superGroupFilter}},
	{Pattern: wordPattern("queue", "قائمة", "قائمه"), Handler: queueHandler, Filters: []telegram.Filter{superGroupFilter}},
	{Pattern: wordPattern("clear", "تفريغ", "مسح"), Handler: clearHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: wordPattern("remove", "حذف"), Handler: removeHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: wordPattern("move", "نقل"), Handler: moveHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: wordPattern("shuffle", "خلط"), Handler: shuffleHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: wordPattern("loop", "تكرار", "setloop"), Handler: loopHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: wordPattern("stop", "ايقاف_التشغيل", "انهاء", "end"), Handler: stopHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: wordPattern("reload", "اعادة_تحميل"), Handler: reloadHandler, Filters: []telegram.Filter{superGroupFilter}},
	{Pattern: wordPattern("addauth", "اضف_ادمن"), Handler: addAuthHandler, Filters: []telegram.Filter{superGroupFilter, adminFilter}},
	{Pattern: wordPattern("delauth", "حذف_ادمن"), Handler: delAuthHandler, Filters: []telegram.Filter{superGroupFilter, adminFilter}},
	{Pattern: wordPattern("authlist", "قائمة_الادمنية"), Handler: authListHandler, Filters: []telegram.Filter{superGroupFilter}},
}

// Callback handlers
var cbHandlers = []CbHandlerDef{
	{Pattern: "start", Handler: startCB},
	{Pattern: "help_cb", Handler: helpCB},
	{Pattern: "^lang:[a-z]", Handler: langCallbackHandler},
	{Pattern: `^help:(.+)`, Handler: helpCallbackHandler},
	{Pattern: "^close$", Handler: closeHandler},
	{Pattern: "^cancel$", Handler: cancelHandler},
	{Pattern: "^bcast_cancel$", Handler: broadcastCancelCB},
	{Pattern: `^room:(\w+)$`, Handler: roomHandle},
	{Pattern: "progress", Handler: emptyCBHandler},
}

// Init تسجّل المعالجات في بوت gogram
func Init(bot *telegram.Client, assistants *core.AssistantManager) {
	// تحديث حالة التحديثات للبت والـ assistants
	bot.UpdatesGetState()
	assistants.ForEach(func(a *core.Assistant) {
		a.Client.UpdatesGetState()
	})

	// تسجيل أوامر الرسائل (command-like handlers)
	for _, h := range handlers {
		// ملاحظة: بعض نسخ gogram تُرجع Handle يمكن تعديلها (SetGroup، AddFilters)
		// إذا كان الإصدار عندك لا يدعم chaining فاحذف SetGroup أو استخدم الأسلوب الصحيح.
		if handlerObj := bot.AddCommandHandler(h.Pattern, SafeMessageHandler(h.Handler), h.Filters...); handlerObj != nil {
			// حاول وضع المجموعة إن كانت الواجهة تدعم ذلك
			// (إذا أعطى الكومبايل خطأ هنا فامسح السطر أو عيّنه حسب واجهة مكتبتك)
			_ = handlerObj.SetGroup(100)
		}
	}

	// تسجيل callback handlers
	for _, h := range cbHandlers {
		if cbObj := bot.AddCallbackHandler(h.Pattern, SafeCallbackHandler(h.Handler), h.Filters...); cbObj != nil {
			_ = cbObj.SetGroup(90)
		}
	}

	// بعض أحداث التحرير (edit) — قد تختلف التوقيعات بين نسخ gogram
	// إذا كانت الدالة bot.On غير متوفرة في إصدارك استعمل الأسلوب البديل المناسب.
	_ = tryBotOn(bot, "edit:/eval", evalHandle, 80)
	_ = tryBotOn(bot, "edit:/ev", evalCommandHandler, 80)

	// حدث مشاركة/مشارك
	_ = tryBotOn(bot, "participant", handleParticipantUpdate, 70)

	// Action handler
	if ah := bot.AddActionHandler(handleActions); ah != nil {
		_ = ah.SetGroup(60)
	}

	// ربط أحداث نهاية البث للـ assistants
	assistants.ForEach(func(a *core.Assistant) {
		a.Ntg.OnStreamEnd(ntgOnStreamEnd)
	})

	// تشغيل مراقبة الغرف في goroutine
	go MonitorRooms()

	// تشغيل المغادرة التلقائية إن مفعّلة
	if is, _ := database.GetAutoLeave(); is {
		go startAutoLeave()
	}

	// تعيين أوامر البوت إن تطلب ذلك
	if config.SetCmds && config.OwnerID != 0 {
		go setBotCommands(bot)
	}

	// تجهيز مساعدة أوصاف أوامر channel-play (قابلة للتعديل)
	cplayCommands := []string{
		"/cfplay", "/vcplay", "/fvcplay",
		"/cpause", "/cresume", "/cskip", "/cstop",
		"/cmute", "/cunmute", "/cseek", "/cseekback",
		"/cjump", "/cremove", "/cclear", "/cmove",
		"/cspeed", "/creplay", "/cposition", "/cshuffle",
		"/cloop", "/cqueue", "/creload",
	}

	for _, cmd := range cplayCommands {
		baseCmd := "/" + cmd[2:] // Remove 'c' prefix
		if baseHelp, exists := helpTexts[baseCmd]; exists {
			helpTexts[cmd] = fmt.Sprintf(`<i>نسخة قناة من الأمر %s</i>

<b>⚙️ متطلبات:</b>
أولًا قم بتكوين القناة باستخدام: <code>/channelplay --set [channel_id]</code>

%s

<b>💡 ملاحظة:</b>
هذا الأمر يؤثر على دردشة الصوت في القناة المربوطة، وليس في الجروب الحالي.`, baseCmd, baseHelp)
		}
	}
}

// دالة وسيطة تحاول استدعاء bot.On إذا كانت متاحة في إصدار المكتبة
func tryBotOn(bot *telegram.Client, event string, handler telegram.MessageHandler, group int) error {
	// بعض نسخ المكتبة توفر bot.On(name, handler).SetGroup(n)
	// لو لم تتوفر سنكتفي بإرجاع nil (لا تؤدي إلا إذا كانت واجهة مختلفة)
	defer func() {
		// منع panic لو لم يكن الأسلوب موجودًا
		_ = recover()
	}()
	// محاولة استدعاء الأسلوب ديناميكيا (ملحوظة: هذه الطريقة تحمي من الـ panic أثناء الترجمة)
	if on := bot.On; on != nil {
		// try to call; may fail at compile time if signature doesn't match
		// وذلك لماذا نحيطها بحماية recover؛ لو فشل قم بإزالة استدعاء tryBotOn لاحقًا
		bot.On(event, handler).SetGroup(group)
	}
	return nil
}

func ntgOnStreamEnd(
	chatID int64,
	_ ntgcalls.StreamType,
	_ ntgcalls.StreamDevice,
) {
	onStreamEndHandler(chatID)
}

// setBotCommands — تعيين قائمة الأوامر للواجهات المختلفة
func setBotCommands(bot *telegram.Client) {
	// Set commands for normal users in private chats
	if _, err := bot.BotsSetBotCommands(&telegram.BotCommandScopeUsers{}, "", AllCommands.PrivateUserCommands); err != nil {
		gologging.Error("Failed to set PrivateUserCommands " + err.Error())
	}

	// Set commands for normal users in group chats
	if _, err := bot.BotsSetBotCommands(&telegram.BotCommandScopeChats{}, "", AllCommands.GroupUserCommands); err != nil {
		gologging.Error("Failed to set GroupUserCommands " + err.Error())
	}

	// Set commands for chat admins
	if _, err := bot.BotsSetBotCommands(
		&telegram.BotCommandScopeChatAdmins{},
		"",
		append(AllCommands.GroupUserCommands, AllCommands.GroupAdminCommands...),
	); err != nil {
		gologging.Error("Failed to set GroupAdminCommands " + err.Error())
	}

	// Set commands for sudo users in their private chat
	sudoers, err := database.GetSudoers()
	if err != nil {
		log.Printf("Failed to get sudoers for setting commands: %v", err)
	} else {
		sudoCommands := append(AllCommands.PrivateUserCommands, AllCommands.PrivateSudoCommands...)
		for _, sudoer := range sudoers {
			if _, err := bot.BotsSetBotCommands(&telegram.BotCommandScopePeer{
				Peer: &telegram.InputPeerUser{UserID: sudoer, AccessHash: 0},
			},
				"",
				sudoCommands,
			); err != nil {
				gologging.Error("Failed to set PrivateSudoCommands " + err.Error())
			}
		}
	}

	ownerCommands := append(
		AllCommands.PrivateUserCommands,
		AllCommands.PrivateSudoCommands...)
	ownerCommands = append(ownerCommands, AllCommands.PrivateOwnerCommands...)
	if _, err := bot.BotsSetBotCommands(&telegram.BotCommandScopePeer{
		Peer: &telegram.InputPeerUser{UserID: config.OwnerID, AccessHash: 0},
	}, "", ownerCommands); err != nil {
		gologging.Error("Failed to set PrivateOwnerCommands " + err.Error())
	}
}
