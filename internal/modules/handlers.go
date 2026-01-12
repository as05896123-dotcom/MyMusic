/*
This file is part of YukkiMusic.

YukkiMusic — A Telegram bot that streams music into group voice chats
with seamless playback and control.

Copyright (C) 2025 TheTeamVivek

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/

package modules

import (
	"fmt"
	"log"

	"github.com/Laky-64/gologging"
	"github.com/amarnathcjd/gogram/telegram"

	"main/internal/config"
	"main/internal/core"
	"main/internal/database"
	"main/ntgcalls"
)

// ملاحظة: الأنماط هنا هي RegExp. استخدمت ^/? لبداية الاختيارية للـ slash
// و (?i) لجعل المطابقة غير حساسة لحالة الحروف.
// لإضافة مرادفات عربية جديدة، ضفها داخل القوسين '(...|مرادف|...)' قبل '\b'.

type MsgHandlerDef struct {
	Pattern string
	Handler telegram.MessageHandler
	Filters []telegram.Filter
}

type CbHandlerDef struct {
	Pattern string
	Handler telegram.CallbackHandler
	Filters []telegram.Filter
}

var handlers = []MsgHandlerDef{
	// أساسي/عام
	{Pattern: `(?i)^/?(json|جيسون)\b`, Handler: jsonHandle},
	{Pattern: `(?i)^/?(eval|قيم)\b`, Handler: evalHandle, Filters: []telegram.Filter{ownerFilter}},
	{Pattern: `(?i)^/?(ev|كود)\b`, Handler: evalCommandHandler, Filters: []telegram.Filter{ownerFilter}},
	{Pattern: `(?i)^/?(bash|sh|تيرمينال|باش)\b`, Handler: shellHandle, Filters: []telegram.Filter{ownerFilter}},
	{Pattern: `(?i)^/?(restart|ريستارت|انعاش|إعادة تشغيل)\b`, Handler: handleRestart, Filters: []telegram.Filter{ownerFilter, ignoreChannelFilter}},

	// sudo management
	{Pattern: `(?i)^/?(addsudo|addsudoer|sudoadd|رفع مطور)\b`, Handler: handleAddSudo, Filters: []telegram.Filter{ownerFilter, ignoreChannelFilter}},
	{Pattern: `(?i)^/?(delsudo|delsudoer|sudodel|remsudo|rmsudo|sudorem|dropsudo|unsudo|تنزيل مطور|حذف مطور)\b`, Handler: handleDelSudo, Filters: []telegram.Filter{ownerFilter, ignoreChannelFilter}},
	{Pattern: `(?i)^/?(sudoers|listsudo|sudolist|المطورين|قائمة المطورين)\b`, Handler: handleGetSudoers, Filters: []telegram.Filter{ignoreChannelFilter}},

	// أدوات
	{Pattern: `(?i)^/?(speedtest|spt|سرعة|سرعة السيرفر)\b`, Handler: sptHandle, Filters: []telegram.Filter{sudoOnlyFilter, ignoreChannelFilter}},
	{Pattern: `(?i)^/?(broadcast|gcast|bcast|إذاعة|اذاعه|نشر)\b`, Handler: broadcastHandler, Filters: []telegram.Filter{ownerFilter, ignoreChannelFilter}},
	{Pattern: `(?i)^/?(ac|active|activevc|activevoice|الكولات|المكالمات)\b`, Handler: activeHandler, Filters: []telegram.Filter{sudoOnlyFilter, ignoreChannelFilter}},
	{Pattern: `(?i)^/?(maintenance|maint|صيانة|وضع الصيانة)\b`, Handler: handleMaintenance, Filters: []telegram.Filter{ownerFilter, ignoreChannelFilter}},
	{Pattern: `(?i)^/?(logger|لوجر|سجل)\b`, Handler: handleLogger, Filters: []telegram.Filter{sudoOnlyFilter, ignoreChannelFilter}},
	{Pattern: `(?i)^/?(autoleave|مغادرة تلقائية)\b`, Handler: autoLeaveHandler, Filters: []telegram.Filter{sudoOnlyFilter, ignoreChannelFilter}},
	{Pattern: `(?i)^/?(log|logs|اللوج|السجلات)\b`, Handler: logsHandler, Filters: []telegram.Filter{sudoOnlyFilter, ignoreChannelFilter}},

	// مساعدة وتشغيل أساسي
	{Pattern: `(?i)^/?(help|مساعدة|اوامر|الأوامر)\b`, Handler: helpHandler, Filters: []telegram.Filter{ignoreChannelFilter}},
	{Pattern: `(?i)^/?(ping|بنج|تست)\b`, Handler: pingHandler, Filters: []telegram.Filter{ignoreChannelFilter}},
	{Pattern: `(?i)^/?(start|ابدأ|ستارت)\b`, Handler: startHandler, Filters: []telegram.Filter{ignoreChannelFilter}},
	{Pattern: `(?i)^/?(stats|احصائيات|الإحصائيات)\b`, Handler: statsHandler, Filters: []telegram.Filter{ignoreChannelFilter, sudoOnlyFilter}},
	{Pattern: `(?i)^/?(bug|بلاغ|مشكلة)\b`, Handler: bugHandler, Filters: []telegram.Filter{ignoreChannelFilter}},
	{Pattern: `(?i)^/?(lang|language|لغة|اللغة)\b`, Handler: langHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},

	// SuperGroup & Admin Filters
	{Pattern: `(?i)^/?(stream|بث)\b`, Handler: streamHandler, Filters: []telegram.Filter{superGroupFilter}},
	{Pattern: `(?i)^/?(streamstop|وقف البث)\b`, Handler: streamStopHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(streamstatus|حالة البث)\b`, Handler: streamStatusHandler, Filters: []telegram.Filter{superGroupFilter}},
	{Pattern: `(?i)^/?(rtmp|setrtmp)\b`, Handler: setRTMPHandler},

	// play / فلوآت تشغيل
	{Pattern: `(?i)^/?(play|شغل|تشغيل|هات|سمعنا)\b`, Handler: playHandler, Filters: []telegram.Filter{superGroupFilter}},
	{Pattern: `(?i)^/?(fplay|playforce|شغل بقوة)\b`, Handler: fplayHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(cplay|شغل في قناة|شغل بالقناة)\b`, Handler: cplayHandler, Filters: []telegram.Filter{superGroupFilter}},
	{Pattern: `(?i)^/?(cfplay|fcplay|cplayforce)\b`, Handler: cfplayHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(vplay|فيديو|شغل فيديو)\b`, Handler: vplayHandler, Filters: []telegram.Filter{superGroupFilter}},
	{Pattern: `(?i)^/?(fvplay|vfplay|vplayforce)\b`, Handler: fvplayHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(vcplay|cvplay)\b`, Handler: vcplayHandler, Filters: []telegram.Filter{superGroupFilter}},
	{Pattern: `(?i)^/?(fvcplay|fvcpay|vcplayforce)\b`, Handler: fvcplayHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},

	// تحكم في التشغيل
	{Pattern: `(?i)^/?(speed|setspeed|speedup|سرعة|السرعة)\b`, Handler: speedHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(skip|next|عدي|تخطي|اللي بعده|سكيب)\b`, Handler: skipHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(pause|مؤقت|اوقفي|استني|هدي)\b`, Handler: pauseHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(resume|كمل|استئناف|واصل)\b`, Handler: resumeHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(replay|عيد|تكرار الاغنية)\b`, Handler: replayHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(mute|اخرس|كتم|اسكت)\b`, Handler: muteHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(unmute|تكلم|الغي الكتم|فك الكتم)\b`, Handler: unmuteHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(seek|قدم|قدم لل)\b`, Handler: seekHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(seekback|رجع|اللي قبل)\b`, Handler: seekbackHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(jump|نط)\b`, Handler: jumpHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(position|مكان|وصلنا فين)\b`, Handler: positionHandler, Filters: []telegram.Filter{superGroupFilter}},

	// قائمة وتشغيل متقدمة
	{Pattern: `(?i)^/?(queue|طابور|القايمة|قايمة|الدور)\b`, Handler: queueHandler, Filters: []telegram.Filter{superGroupFilter}},
	{Pattern: `(?i)^/?(clear|نظف|مسح|تنظيف)\b`, Handler: clearHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(remove|احذف|مسح اغنية)\b`, Handler: removeHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(move|حرك|نقل)\b`, Handler: moveHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(shuffle|لخبط|عشوائي)\b`, Handler: shuffleHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(loop|setloop|تكرار|تكرار القائمة)\b`, Handler: loopHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(end|stop|بس|اقف|كفاية|إيقاف|انهاء|اخرج)\b`, Handler: stopHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(reload|تحديث)\b`, Handler: reloadHandler, Filters: []telegram.Filter{superGroupFilter}},
	{Pattern: `(?i)^/?(addauth|رفع مساعد)\b`, Handler: addAuthHandler, Filters: []telegram.Filter{superGroupFilter, adminFilter}},
	{Pattern: `(?i)^/?(delauth|تنزيل مساعد)\b`, Handler: delAuthHandler, Filters: []telegram.Filter{superGroupFilter, adminFilter}},
	{Pattern: `(?i)^/?(authlist|قائمة المساعدين)\b`, Handler: authListHandler, Filters: []telegram.Filter{superGroupFilter}},

	// CPlay commands (قناة)
	{Pattern: `(?i)^/?(cplay|cvplay|شغل قناة)\b`, Handler: cplayHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(cfplay|fcplay|cforceplay)\b`, Handler: cfplayHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(cpause|cpause)\b`, Handler: cpauseHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(cresume)\b`, Handler: cresumeHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(cmute)\b`, Handler: cmuteHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(cunmute)\b`, Handler: cunmuteHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(cstop|cend)\b`, Handler: cstopHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(cqueue)\b`, Handler: cqueueHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(cskip)\b`, Handler: cskipHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(cloop|csetloop)\b`, Handler: cloopHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(cseek)\b`, Handler: cseekHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(cseekback)\b`, Handler: cseekbackHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(cjump)\b`, Handler: cjumpHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(cremove)\b`, Handler: cremoveHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(cclear)\b`, Handler: cclearHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(cmove)\b`, Handler: cmoveHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(channelplay)\b`, Handler: channelPlayHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(cspeed|csetspeed|cspeedup)\b`, Handler: cspeedHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(creplay)\b`, Handler: creplayHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(cposition)\b`, Handler: cpositionHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(cshuffle)\b`, Handler: cshuffleHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
	{Pattern: `(?i)^/?(creload)\b`, Handler: creloadHandler, Filters: []telegram.Filter{superGroupFilter, authFilter}},
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

func Init(bot *telegram.Client, assistants *core.AssistantManager) {
	// ننفّذ UpdatesGetState لنعرف حالة الويب هوك / التحديثات
	bot.UpdatesGetState()
	assistants.ForEach(func(a *core.Assistant) {
		a.Client.UpdatesGetState()
	})

	// استخدام bot.On مع event pattern "message:<regexp>"
	// ثم إضافة الفلاتر باستخدام AddFilters لتجنب "too many arguments"
	for _, h := range handlers {
		eventPattern := "message:" + h.Pattern

		// استدعاء بنمطين فقط (pattern و handler)
		handlerObj := bot.On(eventPattern, SafeMessageHandler(h.Handler))

		// لو فيه فلاتر، نضيفها بعدين
		if len(h.Filters) > 0 {
			handlerObj.AddFilters(h.Filters...)
		}

		handlerObj.SetGroup(100)
	}

	for _, h := range cbHandlers {
		bot.AddCallbackHandler(h.Pattern, SafeCallbackHandler(h.Handler), h.Filters...).
			SetGroup(90)
	}

	bot.On("edit:/eval", evalHandle).SetGroup(80)
	bot.On("edit:/ev", evalCommandHandler).SetGroup(80)

	bot.On("participant", handleParticipantUpdate).SetGroup(70)

	bot.AddActionHandler(handleActions).SetGroup(60)

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

	// تعليمات مساعدة خاصة بأوامر cplay
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
			helpTexts[cmd] = fmt.Sprintf(`<i>Channel play variant of %s</i>

<b>⚙️ Requires:</b>
First configure channel using: <code>/channelplay --set [channel_id]</code>

%s

<b>💡 Note:</b>
This command affects the linked channel's voice chat, not the current group.`, baseCmd, baseHelp)
		}
	}
}

func ntgOnStreamEnd(
	chatID int64,
	_ ntgcalls.StreamType,
	_ ntgcalls.StreamDevice,
) {
	onStreamEndHandler(chatID)
}

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
