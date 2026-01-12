/*
  - This file is part of YukkiMusic.
  - Arabized & Modified for Slashless Commands
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

// ---------------------------------------------------------
//  تعديل الأنماط (Regex) لدعم العربي والإنجليزي بدون سلاش
// ---------------------------------------------------------
var handlers = []MsgHandlerDef{
	{Pattern: "(?i)^/?(json|جيسون)$", Handler: jsonHandle},
	{
		Pattern: "(?i)^/?(eval|قيم)$",
		Handler: evalHandle,
		Filters: []telegram.Filter{ownerFilter},
	},
	{
		Pattern: "(?i)^/?(ev|كود)$",
		Handler: evalCommandHandler,
		Filters: []telegram.Filter{ownerFilter},
	},
	{
		Pattern: "(?i)^/?(bash|sh|تيرمينال)$",
		Handler: shellHandle,
		Filters: []telegram.Filter{ownerFilter},
	},
	{
		Pattern: "(?i)^/?(restart|ريستارت|انعاش|إعادة تشغيل)$",
		Handler: handleRestart,
		Filters: []telegram.Filter{ownerFilter, ignoreChannelFilter},
	},

	{
		Pattern: "(?i)^/?(addsudo|addsudoer|sudoadd|رفع مطور)$",
		Handler: handleAddSudo,
		Filters: []telegram.Filter{ownerFilter, ignoreChannelFilter},
	},
	{
		Pattern: "(?i)^/?(delsudo|delsudoer|sudodel|remsudo|rmsudo|sudorem|dropsudo|unsudo|تنزيل مطور|حذف مطور)$",
		Handler: handleDelSudo,
		Filters: []telegram.Filter{ownerFilter, ignoreChannelFilter},
	},
	{
		Pattern: "(?i)^/?(sudoers|listsudo|sudolist|المطورين|قائمة المطورين)$",
		Handler: handleGetSudoers,
		Filters: []telegram.Filter{ignoreChannelFilter},
	},

	{
		Pattern: "(?i)^/?(speedtest|spt|سرعة|سرعة السيرفر)$",
		Handler: sptHandle,
		Filters: []telegram.Filter{sudoOnlyFilter, ignoreChannelFilter},
	},

	{
		Pattern: "(?i)^/?(broadcast|gcast|bcast|إذاعة|اذاعه|نشر)$",
		Handler: broadcastHandler,
		Filters: []telegram.Filter{ownerFilter, ignoreChannelFilter},
	},

	{
		Pattern: "(?i)^/?(ac|active|activevc|activevoice|الكولات|المكالمات)$",
		Handler: activeHandler,
		Filters: []telegram.Filter{sudoOnlyFilter, ignoreChannelFilter},
	},
	{
		Pattern: "(?i)^/?(maintenance|maint|صيانة|وضع الصيانة)$",
		Handler: handleMaintenance,
		Filters: []telegram.Filter{ownerFilter, ignoreChannelFilter},
	},
	{
		Pattern: "(?i)^/?(logger|لوجر|سجل)$",
		Handler: handleLogger,
		Filters: []telegram.Filter{sudoOnlyFilter, ignoreChannelFilter},
	},
	{
		Pattern: "(?i)^/?(autoleave|مغادرة تلقائية)$",
		Handler: autoLeaveHandler,
		Filters: []telegram.Filter{sudoOnlyFilter, ignoreChannelFilter},
	},
	{
		Pattern: "(?i)^/?(log|logs|اللوج|السجلات)$",
		Handler: logsHandler,
		Filters: []telegram.Filter{sudoOnlyFilter, ignoreChannelFilter},
	},

	{
		Pattern: "(?i)^/?(help|مساعدة|اوامر|الأوامر)$",
		Handler: helpHandler,
		Filters: []telegram.Filter{ignoreChannelFilter},
	},
	{
		Pattern: "(?i)^/?(ping|بنج|تست)$",
		Handler: pingHandler,
		Filters: []telegram.Filter{ignoreChannelFilter},
	},
	{
		Pattern: "(?i)^/?(start|ابدأ|ستارت)$",
		Handler: startHandler,
		Filters: []telegram.Filter{ignoreChannelFilter},
	},
	{
		Pattern: "(?i)^/?(stats|احصائيات|الإحصائيات)$",
		Handler: statsHandler,
		Filters: []telegram.Filter{ignoreChannelFilter, sudoOnlyFilter},
	},
	{
		Pattern: "(?i)^/?(bug|بلاغ|مشكلة)$",
		Handler: bugHandler,
		Filters: []telegram.Filter{ignoreChannelFilter},
	},
	{
		Pattern: "(?i)^/?(lang|language|لغة|اللغة)$",
		Handler: langHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},

	// SuperGroup & Admin Filters

	{
		Pattern: "(?i)^/?(stream|بث)$",
		Handler: streamHandler,
		Filters: []telegram.Filter{superGroupFilter},
	},
	{
		Pattern: "(?i)^/?(streamstop|وقف البث)$",
		Handler: streamStopHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(streamstatus|حالة البث)$",
		Handler: streamStatusHandler,
		Filters: []telegram.Filter{superGroupFilter},
	},
	{Pattern: "(?i)^/?(rtmp|setrtmp)$", Handler: setRTMPHandler},

	// play/cplay/vplay/fplay commands
	{
		Pattern: "(?i)^/?(play|شغل|تشغيل|هات|سمعنا)$",
		Handler: playHandler,
		Filters: []telegram.Filter{superGroupFilter},
	},
	{
		Pattern: "(?i)^/?(fplay|playforce|شغل بقوة)$",
		Handler: fplayHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(cplay|شغل في قناة)$",
		Handler: cplayHandler,
		Filters: []telegram.Filter{superGroupFilter},
	},
	{
		Pattern: "(?i)^/?(cfplay|fcplay|cplayforce)$",
		Handler: cfplayHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(vplay|فيديو|شغل فيديو)$",
		Handler: vplayHandler,
		Filters: []telegram.Filter{superGroupFilter},
	},
	{
		Pattern: "(?i)^/?(fvplay|vfplay|vplayforce)$",
		Handler: fvplayHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(vcplay|cvplay)$",
		Handler: vcplayHandler,
		Filters: []telegram.Filter{superGroupFilter},
	},
	{
		Pattern: "(?i)^/?(fvcplay|fvcpay|vcplayforce)$",
		Handler: fvcplayHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},

	{
		Pattern: "(?i)^/?(speed|setspeed|speedup|سرعة|السرعة)$",
		Handler: speedHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(skip|next|عدي|تخطي|اللي بعده|سكيب)$",
		Handler: skipHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(pause|مؤقت|اوقفي|استني|هدي)$",
		Handler: pauseHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(resume|كمل|استئناف|واصل)$",
		Handler: resumeHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(replay|عيد|تكرار الاغنية)$",
		Handler: replayHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(mute|اخرس|كتم|اسكت)$",
		Handler: muteHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(unmute|تكلم|الغي الكتم|فك الكتم)$",
		Handler: unmuteHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(seek|قدم)$",
		Handler: seekHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(seekback|رجع)$",
		Handler: seekbackHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(jump|نط)$",
		Handler: jumpHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(position|مكان|وصلنا فين)$",
		Handler: positionHandler,
		Filters: []telegram.Filter{superGroupFilter},
	},
	{
		Pattern: "(?i)^/?(queue|طابور|القايمة|قايمة|الدور)$",
		Handler: queueHandler,
		Filters: []telegram.Filter{superGroupFilter},
	},
	{
		Pattern: "(?i)^/?(clear|نظف|مسح|تنظيف)$",
		Handler: clearHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(remove|احذف|مسح اغنية)$",
		Handler: removeHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(move|حرك|نقل)$",
		Handler: moveHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(shuffle|لخبط|عشوائي)$",
		Handler: shuffleHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(loop|setloop|تكرار|تكرار القائمة)$",
		Handler: loopHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(end|stop|بس|اقف|كفاية|إيقاف|انهاء|اخرج)$",
		Handler: stopHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(reload|تحديث)$",
		Handler: reloadHandler,
		Filters: []telegram.Filter{superGroupFilter},
	},
	{
		Pattern: "(?i)^/?(addauth|رفع مساعد)$",
		Handler: addAuthHandler,
		Filters: []telegram.Filter{superGroupFilter, adminFilter},
	},
	{
		Pattern: "(?i)^/?(delauth|تنزيل مساعد)$",
		Handler: delAuthHandler,
		Filters: []telegram.Filter{superGroupFilter, adminFilter},
	},
	{
		Pattern: "(?i)^/?(authlist|قائمة المساعدين)$",
		Handler: authListHandler,
		Filters: []telegram.Filter{superGroupFilter},
	},

	// CPlay commands (أوامر تشغيل القنوات)
	{
		Pattern: "(?i)^/?(cplay|cvplay)$",
		Handler: cplayHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(cfplay|fcplay|cforceplay)$",
		Handler: cfplayHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(cpause)$",
		Handler: cpauseHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(cresume)$",
		Handler: cresumeHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(cmute)$",
		Handler: cmuteHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(cunmute)$",
		Handler: cunmuteHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(cstop|cend)$",
		Handler: cstopHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(cqueue)$",
		Handler: cqueueHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(cskip)$",
		Handler: cskipHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(cloop|csetloop)$",
		Handler: cloopHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(cseek)$",
		Handler: cseekHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(cseekback)$",
		Handler: cseekbackHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(cjump)$",
		Handler: cjumpHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(cremove)$",
		Handler: cremoveHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(cclear)$",
		Handler: cclearHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(cmove)$",
		Handler: cmoveHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(channelplay)$",
		Handler: channelPlayHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(cspeed|csetspeed|cspeedup)$",
		Handler: cspeedHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(creplay)$",
		Handler: creplayHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(cposition)$",
		Handler: cpositionHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(cshuffle)$",
		Handler: cshuffleHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
	{
		Pattern: "(?i)^/?(creload)$",
		Handler: creloadHandler,
		Filters: []telegram.Filter{superGroupFilter, authFilter},
	},
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
	bot.UpdatesGetState()
	assistants.ForEach(func(a *core.Assistant) {
		a.Client.UpdatesGetState()
	})

	// -----------------------------------------------------
	//  FIXED: حل جذري لمشكلة too many arguments
	//  1. بنستدعي bot.On بالنمط والدالة فقط.
	//  2. بنضيف الفلاتر (إن وجدت) بعدين باستخدام .AddFilters
	// -----------------------------------------------------
	for _, h := range handlers {
		eventPattern := "message:" + h.Pattern

		// استدعاء نظيف بـ 2 معامل فقط عشان نتجنب الخطأ
		handlerObj := bot.On(eventPattern, SafeMessageHandler(h.Handler))

		// لو فيه فلاتر، نضيفها للكائن اللي رجع
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
