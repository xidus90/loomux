package commit

import (
	"fmt"
	"regexp"
	"strings"
)

// germanStopWords holds German function words and common development verbs/nouns
// that do not overlap with ordinary English prose.
var germanStopWords = map[string]bool{
	"der": true, "das": true, "dem": true, "des": true, "und": true, "oder": true,
	"nicht": true, "ein": true, "eine": true, "einen": true, "einem": true, "eines": true,
	"einer": true, "sind": true, "waren": true, "haben": true, "wird": true, "wurde": true,
	"werden": true, "mit": true, "von": true, "fuer": true, "für": true, "ueber": true,
	"über": true, "aus": true, "nach": true, "ohne": true, "beim": true, "zum": true,
	"zur": true, "zu": true, "auf": true, "durch": true, "gegen": true, "dass": true,
	"weil": true, "wenn": true, "schon": true, "noch": true, "jeden": true, "jede": true,
	"jeder": true, "wieder": true, "statt": true, "samt": true, "unter": true, "zusammen": true,
	"heraus": true, "ihn": true, "deutsch": true, "deutsche": true, "nachricht": true,
	"nachrichten": true, "fuege": true, "füge": true, "hinzu": true, "aktualisiere": true,
	"korrigiere": true, "entferne": true, "behebe": true, "aendere": true, "ändere": true,
	"erweitere": true, "ueberarbeite": true, "überarbeite": true, "verbessere": true,
	"erstelle": true, "dateien": true, "datei": true, "fehler": true, "beispiele": true,
	"beispiel": true, "dokumentation": true, "funktionen": true, "funktion": true,
	"komponenten": true, "komponente": true, "bausteine": true, "baustein": true,
	"schnittstelle": true, "schnittstellen": true, "bereinige": true, "bereinigung": true,
	"anpassung": true, "anpassungen": true, "pruefung": true, "prüfung": true,
}

var umlautRegex = regexp.MustCompile(`[äöüÄÖÜß]`)

// ValidateCommitMessage checks that a commit message is non-empty and written in English.
func ValidateCommitMessage(msg string) error {
	trimmed := strings.TrimSpace(msg)
	if trimmed == "" {
		return fmt.Errorf("commit message cannot be empty")
	}

	firstLine := strings.Split(trimmed, "\n")[0]

	// 1. Direct Umlaut Check
	if umlautRegex.MatchString(firstLine) {
		return fmt.Errorf("commit message contains German umlauts/characters (%s). All commit messages must be in English (AGENTS.md)", firstLine)
	}

	// 2. Tokenize words and check against German indicators
	words := strings.Fields(firstLine)
	germanHits := 0
	for _, w := range words {
		cleaned := strings.ToLower(strings.Trim(w, ":,().'\"`[]{}!?-"))
		if germanStopWords[cleaned] {
			germanHits++
		}
	}

	if germanHits > 0 {
		return fmt.Errorf("commit message appears to be in German (%d German keyword(s) detected). All commit messages must be in English (AGENTS.md)", germanHits)
	}

	return nil
}
