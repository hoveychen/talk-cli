// Package voices provides hardcoded voice lists for all supported languages.
// Voice listing is offline-capable — no engine or model download required.
package voices

import "strings"

// Voice is a voice with its human-readable description.
type Voice struct {
	Name string
	Desc string
}

// DefaultEN is the default English voice.
const DefaultEN = "af_heart"

// DefaultZH is the default Chinese voice.
const DefaultZH = "zf_001"

// DefaultFor returns the default voice name for the given language.
func DefaultFor(lang string) string {
	switch lang {
	case "zh":
		return DefaultZH
	case "es":
		return "ef_dora"
	case "fr":
		return "ff_siwis"
	case "hi":
		return "hf_alpha"
	case "it":
		return "if_sara"
	case "ja":
		return "jf_alpha"
	case "pt":
		return "pf_dora"
	default:
		return DefaultEN
	}
}

// EN is the complete list of English voices (Kokoro v1.0, 54 voices).
var EN = buildList(enNames)

// ZH is the complete list of Chinese voices (Kokoro v1.1-zh, 103 voices).
var ZH = buildList(zhNames)

// langPrefixes maps a language tag to the voice-name prefixes belonging to it.
var langPrefixes = map[string][]string{
	"en": {"af_", "am_", "bf_", "bm_"},
	"es": {"ef_", "em_"},
	"fr": {"ff_"},
	"hi": {"hf_", "hm_"},
	"it": {"if_", "im_"},
	"ja": {"jf_", "jm_"},
	"pt": {"pf_", "pm_"},
	"zh": {"zf_", "zm_"},
}

// All returns the voice list for the requested lang:
//
//	"en"  → English voices
//	"zh"  → Chinese voices
//	"es"  → Spanish voices
//	"fr"  → French voices
//	"hi"  → Hindi voices
//	"it"  → Italian voices
//	"ja"  → Japanese voices
//	"pt"  → Portuguese (BR) voices
//	"all" → all voices combined
func All(lang string) []Voice {
	if lang == "all" {
		out := make([]Voice, 0, len(EN)+len(ZH))
		out = append(out, EN...)
		out = append(out, ZH...)
		return out
	}
	prefixes, ok := langPrefixes[lang]
	if !ok {
		return nil
	}
	var pool []Voice
	if lang == "zh" {
		pool = ZH
	} else {
		pool = EN
	}
	var out []Voice
	for _, v := range pool {
		for _, p := range prefixes {
			if strings.HasPrefix(v.Name, p) {
				out = append(out, v)
				break
			}
		}
	}
	return out
}

// Describe returns a short human-readable description based on the voice prefix.
func Describe(name string) string {
	switch {
	case strings.HasPrefix(name, "af_"):
		return "American English, female"
	case strings.HasPrefix(name, "am_"):
		return "American English, male"
	case strings.HasPrefix(name, "bf_"):
		return "British English, female"
	case strings.HasPrefix(name, "bm_"):
		return "British English, male"
	case strings.HasPrefix(name, "ef_"):
		return "Spanish, female"
	case strings.HasPrefix(name, "em_"):
		return "Spanish, male"
	case strings.HasPrefix(name, "ff_"):
		return "French, female"
	case strings.HasPrefix(name, "hf_"):
		return "Hindi, female"
	case strings.HasPrefix(name, "hm_"):
		return "Hindi, male"
	case strings.HasPrefix(name, "if_"):
		return "Italian, female"
	case strings.HasPrefix(name, "im_"):
		return "Italian, male"
	case strings.HasPrefix(name, "jf_"):
		return "Japanese, female"
	case strings.HasPrefix(name, "jm_"):
		return "Japanese, male"
	case strings.HasPrefix(name, "pf_"):
		return "Portuguese (BR), female"
	case strings.HasPrefix(name, "pm_"):
		return "Portuguese (BR), male"
	case strings.HasPrefix(name, "zf_"):
		return "Mandarin Chinese, female"
	case strings.HasPrefix(name, "zm_"):
		return "Mandarin Chinese, male"
	default:
		return ""
	}
}

// ── internal ──────────────────────────────────────────────────────────────────

func buildList(names []string) []Voice {
	out := make([]Voice, len(names))
	for i, n := range names {
		out[i] = Voice{Name: n, Desc: Describe(n)}
	}
	return out
}

var enNames = []string{
	"af_alloy", "af_aoede", "af_bella", "af_heart", "af_jessica",
	"af_kore", "af_nicole", "af_nova", "af_river", "af_sarah", "af_sky",
	"am_adam", "am_echo", "am_eric", "am_fenrir", "am_liam",
	"am_michael", "am_onyx", "am_puck", "am_santa",
	"bf_alice", "bf_emma", "bf_isabella", "bf_lily",
	"bm_daniel", "bm_fable", "bm_george", "bm_lewis",
	"ef_dora", "em_alex", "em_santa",
	"ff_siwis",
	"hf_alpha", "hf_beta", "hm_omega", "hm_psi",
	"if_sara", "im_nicola",
	"jf_alpha", "jf_gongitsune", "jf_nezumi", "jf_tebukuro", "jm_kumo",
	"pf_dora", "pm_alex", "pm_santa",
	"zf_xiaobei", "zf_xiaoni", "zf_xiaoxiao", "zf_xiaoyi",
	"zm_yunjian", "zm_yunxi", "zm_yunxia", "zm_yunyang",
}

var zhNames = []string{
	// Mixed-language voices bundled with zh variant
	"af_maple", "af_sol", "bf_vale",
	// Chinese female voices
	"zf_001", "zf_002", "zf_003", "zf_004", "zf_005",
	"zf_006", "zf_007", "zf_008", "zf_017", "zf_018",
	"zf_019", "zf_021", "zf_022", "zf_023", "zf_024",
	"zf_026", "zf_027", "zf_028", "zf_032", "zf_036",
	"zf_038", "zf_039", "zf_040", "zf_042", "zf_043",
	"zf_044", "zf_046", "zf_047", "zf_048", "zf_049",
	"zf_051", "zf_059", "zf_060", "zf_067", "zf_070",
	"zf_071", "zf_072", "zf_073", "zf_074", "zf_075",
	"zf_076", "zf_077", "zf_078", "zf_079", "zf_083",
	"zf_084", "zf_085", "zf_086", "zf_087", "zf_088",
	"zf_090", "zf_092", "zf_093", "zf_094", "zf_099",
	// Chinese male voices
	"zm_009", "zm_010", "zm_011", "zm_012", "zm_013",
	"zm_014", "zm_015", "zm_016", "zm_020", "zm_025",
	"zm_029", "zm_030", "zm_031", "zm_033", "zm_034",
	"zm_035", "zm_037", "zm_041", "zm_045", "zm_050",
	"zm_052", "zm_053", "zm_054", "zm_055", "zm_056",
	"zm_057", "zm_058", "zm_061", "zm_062", "zm_063",
	"zm_064", "zm_065", "zm_066", "zm_068", "zm_069",
	"zm_080", "zm_081", "zm_082", "zm_089", "zm_091",
	"zm_095", "zm_096", "zm_097", "zm_098", "zm_100",
}
