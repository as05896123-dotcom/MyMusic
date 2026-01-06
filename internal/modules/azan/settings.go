// modules/azan/settings.go
package azan

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"main/internal/database"
)

// هيكل إعدادات الأذان لكل مجموعة (الدردشة)
type ChatAzanSettings struct {
	ChatID         int64           `bson:"chat_id"`
	AzanActive     bool            `bson:"azan_active"`
	ForcedActive   bool            `bson:"forced_active"`
	DuaActive      bool            `bson:"dua_active"`
	NightDuaActive bool            `bson:"night_dua_active"`
	Prayers        map[string]bool `bson:"prayers"`
}

// الحصول على إعدادات الأذان لمحادثة معينة (أو إنشاء وثيقة جديدة بالتخصيص الافتراضي)
func GetChatSettings(chatID int64) (*ChatAzanSettings, error) {
	var settings ChatAzanSettings
	collection := database.MongoDB.Collection("azan_settings")

	filter := bson.M{"chat_id": chatID}
	err := collection.FindOne(context.TODO(), filter).Decode(&settings)
	if err != nil {
		// إعداد افتراضي إذا لم توجد وثيقة مسبقًا
		newDoc := ChatAzanSettings{
			ChatID:         chatID,
			AzanActive:     true,
			DuaActive:      true,
			NightDuaActive: true,
			Prayers:        map[string]bool{"Fajr": true, "Dhuhr": true, "Asr": true, "Maghrib": true, "Isha": true},
		}
		collection.InsertOne(context.TODO(), newDoc)
		return &newDoc, nil
	}
	// تهيئة خريطة الصلوات إذا كانت nil
	if settings.Prayers == nil {
		settings.Prayers = map[string]bool{"Fajr": true, "Dhuhr": true, "Asr": true, "Maghrib": true, "Isha": true}
	}
	return &settings, nil
}

// تحديث إعداد عام (مثلاً تفعيل/تعطيل الأذان أو الأذكار)
func UpdateChatSetting(chatID int64, key string, value interface{}) {
	collection := database.MongoDB.Collection("azan_settings")
	opts := options.Update().SetUpsert(true)
	update := bson.M{"$set": bson.M{key: value}}
	collection.UpdateOne(context.TODO(), bson.M{"chat_id": chatID}, update, opts)
}

// تحديث حالة صلاة معينة (مثلاً تعطيل أذان صلاة الظهر للمجموعة)
func UpdatePrayerSetting(chatID int64, prayerKey string, value bool) {
	collection := database.MongoDB.Collection("azan_settings")
	opts := options.Update().SetUpsert(true)
	update := bson.M{"$set": bson.M{fmt.Sprintf("prayers.%s", prayerKey): value}}
	collection.UpdateOne(context.TODO(), bson.M{"chat_id": chatID}, update, opts)
}

// جلب كل المجموعات التي تم تفعيل الأذان فيها ليتم بث الأذان لها تلقائياً
func GetAllActiveChats() ([]ChatAzanSettings, error) {
	var results []ChatAzanSettings
	cursor, err := database.MongoDB.Collection("azan_settings").Find(context.TODO(), bson.M{"azan_active": true})
	if err != nil {
		return nil, err
	}
	cursor.All(context.TODO(), &results)
	return results, nil
}
