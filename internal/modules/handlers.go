// internal/modules/handlers.go
/*
  - هذا الملف جزء من مشروع YukkiMusic (معدّل لدعم أوامر عربية بدون /)
  - تم إصلاح مشكلة توافق توقيعات المعالجات (handlers) عبر استخدام reflection عند استدعاء bot.On
  - ملاحظة: يفترض وجود تعاريف/دوال أخرى في المشروع (jsonHandle, playHandler, ...).
*/
package modules

import (
	"fmt"
	"log"
	"reflect"
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

// wordPattern: يبني regex يقبل الكلمات بالانجليزي او العربي، مع او بدون /
func wordPattern(words ...string) string {
	escaped := make([]string, 0, len(words))
	for _, w := range words {
		escaped = append(escaped, regexp.QuoteMeta(w))
	}
	// (?i) -> case-insensitive, (?:/)? -> يسمح بوجود "/" أو لا
	return `(?i)^(?:/)?(?:` + strings.Join(escaped, "|") + `)\b`
}

var handlers = []MsgHandlerDef{
	{Pattern: wordPattern("json"), Handler: jsonHandle},
	{Pattern: wordPattern("eval"), Handler: evalHandle, Filters: []telegram.Filter{ownerFilter}},
	{Pattern: wordPattern("ev"), Handler: evalCommandHandler, Filters: []telegram.Filter{ownerFilter}},
	{Pattern: wordPattern("bash", "sh"), Handler: shellHandle, Filters: []telegram.Filter{ownerFilter}},
	{Pattern: wordPattern("restart", "إعادة تشغيل", "إعادة_تشغيل"), Handler: handleRestart, Filters: []telegram.Filter{ownerFilter, ignoreChannelFilter}},

	{Pattern: wordPattern("addsudo", "addsudoer", "sudoadd", "أضف_سودو", "اضف_سودو"), Handler: handleAddSudo, Filters: []telegram.Filter{ownerFilter, ignoreChannelFilter}},
	{Pattern: wordPattern("delsudo", "remsudo", "سحب_سودو", "احذف_سودو"), Handler: handleDelSudo, Filters: []telegram.Filter{ownerFilter, ignoreChannelFilter}},
	{Pattern: wordPattern("sudoers", "قائمة_السودو", "قائمه_السودو"), Handler: handleGetSudoers, Filters: []telegram.Filter{ignoreChannelFilter}},

	{Pattern: wordPattern("speedtest", "spt", "اختبار_سرعة"), Handler: sptHandle, Filters: []telegram.Filter{sudoOnlyFilter, ignoreChannelFilter}},

	{Pattern: wordPattern("broadcast", "gcast", "bcast", "بث"), Handler: broadcastHandler, Filters: []telegram.Filter{ownerFilter, ignoreChannelFilter}},

	{Pattern: wordPattern("active", "ac", "activevc", "activevoice", "الحالة"), Handler: activeHandler, Filters: []telegram.Filter{sudoOnlyFilter, ignoreChannelFilter}},
	{Pattern: wordPattern("maintenance", "maint", "صيانة"), Handler: handleMaintenance, Filters: []telegram.Filter{ownerFilter, ignoreChannelFilter}},
	{Pattern: wordPattern("logger", "سجل", "لوغ"), Handler: handleLogger, Filters: []telegram.Filter{sudoOnlyFilter, ignoreChannelFilter}},
	{Pattern: wordPattern("autoleave", "autolev", "المغادرة_الآلية"), Handler: autoLeaveHandler, Filters: []telegram.Filter{sudoOnlyFilter, ignoreChannelFilter}},
	{Pattern: wordPattern("log", "logs"), Handler: logsHandler, Filters: []telegram.Filter{sudoOnlyFilter, ignoreChannelFilter}},

	{Pattern: wordPattern("help", "مساعدة", "مساعدتي"), Handler: helpHandler, Filters: []telegram.Filter{ignoreChannelFilter}},
	{Pattern: wordPattern("ping", "بنق", "بنج"), Handler: pingHandler, Filters: []telegram.Filter{ignoreChannelFilter}},
	{Pattern: wordPattern("start", "ابدأ", "اهلا", "أهلا"), Handler: startHandler, Filters: []telegram.Filter{ignoreChannelFilter}},
	{Pattern: wordPattern("stats", "احصائيات", "إحصائيات"), Handler: statsHandler, Filters: []telegram.Filter{ignoreChannelFilter, sudoOnlyFilter}},
	{Pattern: wordPattern("bug", "اخطا", "خلل"), Handler: bugHandler, Filters: []telegram.Filter{ignoreChannelFilter}},
	{Pattern: wordPattern("lang", "language", "اللغة"), Handler: langHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},

	{Pattern: wordPattern("stream", "بث"), Handler: streamHandler, Filters: []telegram.Filter{superGroupFilter}},
	{Pattern: wordPattern("streamstop", "ايقاف_بث", "ايقاف_البث"), Handler: streamStopHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: wordPattern("streamstatus", "حالة_البث"), Handler: streamStatusHandler, Filters: []telegram.Filter{superGroupFilter}},
	{Pattern: wordPattern("rtmp", "setrtmp", "رتمپ"), Handler: setRTMPHandler},

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

// Init: تسجل المعالجات في البوت
func Init(bot *telegram.Client, assistants *core.AssistantManager) {
	bot.UpdatesGetState()
	assistants.ForEach(func(a *core.Assistant) {
		a.Client.UpdatesGetState()
	})

	// تسجيل handlers كـ command-like (regex patterns)
	for _, h := range handlers {
		// حاول تسجيل وأضف SetGroup إن أمكن
		// بعض إصدارات gogram تُعيد قيمة قابلة للاستعمال، وبعضها لا.
		// لذا نتعامل بحذر: إذا أعاد AddCommandHandler شيئًا نستدعي SetGroup بالـ reflection-friendly approach.
		if handlerObj := bot.AddCommandHandler(h.Pattern, SafeMessageHandler(h.Handler), h.Filters...); handlerObj != nil {
			// إذا كان لديه SetGroup فنعيّنها، وإلا نتجاهل
			_ = trySetGroup(handlerObj, 100)
		}
	}

	// تسجيل callback handlers
	for _, h := range cbHandlers {
		if cbObj := bot.AddCallbackHandler(h.Pattern, SafeCallbackHandler(h.Handler), h.Filters...); cbObj != nil {
			_ = trySetGroup(cbObj, 90)
		}
	}

	// الآن نستدعي bot.On للأحداث المتغايرة التواقيع (edit, participant, ...)
	_ = tryBotOn(bot, "edit:/eval", evalHandle, 80)
	_ = tryBotOn(bot, "edit:/ev", evalCommandHandler, 80)

	// 'participant' يستخدم توقيعًا مختلفًا عند البعض -> استعمل tryBotOn بالديناميكية
	_ = tryBotOn(bot, "participant", handleParticipantUpdate, 70)

	// Action handler
	if ah := bot.AddActionHandler(handleActions); ah != nil {
		_ = trySetGroup(ah, 60)
	}

	assistants.ForEach(func(a *core.Assistant) {
		a.Ntg.OnStreamEnd(ntgOnStreamEnd)
	})

	go MonitorRooms()

	if is, _ := database.GetAutoLeave(); is {
		go startAutoLeave()
	}

	if config.SetCmds && config.OwnerID != 0 {
		go setBotCommands(bot)
	}

	// تجهيز أوصاف أوامر القناة
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

// trySetGroup: يحاول استدعاء SetGroup عبر reflection إن وجدت
func trySetGroup(obj interface{}, group int) error {
	defer func() { _ = recover() }()
	v := reflect.ValueOf(obj)
	if !v.IsValid() {
		return nil
	}
	setGroup := v.MethodByName("SetGroup")
	if !setGroup.IsValid() {
		return nil
	}
	// استدعاء SetGroup(int)
	setGroup.Call([]reflect.Value{reflect.ValueOf(group)})
	return nil
}

// tryBotOn: يستعمل reflection لاستدعاء bot.On(event, handler).SetGroup(group)
// handler يمكن أن يكون أي توقيع (message handler, participant handler, ...).
// هذا يجنب أخطاء التوافق بين توقيعات الدوال عند التجميع.
func tryBotOn(bot *telegram.Client, event string, handler interface{}, group int) error {
	defer func() { _ = recover() }()

	bv := reflect.ValueOf(bot)
	if !bv.IsValid() {
		return nil
	}
	on := bv.MethodByName("On")
	if !on.IsValid() {
		// واجهة bot لا تحتوي On — نسكت
		return nil
	}

	// استدعاء bot.On(event, handler)
	res := on.Call([]reflect.Value{reflect.ValueOf(event), reflect.ValueOf(handler)})
	if len(res) == 0 {
		return nil
	}
	// قد يرجع Handle واحد؛ حاول استدعاء SetGroup عليه
	trySetGroup(res[0].Interface(), group)
	return nil
}

func ntgOnStreamEnd(
	chatID int64,
	_ ntgcalls.StreamType,
	_ ntgcalls.StreamDevice,
) {
	onStreamEndHandler(chatID)
}

// setBotCommands — مساعدة لتعيين الأوامر
func setBotCommands(bot *telegram.Client) {
	if _, err := bot.BotsSetBotCommands(&telegram.BotCommandScopeUsers{}, "", AllCommands.PrivateUserCommands); err != nil {
		gologging.Error("Failed to set PrivateUserCommands " + err.Error())
	}

	if _, err := bot.BotsSetBotCommands(&telegram.BotCommandScopeChats{}, "", AllCommands.GroupUserCommands); err != nil {
		gologging.Error("Failed to set GroupUserCommands " + err.Error())
	}

	if _, err := bot.BotsSetBotCommands(
		&telegram.BotCommandScopeChatAdmins{},
		"",
		append(AllCommands.GroupUserCommands, AllCommands.GroupAdminCommands...),
	); err != nil {
		gologging.Error("Failed to set GroupAdminCommands " + err.Error())
	}

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
