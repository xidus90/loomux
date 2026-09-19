package commit

import "sync"

var ordinaryEN = map[string]struct{}{
	"a": {}, "all": {}, "an": {}, "and": {}, "any": {}, "are": {}, "as": {}, "at": {},
	"back": {}, "be": {}, "been": {}, "both": {}, "by": {}, "can": {}, "car": {}, "care": {},
	"com": {}, "come": {}, "contra": {}, "del": {}, "den": {}, "did": {}, "die": {}, "do": {},
	"does": {}, "dos": {}, "dove": {}, "down": {}, "each": {}, "est": {}, "even": {}, "fest": {},
	"fins": {}, "first": {}, "for": {}, "from": {}, "get": {}, "had": {}, "has": {}, "hat": {},
	"have": {}, "her": {}, "here": {}, "him": {}, "his": {}, "how": {}, "if": {}, "in": {},
	"into": {}, "is": {}, "it": {}, "its": {}, "just": {}, "last": {}, "le": {}, "less": {},
	"lo": {}, "made": {}, "make": {}, "man": {}, "many": {}, "may": {}, "more": {}, "most": {},
	"much": {}, "new": {}, "next": {}, "no": {}, "not": {}, "now": {}, "of": {}, "off": {},
	"old": {}, "on": {}, "one": {}, "only": {}, "or": {}, "other": {}, "our": {}, "out": {},
	"over": {}, "pendant": {}, "per": {}, "plus": {}, "prima": {}, "put": {}, "run": {}, "same": {},
	"sans": {}, "sense": {}, "set": {}, "so": {}, "some": {}, "son": {}, "still": {}, "such": {},
	"tan": {}, "than": {}, "that": {}, "the": {}, "then": {}, "there": {}, "these": {}, "this": {},
	"those": {}, "to": {}, "tout": {}, "two": {}, "um": {}, "under": {}, "up": {}, "us": {},
	"use": {}, "used": {}, "very": {}, "war": {}, "was": {}, "way": {}, "we": {}, "were": {},
	"what": {}, "when": {}, "where": {}, "which": {}, "who": {}, "why": {}, "will": {}, "with": {},
	"would": {}, "you": {},
}

var ordinaryDE = map[string]struct{}{
	"alle": {}, "als": {}, "also": {}, "am": {}, "an": {}, "auch": {}, "auf": {}, "aus": {},
	"bei": {}, "bis": {}, "da": {}, "dann": {}, "das": {}, "dass": {}, "dem": {}, "den": {},
	"der": {}, "des": {}, "die": {}, "doch": {}, "du": {}, "durch": {}, "ein": {}, "eine": {},
	"einen": {}, "er": {}, "es": {}, "fuer": {}, "gegen": {}, "gut": {}, "haben": {}, "hat": {},
	"hier": {}, "ich": {}, "ihr": {}, "im": {}, "in": {}, "ist": {}, "ja": {}, "jetzt": {},
	"kann": {}, "mal": {}, "man": {}, "mehr": {}, "mit": {}, "muss": {}, "nach": {}, "neu": {},
	"nicht": {}, "noch": {}, "nun": {}, "nur": {}, "oder": {}, "ohne": {}, "per": {}, "plus": {},
	"prima": {}, "schon": {}, "sehr": {}, "sein": {}, "sie": {}, "sind": {}, "so": {}, "soll": {},
	"ueber": {}, "um": {}, "und": {}, "unter": {}, "von": {}, "vor": {}, "war": {}, "waren": {},
	"was": {}, "weil": {}, "wenn": {}, "werden": {}, "wie": {}, "wir": {}, "wird": {}, "wurde": {},
	"zu": {},
}

var germanSource = map[string]struct{}{
	"auf": {}, "aus": {}, "beim": {}, "das": {}, "dass": {}, "dem": {}, "der": {}, "des": {},
	"durch": {}, "ein": {}, "eine": {}, "einem": {}, "einen": {}, "einer": {}, "eines": {}, "fuer": {},
	"gegen": {}, "haben": {}, "heraus": {}, "ihn": {}, "jede": {}, "jeden": {}, "jeder": {}, "mit": {},
	"nach": {}, "nicht": {}, "noch": {}, "oder": {}, "ohne": {}, "samt": {}, "schon": {}, "sind": {},
	"statt": {}, "ueber": {}, "und": {}, "unter": {}, "von": {}, "waren": {}, "weil": {}, "wenn": {},
	"werden": {}, "wieder": {}, "wird": {}, "wurde": {}, "zu": {}, "zum": {}, "zur": {}, "zusammen": {},
}

