package app_defines

// 言語.
const (
	LanguageCodeJa   = "ja"
	LanguageCodeEn   = "en"
	LanguageCodeZhCN = "zh-cn"
	LanguageCodeZhTW = "zh-tw"
)

// LanguageCodes : 言語一覧.
func LanguageCodes() []string {
	return []string{
		LanguageCodeJa,
		LanguageCodeEn,
		LanguageCodeZhCN,
		LanguageCodeZhTW,
	}
}

// LanguageCodeName : 言語名.
func LanguageCodeName(lang string) string {
	switch lang {
	case LanguageCodeJa:
		return "日本語"
	case LanguageCodeEn:
		return "英語"
	case LanguageCodeZhCN:
		return "中国語(簡体字)"
	case LanguageCodeZhTW:
		return "中国語(繁体字)"
	}
	return ""
}

// 日付が変わる時間(JST).
const DateChangeTime = 0

// 場面.
const (
	SceneManagementPage = iota // 管理ページ.
	SceneMarketOrder
	SceneDistributeMarketFeeDividend
)

// SceneText: 場面テキスト.
func SceneText(scene int) string {
	switch scene {
	case SceneManagementPage:
		return "管理ページ"
	case SceneMarketOrder:
		return "マーケット取引"
	case SceneDistributeMarketFeeDividend:
		return "マーケット手数料デポジットの移動"
	}
	return ""
}
