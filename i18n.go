package main

import (
	"fmt"
	"sync/atomic"
)

const defaultLanguage = "en"

// languages lists the supported UI languages, each shown by its own name in the tray menu.
var languages = []struct{ code, name string }{
	{"en", "English"},
	{"pt-BR", "Português (Brasil)"},
}

var messages = map[string]map[string]string{
	"en": {
		"windowTitle":   "Claude usage",
		"header":        "Your usage limits",
		"session":       "Current session",
		"weekly":        "Weekly · all models",
		"weeklyOpus":    "Weekly · Opus",
		"weeklySonnet":  "Weekly · Sonnet",
		"loading":       "Loading…",
		"refreshing":    "Refreshing…",
		"updatedAt":     "Updated at %s",
		"used":          "%s used",
		"resetting":     "Resetting…",
		"resetsD":       "Resets in %d d",
		"resetsDH":      "Resets in %d d %d h",
		"resetsHM":      "Resets in %d h %d min",
		"resetsM":       "Resets in %d min",
		"planTeam":      "Team",
		"errNoCreds":    "credentials not found",
		"errBadCreds":   "invalid credentials",
		"errLogin":      "log in to Claude Code",
		"errExpired":    "token expired: open Claude Code",
		"errOffline":    "no connection",
		"errRateLimit":  "rate limited (429)",
		"errHTTP":       "HTTP error %d",
		"errBadReply":   "invalid response",
		"errAutostart":  "Could not change start with Windows",
		"menuRefresh":   "Refresh now",
		"menuPassThru":  "Click-through (ignore clicks)",
		"menuHide":      "Hide",
		"menuShow":      "Show",
		"menuResetPos":  "Reset position",
		"menuAutostart": "Start with Windows",
		"menuLanguage":  "Language",
		"menuQuit":      "Quit",
		"cliError":      "error:",
		"cliPlan":       "plan:",
		"cliSession":    "session",
		"cliWeekly":     "weekly",
	},
	"pt-BR": {
		"windowTitle":   "Uso do Claude",
		"header":        "Seus limites de uso",
		"session":       "Sessão atual",
		"weekly":        "Semanal · todos os modelos",
		"weeklyOpus":    "Semanal · Opus",
		"weeklySonnet":  "Semanal · Sonnet",
		"loading":       "Carregando…",
		"refreshing":    "Atualizando…",
		"updatedAt":     "Atualizado às %s",
		"used":          "%s usado",
		"resetting":     "Reiniciando…",
		"resetsD":       "Reinicia em %d d",
		"resetsDH":      "Reinicia em %d d %d h",
		"resetsHM":      "Reinicia em %d h %d min",
		"resetsM":       "Reinicia em %d min",
		"planTeam":      "Equipe",
		"errNoCreds":    "credenciais não encontradas",
		"errBadCreds":   "credenciais inválidas",
		"errLogin":      "faça login no Claude Code",
		"errExpired":    "token expirado: abra o Claude Code",
		"errOffline":    "sem conexão",
		"errRateLimit":  "limite de requisições (429)",
		"errHTTP":       "erro HTTP %d",
		"errBadReply":   "resposta inválida",
		"errAutostart":  "Erro ao configurar início automático",
		"menuRefresh":   "Atualizar agora",
		"menuPassThru":  "Ignorar cliques (atravessar)",
		"menuHide":      "Ocultar",
		"menuShow":      "Mostrar",
		"menuResetPos":  "Restaurar posição",
		"menuAutostart": "Iniciar com o Windows",
		"menuLanguage":  "Idioma",
		"menuQuit":      "Sair",
		"cliError":      "erro:",
		"cliPlan":       "plano:",
		"cliSession":    "sessão",
		"cliWeekly":     "semanal",
	},
}

// currentLanguage is read by the fetch goroutine (error messages), hence atomic.
var currentLanguage atomic.Value

// setLanguage switches the UI language; unknown codes fall back to English.
func setLanguage(code string) {
	if _, ok := messages[code]; !ok {
		code = defaultLanguage
	}
	currentLanguage.Store(code)
}

func languageCode() string {
	if c, ok := currentLanguage.Load().(string); ok {
		return c
	}
	return defaultLanguage
}

// T returns the message for key in the current language (English if missing).
func T(key string, args ...any) string {
	s, ok := messages[languageCode()][key]
	if !ok {
		s = messages[defaultLanguage][key]
	}
	if len(args) > 0 {
		return fmt.Sprintf(s, args...)
	}
	return s
}

// usageError is translated when displayed, so it follows a language change.
type usageError struct {
	key  string
	args []any
}

func (e usageError) Error() string { return T(e.key, e.args...) }