var englishSource = map[string]struct{}{
	"about": {}, "again": {}, "against": {}, "already": {}, "and": {}, "because": {}, "between": {}, "could": {},
	"each": {}, "every": {}, "from": {}, "instead": {}, "into": {}, "rather": {}, "should": {}, "that": {},
	"the": {}, "their": {}, "there": {}, "this": {}, "through": {}, "which": {}, "while": {}, "with": {},
	"without": {}, "would": {},
}

var romanceSource = map[string]struct{}{
	"aceasta": {}, "acest": {}, "aceste": {}, "acestea": {}, "acum": {}, "adesso": {}, "afin": {}, "agli": {},
	"ahora": {}, "ainda": {}, "ainsi": {}, "al": {}, "alcuni": {}, "alguna": {}, "algunos": {}, "alla": {},
	"alle": {}, "allo": {}, "alors": {}, "als": {}, "amb": {}, "anche": {}, "ancora": {}, "antes": {},
	"ao": {}, "aos": {}, "aquela": {}, "aquele": {}, "aquest": {}, "aquesta": {}, "aquestes": {}, "aquests": {},
	"aqui": {}, "aquilo": {}, "ara": {}, "as": {}, "asi": {}, "assim": {}, "asta": {}, "atunci": {},
	"au": {}, "aucun": {}, "aucune": {}, "aunque": {}, "aussi": {}, "autre": {}, "autres": {}, "aux": {},
	"avant": {}, "avec": {}, "beaucoup": {}, "bien": {}, "cada": {}, "car": {}, "care": {}, "ce": {},
	"ceci": {}, "cela": {}, "celle": {}, "celui": {}, "ces": {}, "cet": {}, "cette": {}, "ceux": {},
	"chaque": {}, "che": {}, "chez": {}, "com": {}, "come": {}, "como": {}, "con": {}, "contra": {},
	"cu": {}, "cual": {}, "cuales": {}, "cualquier": {}, "cuando": {}, "da": {}, "dai": {}, "dal": {},
	"dalla": {}, "dans": {}, "dar": {}, "das": {}, "de": {}, "debe": {}, "deci": {}, "degli": {},
	"dei": {}, "del": {}, "della": {}, "delle": {}, "dello": {}, "dels": {}, "depois": {}, "depuis": {},
	"des": {}, "desde": {}, "din": {}, "do": {}, "doit": {}, "donc": {}, "doncs": {}, "donde": {},
	"dont": {}, "dopo": {}, "dos": {}, "dove": {}, "du": {}, "durante": {}, "el": {}, "ela": {},
	"elas": {}, "ele": {}, "eles": {}, "elle": {}, "elles": {}, "ells": {}, "els": {}, "encara": {},
	"encore": {}, "entonces": {}, "entre": {}, "esa": {}, "ese": {}, "eso": {}, "essa": {}, "esse": {},
	"essere": {}, "essi": {}, "est": {}, "esta": {}, "estar": {}, "estas": {}, "este": {}, "esto": {},
	"estos": {}, "fiecare": {}, "fins": {}, "foarte": {}, "foi": {}, "fra": {}, "gli": {}, "hacia": {},
	"hasta": {}, "il": {}, "ils": {}, "in": {}, "isso": {}, "isto": {}, "ja": {}, "je": {},
	"la": {}, "las": {}, "le": {}, "lei": {}, "les": {}, "leur": {}, "leurs": {}, "llavors": {},
	"lo": {}, "loro": {}, "lorsque": {}, "los": {}, "lui": {}, "ma": {}, "mas": {}, "mentre": {},
	"mereu": {}, "mes": {}, "meu": {}, "mi": {}, "mientras": {}, "minha": {}, "mis": {}, "misma": {},
	"mismo": {}, "moins": {}, "molt": {}, "molti": {}, "molto": {}, "molts": {}, "mon": {}, "muito": {},
	"mult": {}, "muy": {}, "na": {}, "nada": {}, "nadie": {}, "nas": {}, "negli": {}, "nei": {},
	"nel": {}, "nella": {}, "nelle": {}, "no": {}, "noi": {}, "nos": {}, "nosaltres": {}, "notre": {},
	"nous": {}, "nu": {}, "numa": {}, "nunca": {}, "ogni": {}, "on": {}, "orice": {}, "otra": {},
	"otras": {}, "otro": {}, "otros": {}, "par": {}, "para": {}, "parce": {}, "pas": {}, "pe": {},
	"pela": {}, "pelas": {}, "pelo": {}, "pelos": {}, "pels": {}, "pendant": {}, "pentru": {}, "pero": {},
	"peu": {}, "peut": {}, "plus": {}, "plusieurs": {}, "poco": {}, "pode": {}, "pois": {}, "por": {},
	"porque": {}, "pour": {}, "pourquoi": {}, "prima": {}, "prin": {}, "puede": {}, "puisque": {}, "quais": {},
	"qual": {}, "quand": {}, "que": {}, "quel": {}, "quella": {}, "quelle": {}, "quelles": {}, "quelli": {},
	"quello": {}, "quelque": {}, "quels": {}, "quem": {}, "questa": {}, "queste": {}, "questi": {}, "questo": {},
	"qui": {}, "quien": {}, "quienes": {}, "quindi": {}, "quoi": {}, "sa": {}, "sans": {}, "sau": {},
	"se": {}, "selon": {}, "sem": {}, "sense": {}, "senza": {}, "ser": {}, "ses": {}, "seu": {},
	"seus": {}, "seva": {}, "si": {}, "siamo": {}, "siempre": {}, "sin": {}, "sobre": {}, "soit": {},
	"son": {}, "sono": {}, "sont": {}, "sous": {}, "stesso": {}, "su": {}, "sua": {}, "suas": {},
	"sul": {}, "sulla": {}, "sunt": {}, "sur": {}, "sus": {}, "tan": {}, "tanta": {}, "tanto": {},
	"tem": {}, "tiene": {}, "toate": {}, "toda": {}, "todas": {}, "todo": {}, "todos": {}, "toujours": {},
	"tous": {}, "tout": {}, "toute": {}, "toutes": {}, "tra": {}, "tu": {}, "tutta": {}, "tutte": {},
	"tutti": {}, "um": {}, "uma": {}, "umas": {}, "una": {}, "unas": {}, "unde": {}, "une": {},
	"uno": {}, "unos": {}, "uns": {}, "voi": {}, "vosaltres": {}, "votre": {}, "vous": {},
}

var goSource = map[string]struct{}{
	"aendere": {}, "aktualisiere": {}, "anpassung": {}, "anpassungen": {}, "auf": {}, "aus": {}, "baustein": {}, "bausteine": {},
	"behebe": {}, "beim": {}, "beispiel": {}, "beispiele": {}, "bereinige": {}, "bereinigung": {}, "das": {}, "dass": {},
	"datei": {}, "dateien": {}, "dem": {}, "der": {}, "des": {}, "deutsch": {}, "deutsche": {}, "dokumentation": {},
	"durch": {}, "ein": {}, "eine": {}, "einem": {}, "einen": {}, "einer": {}, "eines": {}, "entferne": {},
	"erstelle": {}, "erweitere": {}, "fehler": {}, "fuege": {}, "fuer": {}, "funktion": {}, "funktionen": {}, "gegen": {},
	"haben": {}, "heraus": {}, "hinzu": {}, "ihn": {}, "jede": {}, "jeden": {}, "jeder": {}, "komponente": {},
	"komponenten": {}, "korrigiere": {}, "mit": {}, "nach": {}, "nachricht": {}, "nachrichten": {}, "nicht": {}, "noch": {},
	"oder": {}, "ohne": {}, "pruefung": {}, "samt": {}, "schnittstelle": {}, "schnittstellen": {}, "schon": {}, "sind": {},
	"statt": {}, "ueber": {}, "ueberarbeite": {}, "und": {}, "unter": {}, "verbessere": {}, "von": {}, "waren": {},
	"weil": {}, "wenn": {}, "werden": {}, "wieder": {}, "wird": {}, "wurde": {}, "zu": {}, "zum": {},
	"zur": {}, "zusammen": {},
}

var stopwordsEN = sync.OnceValue(func() map[string]struct{} {
	m := make(map[string]struct{})
	for w := range germanSource {
		if _, ok := ordinaryEN[w]; !ok {
			m[w] = struct{}{}
		}
	}
	for w := range romanceSource {
		if _, ok := ordinaryEN[w]; !ok {
			m[w] = struct{}{}
		}
	}
	for w := range goSource {
		if _, ok := ordinaryEN[w]; !ok {
			m[w] = struct{}{}
		}
	}
	return m
})

var stopwordsDE = sync.OnceValue(func() map[string]struct{} {
	m := make(map[string]struct{})
	for w := range englishSource {
		if _, ok := ordinaryDE[w]; !ok {
			m[w] = struct{}{}
		}
	}
	for w := range romanceSource {
		if _, ok := ordinaryDE[w]; !ok {
			m[w] = struct{}{}
		}
	}
	return m
})

// Stopwords returns the filtered stopword set for the target language.
// The union is assembled on first call, not at package init.
func Stopwords(lang string) map[string]struct{} {
	switch lang {
	case "en":
		return stopwordsEN()
	case "de":
		return stopwordsDE()
	default:
		return nil
	}
}
